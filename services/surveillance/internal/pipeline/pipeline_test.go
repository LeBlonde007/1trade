package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/trade1/surveillance/internal/detect"
)

// memStore is an in-memory Store keyed by alert id.
type memStore struct {
	mu   sync.Mutex
	rows map[string]detect.Alert
}

// SaveAlert stores once and reports whether the alert was new.
func (m *memStore) SaveAlert(_ context.Context, a detect.Alert) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rows == nil {
		m.rows = map[string]detect.Alert{}
	}
	if _, ok := m.rows[a.AlertID]; ok {
		return false, nil
	}
	m.rows[a.AlertID] = a
	return true, nil
}

// memPub records published alerts.
type memPub struct {
	mu  sync.Mutex
	ids []string
}

// PublishAlert records the alert id.
func (m *memPub) PublishAlert(a detect.Alert) error {
	m.mu.Lock()
	m.ids = append(m.ids, a.AlertID)
	m.mu.Unlock()
	return nil
}

// roundTrip is an exact wash round trip as the two engine payloads.
func roundTrip() [][2]string {
	t0 := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	mk := func(id, buyer, seller string, at time.Time) [2]string {
		b, _ := json.Marshal(detect.Trade{TradeID: id, ProductID: "EAI-IDX", Price: "0.001000", Quantity: "5000.000000",
			Buyer: detect.Party{TenantID: buyer}, Seller: detect.Party{TenantID: seller}, AggressorSide: "buy", IsPaper: true, ExecutedAt: at})
		return [2]string{SubjectTrades, string(b)}
	}
	return [][2]string{mk("t1", "A", "B", t0), mk("t2", "B", "A", t0.Add(time.Minute))}
}

// TestHandleStoresThenPublishesOnce: a new alert is stored and published; replaying the same stream
// into a fresh pipeline (a restart) against the same store publishes nothing.
func TestHandleStoresThenPublishesOnce(t *testing.T) {
	st, pub := &memStore{}, &memPub{}
	feed := func(p *Pipeline) {
		for _, m := range roundTrip() {
			if _, err := p.Handle(context.Background(), m[0], []byte(m[1])); err != nil {
				t.Fatal(err)
			}
		}
	}
	feed(New(detect.DefaultConfig(), st, pub))
	if len(pub.ids) != 1 || len(st.rows) != 1 {
		t.Fatalf("first pass: published %d, stored %d; want 1 and 1", len(pub.ids), len(st.rows))
	}
	feed(New(detect.DefaultConfig(), st, pub)) // restart: replay from the beginning
	if len(pub.ids) != 1 {
		t.Errorf("restart re-published: %v", pub.ids)
	}
}

// TestPoisonMessageIsSkipped: an undecodable payload is an error for that message only.
func TestPoisonMessageIsSkipped(t *testing.T) {
	p := New(detect.DefaultConfig(), &memStore{}, &memPub{})
	if _, err := p.Handle(context.Background(), SubjectTrades, []byte("{not json")); err == nil {
		t.Fatal("poison message accepted")
	}
	if n, err := p.Handle(context.Background(), "unrelated.subject", []byte("x")); err != nil || n != 0 {
		t.Fatalf("unrelated subject: %d %v", n, err)
	}
}

// startJetStream runs an embedded NATS server with JetStream for the test.
func startJetStream(t *testing.T) string {
	t.Helper()
	s, err := natsserver.NewServer(&natsserver.Options{Port: -1, JetStream: true, StoreDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	go s.Start()
	if !s.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats not ready")
	}
	t.Cleanup(s.Shutdown)
	return s.ClientURL()
}

// TestEndToEndOverJetStream proves the real transport: events published BEFORE surveillance starts are
// replayed, later events are followed live, the alert arrives on surveillance.alert.v1 exactly once,
// and a restart replays silently.
func TestEndToEndOverJetStream(t *testing.T) {
	url := startJetStream(t)
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	js, _ := nc.JetStream()
	if _, err := js.AddStream(&nats.StreamConfig{Name: Stream, Subjects: []string{SubjectOrders, SubjectTrades}}); err != nil {
		t.Fatal(err)
	}
	alerts := make(chan *nats.Msg, 16)
	if _, err := nc.ChanSubscribe(SubjectAlerts, alerts); err != nil {
		t.Fatal(err)
	}

	msgs := roundTrip()
	if _, err := js.Publish(msgs[0][0], []byte(msgs[0][1])); err != nil { // before surveillance starts
		t.Fatal(err)
	}
	st := &memStore{}
	start := func() *Runner {
		r, err := Start(url, func(pub Publisher) *Pipeline { return New(detect.DefaultConfig(), st, pub) })
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	r := start()
	if _, err := js.Publish(msgs[1][0], []byte(msgs[1][1])); err != nil { // live
		t.Fatal(err)
	}
	select {
	case m := <-alerts:
		var a detect.Alert
		if err := json.Unmarshal(m.Data, &a); err != nil || a.Rule != detect.RuleWash || len(a.TradeIDs) != 2 {
			t.Fatalf("alert = %s (%v)", m.Data, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no alert published")
	}
	r.Close()

	r2 := start() // restart replays both trades
	defer r2.Close()
	select {
	case m := <-alerts:
		t.Fatalf("restart re-published an alert: %s", m.Data)
	case <-time.After(1500 * time.Millisecond):
	}
	if len(st.rows) != 1 {
		t.Errorf("stored %d alerts, want 1", len(st.rows))
	}
}

// TestOrderingAcrossSubjects: a spoof depends on an order, a trade, then a cancel — across both
// subjects. One stream keeps them in emission order, so the detector sees the pattern.
func TestOrderingAcrossSubjects(t *testing.T) {
	url := startJetStream(t)
	nc, _ := nats.Connect(url)
	defer nc.Close()
	js, _ := nc.JetStream()
	_, _ = js.AddStream(&nats.StreamConfig{Name: Stream, Subjects: []string{SubjectOrders, SubjectTrades}})
	alerts := make(chan *nats.Msg, 16)
	_, _ = nc.ChanSubscribe(SubjectAlerts, alerts)

	t0 := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	pubOrder := func(o detect.Order) { b, _ := json.Marshal(o); _, _ = js.Publish(SubjectOrders, b) }
	px := "3.10"
	for i := range 5 {
		pubOrder(detect.Order{EventID: fmt.Sprint("bg", i), OrderID: fmt.Sprint("bg", i), TenantID: fmt.Sprint("bgt", i), ProductID: "H100-SPOT",
			Side: "buy", State: "accepted", Quantity: "10", FilledQuantity: "0", LimitPrice: &px, IsPaper: true, TS: t0})
	}
	pubOrder(detect.Order{EventID: "e1", OrderID: "big", TenantID: "S", ProductID: "H100-SPOT", Side: "sell", State: "accepted",
		Quantity: "500", FilledQuantity: "0", LimitPrice: &px, IsPaper: true, TS: t0.Add(time.Second)})
	tb, _ := json.Marshal(detect.Trade{TradeID: "x1", ProductID: "H100-SPOT", Price: "3.00", Quantity: "20",
		Buyer: detect.Party{TenantID: "S"}, Seller: detect.Party{TenantID: "M"}, AggressorSide: "buy", IsPaper: true, ExecutedAt: t0.Add(2 * time.Second)})
	_, _ = js.Publish(SubjectTrades, tb)
	reason := "user_cancel"
	pubOrder(detect.Order{EventID: "e2", OrderID: "big", TenantID: "S", State: "cancelled", Reason: &reason,
		FilledQuantity: "0", IsPaper: true, TS: t0.Add(4 * time.Second)})

	r, err := Start(url, func(pub Publisher) *Pipeline { return New(detect.DefaultConfig(), &memStore{}, pub) })
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	select {
	case m := <-alerts:
		var a detect.Alert
		_ = json.Unmarshal(m.Data, &a)
		if a.Rule != detect.RuleSpoofing {
			t.Fatalf("alert rule = %s", a.Rule)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("spoof across subjects not detected")
	}
}
