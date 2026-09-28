package settle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/trade1/matching-engine/internal/engine"
)

// httpLedger is an in-memory credit-ledger speaking the v1.2 reservation API over HTTP. It is faithful
// where the reconciler's safety depends on it: reserve is idempotent on order_id and a replay answers
// with the reservation's CURRENT state (a released one comes back as released), and release is
// idempotent and 404s when nothing was reserved. Settlement is accepted and ignored.
type httpLedger struct {
	*httptest.Server
	mu  sync.Mutex
	res map[string]*heldRes
	// commitThenFail makes that many upcoming reserves commit and then answer 500, like a timeout
	// that fires after the ledger's commit. failRelease makes that many releases answer 503 unapplied.
	commitThenFail int
	failRelease    int
}

// heldRes is one reservation.
type heldRes struct {
	tenant, kind, asset, amount string
	open                        bool
}

// newHTTPLedger starts a fake ledger that closes with the test.
func newHTTPLedger(t *testing.T) *httpLedger {
	t.Helper()
	f := &httpLedger{res: map[string]*heldRes{}}
	f.Server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.Close)
	return f
}

// serve implements reserve, release and settle-trade.
func (f *httpLedger) serve(w http.ResponseWriter, r *http.Request) {
	var b struct {
		OrderID   string `json:"order_id"`
		TenantID  string `json:"tenant_id"`
		AssetKind string `json:"asset_kind"`
		Asset     string `json:"asset"`
		Amount    string `json:"amount"`
	}
	_ = json.NewDecoder(r.Body).Decode(&b)
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.URL.Path {
	case "/v1/credits/reserve":
		if x, ok := f.res[b.OrderID]; ok {
			if x.tenant != b.TenantID || x.kind != b.AssetKind || x.asset != b.Asset || x.amount != b.Amount {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"code":"IDEMPOTENCY_CONFLICT"}`))
				return
			}
			writeHeldRes(w, b.OrderID, x)
			return
		}
		x := &heldRes{tenant: b.TenantID, kind: b.AssetKind, asset: b.Asset, amount: b.Amount, open: true}
		f.res[b.OrderID] = x
		if f.commitThenFail > 0 {
			f.commitThenFail--
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		writeHeldRes(w, b.OrderID, x)
	case "/v1/credits/release":
		if f.failRelease > 0 {
			f.failRelease--
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		x, ok := f.res[b.OrderID]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"code":"not_found"}`))
			return
		}
		x.open = false
		writeHeldRes(w, b.OrderID, x)
	case "/v1/credits/settle-trade":
		_, _ = w.Write([]byte(`{}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// writeHeldRes renders a reservation in its contract shape.
func writeHeldRes(w http.ResponseWriter, id string, x *heldRes) {
	state, remaining := "open", x.amount
	if !x.open {
		state, remaining = "released", "0.000000"
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"order_id": id, "tenant_id": x.tenant, "sub_account_id": nil,
		"asset_kind": x.kind, "asset": x.asset, "amount": x.amount, "remaining": remaining, "state": state, "is_paper": true})
}

// isOpen reports whether an order_id has an open reservation.
func (f *httpLedger) isOpen(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	x, ok := f.res[id]
	return ok && x.open
}

// openIDs returns every order_id with an open reservation.
func (f *httpLedger) openIDs() map[string]bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]bool{}
	for id, x := range f.res {
		if x.open {
			out[id] = true
		}
	}
	return out
}

// rig is an engine wired the way cutover wires it: ReserveRisk against a ledger, a journal that can be
// taken down, and a reconciler fed by OnUnjournaled. Its clock is settable.
type rig struct {
	e    *engine.Engine
	rec  *Reconciler
	led  *httpLedger
	c    *Client
	down atomic.Bool

	mu  sync.Mutex
	now time.Time
}

// newRig builds a rig with a one-minute grace period.
func newRig(t *testing.T) *rig {
	t.Helper()
	rg := &rig{led: newHTTPLedger(t), now: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)}
	rg.c = New(rg.led.URL, "engine-token", time.Second)
	rg.rec = &Reconciler{Ledger: rg.c, Grace: time.Minute, Now: rg.clock}
	rg.e = engine.New(engine.Config{
		Epoch: "reconcile-test",
		Risk:  ReserveRisk(rg.c, time.Second),
		Persist: func(uint64, engine.Command) error {
			if rg.down.Load() {
				return errors.New("journal database unreachable")
			}
			return nil
		},
		OnUnjournaled: rg.rec.Suspect,
	})
	rg.rec.Engine = rg.e
	return rg
}

// clock reads the rig's time.
func (rg *rig) clock() time.Time {
	rg.mu.Lock()
	defer rg.mu.Unlock()
	return rg.now
}

// advance moves the rig's time forward.
func (rg *rig) advance(d time.Duration) {
	rg.mu.Lock()
	rg.now = rg.now.Add(d)
	rg.mu.Unlock()
}

// sell is a paper limit sell of qty H100 @ 2.99 (it holds qty credits).
func sell(id, tenant, qty string, ts time.Time) engine.SubmitCmd {
	return engine.SubmitCmd{OrderID: id, TenantID: tenant, ProductID: "H100-SPOT", Side: engine.Sell, Type: engine.Limit,
		Price: engine.MustFixed("2.99"), Quantity: engine.MustFixed(qty), IsPaper: true, TS: ts}
}

// orphan submits id while the journal is down: the ledger reserves, the journal refuses.
func (rg *rig) orphan(t *testing.T, id, tenant string) {
	t.Helper()
	rg.down.Store(true)
	if _, err := rg.e.Submit(sell(id, tenant, "1", rg.clock())); !errors.Is(err, engine.ErrJournal) {
		t.Fatalf("submit with the journal down = %v, want ErrJournal", err)
	}
	rg.down.Store(false)
	if !rg.led.isOpen(id) {
		t.Fatal("setup: the ledger holds no reservation for the unjournaled order")
	}
}

// TestReconcilerVoidsAndReleasesAnOrphan: after the grace period the id is tombstoned, the reservation
// is released, and a late retry of the id is refused without reserving again.
func TestReconcilerVoidsAndReleasesAnOrphan(t *testing.T) {
	rg := newRig(t)
	rg.orphan(t, "o1", "t1")
	if rep := rg.rec.RunOnce(context.Background()); rep != (Report{Waiting: 1}) || !rg.led.isOpen("o1") {
		t.Fatalf("inside the grace period: %+v, open=%v", rep, rg.led.isOpen("o1"))
	}
	rg.advance(time.Minute)
	if rep := rg.rec.RunOnce(context.Background()); rep != (Report{Voided: 1, Released: 1}) {
		t.Fatalf("after grace: %+v", rep)
	}
	if rg.led.isOpen("o1") || rg.rec.Pending() != 0 {
		t.Fatalf("reservation open=%v, pending=%d", rg.led.isOpen("o1"), rg.rec.Pending())
	}
	j := rg.e.Journal()
	if len(j) != 1 || j[0].Void == nil || j[0].Void.OrderID != "o1" || !j[0].Void.IsPaper {
		t.Fatalf("journal = %+v, want exactly the void", j)
	}
	r, err := rg.e.Submit(sell("o1", "t1", "1", rg.clock()))
	if err != nil || !r.Duplicate || r.Order.Reason != engine.ReasonVoided || rg.led.isOpen("o1") {
		t.Fatalf("late retry = %+v, %v (reservation open=%v)", r, err, rg.led.isOpen("o1"))
	}
}

// TestSuspectKeepsTheFirstReport: a repeat report does not restart the grace period.
func TestSuspectKeepsTheFirstReport(t *testing.T) {
	rg := newRig(t)
	rg.orphan(t, "o1", "t1")
	rg.advance(59 * time.Second)
	rg.rec.Suspect(engine.Order{OrderID: "o1", TenantID: "t1", IsPaper: true})
	rg.advance(time.Second)
	if rep := rg.rec.RunOnce(context.Background()); rep != (Report{Voided: 1, Released: 1}) {
		t.Fatalf("a repeat report restarted the grace period: %+v", rep)
	}
}

// TestReconcilerKeepsAReservationARetryReused: a retry inside the grace period reuses the reservation,
// and the reconciler must leave it backing the live order.
func TestReconcilerKeepsAReservationARetryReused(t *testing.T) {
	rg := newRig(t)
	rg.orphan(t, "o1", "t1")
	if r, err := rg.e.Submit(sell("o1", "t1", "1", rg.clock())); err != nil || r.Duplicate || r.Order.State != engine.Accepted {
		t.Fatalf("prompt retry = %+v, %v", r, err)
	}
	rg.advance(time.Hour)
	if rep := rg.rec.RunOnce(context.Background()); rep != (Report{Kept: 1}) {
		t.Fatalf("report = %+v", rep)
	}
	if !rg.led.isOpen("o1") {
		t.Fatal("the reconciler released the reservation of a live order")
	}
	if o, _ := rg.e.Order("o1", "t1"); o.State != engine.Accepted {
		t.Fatalf("order = %+v", o)
	}
}

// TestReconcilerWaitsOutAJournalOutage: a void needs the journal; while it is down nothing is released.
func TestReconcilerWaitsOutAJournalOutage(t *testing.T) {
	rg := newRig(t)
	rg.orphan(t, "o1", "t1")
	rg.advance(time.Minute)
	rg.down.Store(true)
	if rep := rg.rec.RunOnce(context.Background()); rep != (Report{Failed: 1}) || !rg.led.isOpen("o1") || rg.rec.Pending() != 1 {
		t.Fatalf("with the journal down: %+v, open=%v, pending=%d", rep, rg.led.isOpen("o1"), rg.rec.Pending())
	}
	rg.down.Store(false)
	if rep := rg.rec.RunOnce(context.Background()); rep != (Report{Voided: 1, Released: 1}) || rg.led.isOpen("o1") {
		t.Fatalf("after recovery: %+v, open=%v", rep, rg.led.isOpen("o1"))
	}
}

// TestReconcilerRetriesAFailedRelease: the tombstone stays; the release is retried until it lands.
func TestReconcilerRetriesAFailedRelease(t *testing.T) {
	rg := newRig(t)
	rg.orphan(t, "o1", "t1")
	rg.advance(time.Minute)
	rg.led.mu.Lock()
	rg.led.failRelease = 1
	rg.led.mu.Unlock()
	if rep := rg.rec.RunOnce(context.Background()); rep != (Report{Voided: 1, Failed: 1}) || !rg.led.isOpen("o1") {
		t.Fatalf("first pass: %+v, open=%v", rep, rg.led.isOpen("o1"))
	}
	if r, _ := rg.e.Submit(sell("o1", "t1", "1", rg.clock())); r.Order.Reason != engine.ReasonVoided {
		t.Fatalf("a retry between the void and the release claimed the reservation: %+v", r)
	}
	if rep := rg.rec.RunOnce(context.Background()); rep != (Report{Released: 1}) || rg.led.isOpen("o1") {
		t.Fatalf("second pass: %+v, open=%v", rep, rg.led.isOpen("o1"))
	}
	if n := len(rg.e.Journal()); n != 1 {
		t.Fatalf("journal has %d commands; the retried void must not be journaled twice", n)
	}
}

// TestReconcilerReleasesARejectedRetry: a retry that is rejected at entry closes the id for good, so
// the orphaned reservation is released without a void.
func TestReconcilerReleasesARejectedRetry(t *testing.T) {
	rg := newRig(t)
	rg.orphan(t, "o1", "t1")
	// Same id, different quantity: the ledger refuses the body (409) and the order is rejected.
	if r, err := rg.e.Submit(sell("o1", "t1", "2", rg.clock())); err != nil || r.Order.State != engine.Rejected || r.Order.Reason != ReasonRiskUnavailable {
		t.Fatalf("conflicting retry = %+v, %v", r, err)
	}
	rg.advance(time.Minute)
	if rep := rg.rec.RunOnce(context.Background()); rep != (Report{Released: 1}) || rg.led.isOpen("o1") {
		t.Fatalf("report = %+v, open=%v", rep, rg.led.isOpen("o1"))
	}
}

// TestReconcilerLeavesAnotherTenantsIDAlone: if another tenant's order holds the id, its reservation
// may back that order, so the reconciler must never release it.
func TestReconcilerLeavesAnotherTenantsIDAlone(t *testing.T) {
	rg := newRig(t)
	rg.rec.Suspect(engine.Order{OrderID: "o1", TenantID: "t1", IsPaper: true})
	if r, err := rg.e.Submit(sell("o1", "t2", "1", rg.clock())); err != nil || r.Order.State != engine.Accepted {
		t.Fatalf("t2's order = %+v, %v", r, err)
	}
	rg.advance(time.Minute)
	if rep := rg.rec.RunOnce(context.Background()); rep != (Report{Kept: 1}) || !rg.led.isOpen("o1") || rg.rec.Pending() != 0 {
		t.Fatalf("report = %+v, open=%v, pending=%d", rep, rg.led.isOpen("o1"), rg.rec.Pending())
	}
}

// TestReplayOfAClosedReservationIsRefused: the engine must not accept an order on a replayed reservation
// that no longer locks anything (the ledger answers such a replay with 200 and its current state).
func TestReplayOfAClosedReservationIsRefused(t *testing.T) {
	rg := newRig(t)
	o := engine.Order{OrderID: "o1", TenantID: "t1", IsPaper: true}
	h := engine.Hold{Kind: "credit", Asset: "gpu_h100", Amount: engine.MustFixed("1")}
	if err := rg.c.Reserve(context.Background(), o, h); err != nil {
		t.Fatal(err)
	}
	if err := rg.c.Release(context.Background(), "o1"); err != nil {
		t.Fatal(err)
	}
	if err := rg.c.Reserve(context.Background(), o, h); !errors.Is(err, ErrReservationUnusable) {
		t.Fatalf("reserve replay of a released reservation = %v, want ErrReservationUnusable", err)
	}
	r, err := rg.e.Submit(sell("o1", "t1", "1", rg.clock()))
	if err != nil || r.Order.State != engine.Rejected || r.Order.Reason != ReasonOrderIDReused {
		t.Fatalf("order on a closed reservation = %+v, %v; want rejected/order_id_reused", r, err)
	}
}

// TestCheckReservation pins every way a 200 reserve answer can fail to lock the whole hold.
func TestCheckReservation(t *testing.T) {
	hold := engine.MustFixed("10")
	body := func(id, state, amount, remaining string) []byte {
		return []byte(fmt.Sprintf(`{"order_id":%q,"state":%q,"amount":%q,"remaining":%q}`, id, state, amount, remaining))
	}
	cases := []struct {
		name     string
		body     []byte
		unusable bool // want ErrReservationUnusable
		ok       bool
	}{
		{"open, whole hold", body("o1", "open", "10.000000", "10.000000"), false, true},
		{"uuid case differs", body("O1", "open", "10.000000", "10.000000"), false, true},
		{"released", body("o1", "released", "10.000000", "0.000000"), true, false},
		{"partly consumed", body("o1", "open", "10.000000", "4.000000"), true, false},
		{"different amount", body("o1", "open", "9.000000", "9.000000"), false, false},
		{"different order", body("o2", "open", "10.000000", "10.000000"), false, false},
		{"unparsable amount", body("o1", "open", "ten", "10.000000"), false, false},
		{"not json", []byte("<html>"), false, false},
		{"empty", nil, false, false},
	}
	for _, tc := range cases {
		err := checkReservation(tc.body, "o1", hold)
		switch {
		case tc.ok && err != nil:
			t.Errorf("%s: %v, want nil", tc.name, err)
		case !tc.ok && err == nil:
			t.Errorf("%s: accepted", tc.name)
		case !tc.ok && errors.Is(err, ErrReservationUnusable) != tc.unusable:
			t.Errorf("%s: %v, unusable=%v", tc.name, err, tc.unusable)
		}
	}
}

// TestReconcilerStress races clients (with retries), cancels, a flapping journal, reserves that commit
// and then fail, and a reconciler looping with a tiny grace period. Afterwards the journal is replayed
// into the settlement worker, exactly as the relay feeds it, and the ledger must hold an open
// reservation for every live order and for nothing else: no order is ever unfunded, and no orphan is
// left behind. Run with -race; a deadlock between Suspect and Void hangs it.
func TestReconcilerStress(t *testing.T) {
	for seed := int64(1); seed <= 5; seed++ {
		t.Run(fmt.Sprint("seed ", seed), func(t *testing.T) { stress(t, seed) })
	}
}

// stress runs one seeded stress scenario.
func stress(t *testing.T, seed int64) {
	led := newHTTPLedger(t)
	c := New(led.URL, "engine-token", time.Second)
	var rmu sync.Mutex
	rng := rand.New(rand.NewSource(seed)) // #nosec G404 -- deterministic test input
	chance := func(n int) bool { rmu.Lock(); defer rmu.Unlock(); return rng.Intn(n) == 0 }
	var flaky atomic.Bool
	flaky.Store(true)
	rec := &Reconciler{Ledger: c, Grace: 3 * time.Millisecond}
	e := engine.New(engine.Config{
		Epoch: "stress",
		Risk:  ReserveRisk(c, time.Second),
		Persist: func(uint64, engine.Command) error {
			if flaky.Load() && chance(4) {
				return errors.New("journal write failed")
			}
			return nil
		},
		OnUnjournaled: rec.Suspect,
	})
	rec.Engine = e

	ctx, stop := context.WithCancel(context.Background())
	recDone := make(chan struct{})
	go func() {
		defer close(recDone)
		for ctx.Err() == nil {
			rec.RunOnce(ctx)
			time.Sleep(time.Millisecond)
		}
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		for w := range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				tenant := fmt.Sprintf("t%d", w)
				for i := range 60 {
					id := fmt.Sprintf("s%d-%d-%d", seed, w, i)
					if chance(8) {
						led.mu.Lock()
						led.commitThenFail++
						led.mu.Unlock()
					}
					side, price := engine.Sell, "3.05"
					if w%2 == 0 {
						side, price = engine.Buy, "2.95"
					}
					if chance(3) { // some orders cross and trade
						price = "3.00"
					}
					cmd := engine.SubmitCmd{OrderID: id, TenantID: tenant, ProductID: "H100-SPOT", Side: side, Type: engine.Limit,
						Price: engine.MustFixed(price), Quantity: engine.MustFixed("1"), IsPaper: true, TS: time.Now()}
					for attempt := 0; attempt < 3; attempt++ { // a client retrying a failed submit
						if _, err := e.Submit(cmd); !errors.Is(err, engine.ErrJournal) {
							break
						}
						time.Sleep(time.Duration(attempt*2) * time.Millisecond)
					}
					if chance(4) {
						_, _ = e.Cancel(engine.CancelCmd{OrderID: id, TenantID: tenant, TS: time.Now()})
					}
				}
			}()
		}
		wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("stress run hung (deadlock between the engine and the reconciler?)")
	}

	// Quiesce: healthy journal, zero grace, drain every suspect.
	flaky.Store(false)
	stop()
	<-recDone
	rec.Grace, rec.Now = time.Nanosecond, func() time.Time { return time.Now().Add(time.Hour) }
	for i := 0; rec.Pending() > 0; i++ {
		if i > 10 {
			t.Fatalf("%d suspects never resolved", rec.Pending())
		}
		rec.RunOnce(context.Background())
	}
	// The settlement worker applies the journal's events in emission order, as the relay feeds it.
	_, evs, err := engine.Replay(engine.Config{Epoch: "stress"}, e.Journal())
	if err != nil {
		t.Fatal(err)
	}
	if n, err := (&Worker{Ledger: c, Attempts: 1}).Process(context.Background(), evs); err != nil || n != len(evs) {
		t.Fatalf("worker = %d/%d, %v", n, len(evs), err)
	}

	open := led.openIDs()
	live, voided := 0, 0
	for _, cmd := range e.Journal() {
		var id, tenant string
		switch {
		case cmd.Submit != nil:
			id, tenant = cmd.Submit.OrderID, cmd.Submit.TenantID
		case cmd.Void != nil:
			id, tenant = cmd.Void.OrderID, cmd.Void.TenantID
			voided++
		default:
			continue
		}
		o, _ := e.Order(id, tenant)
		switch {
		case o.Open() && !open[id]:
			t.Fatalf("live order %s has no open reservation: it is unfunded", id)
		case !o.Open() && open[id]:
			t.Fatalf("%s order %s (%s) still holds an open reservation", o.State, id, o.Reason)
		}
		if o.Open() {
			live++
		}
		delete(open, id)
	}
	for id := range open {
		t.Fatalf("orphaned reservation %s: no order in the journal", id)
	}
	if live == 0 || voided == 0 {
		t.Fatalf("scenario too tame: %d live orders, %d voids", live, voided)
	}
	t.Logf("seed %d: %d journal entries, %d live, %d voided, %d events", seed, len(e.Journal()), live, voided, len(evs))
}

// Compile-time checks that the real types satisfy the reconciler's interfaces.
var (
	_ Voider   = (*engine.Engine)(nil)
	_ Releaser = (*Client)(nil)
)

// TestReconcilerAgainstRealLedger drives the orphan path through the real credit-ledger (runs when
// LEDGER_E2E_URL, LEDGER_SERVICE_TOKEN and LEDGER_SETTLE_TOKEN point at a running ledger in dev mode):
//   - an orphaned reservation is voided and released, so the seller's credits unlock;
//   - a late retry of the voided id reserves nothing;
//   - a prompt retry keeps its reservation;
//   - the real ledger's 200 replay of a released reservation is refused by the client.
func TestReconcilerAgainstRealLedger(t *testing.T) {
	base, svc, settleTok := os.Getenv("LEDGER_E2E_URL"), os.Getenv("LEDGER_SERVICE_TOKEN"), os.Getenv("LEDGER_SETTLE_TOKEN")
	if base == "" || svc == "" || settleTok == "" {
		t.Skip("LEDGER_E2E_URL / LEDGER_SERVICE_TOKEN / LEDGER_SETTLE_TOKEN not set")
	}
	stamp := time.Now().UnixNano() % 1_000_000_000_000
	id := func(k int) string { return fmt.Sprintf("00000000-0000-4000-%04d-%012d", 9500+k, stamp) }
	seller := id(1)
	call := func(method, path string, headers map[string]string, body any, out any) int {
		t.Helper()
		var rd *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rd = bytes.NewReader(b)
		} else {
			rd = bytes.NewReader(nil)
		}
		req, _ := http.NewRequest(method, base+path, rd)
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if out != nil {
			_ = json.NewDecoder(resp.Body).Decode(out)
		}
		return resp.StatusCode
	}
	if code := call(http.MethodPost, "/v1/credits/purchase", map[string]string{"Authorization": "Bearer " + svc, "Idempotency-Key": "seed:" + seller},
		map[string]any{"tenant_id": seller, "credit_type": "gpu_h100", "amount": "10", "reference_id": "seed:" + seller, "is_paper": true}, nil); code != 200 {
		t.Fatalf("fund seller: %d", code)
	}
	locked := func() string {
		t.Helper()
		var out struct {
			Balances []map[string]any `json:"balances"`
		}
		call(http.MethodGet, "/v1/credits/balances", map[string]string{"X-Dev-Tenant": seller}, nil, &out)
		for _, b := range out.Balances {
			if b["credit_type"] == "gpu_h100" {
				return b["locked_amount"].(string)
			}
		}
		return "none"
	}

	c := New(base, settleTok, 5*time.Second)
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	var down atomic.Bool
	rec := &Reconciler{Ledger: c, Grace: time.Minute, Now: func() time.Time { return now }}
	e := engine.New(engine.Config{Epoch: fmt.Sprint("reconcile-e2e-", stamp), Risk: ReserveRisk(c, 5*time.Second),
		Persist: func(uint64, engine.Command) error {
			if down.Load() {
				return errors.New("journal database unreachable")
			}
			return nil
		},
		OnUnjournaled: rec.Suspect})
	rec.Engine = e

	// An orphan: the ledger locks 3 credits for an order the journal never took.
	x := id(10)
	down.Store(true)
	if _, err := e.Submit(sell(x, seller, "3", now)); !errors.Is(err, engine.ErrJournal) {
		t.Fatal(err)
	}
	down.Store(false)
	if got := locked(); got != "3.000000" {
		t.Fatalf("locked after the orphaned reserve = %s, want 3", got)
	}
	if rep := rec.RunOnce(context.Background()); rep != (Report{Waiting: 1}) || locked() != "3.000000" {
		t.Fatalf("inside grace: %+v, locked %s", rep, locked())
	}
	now = now.Add(time.Minute)
	if rep := rec.RunOnce(context.Background()); rep != (Report{Voided: 1, Released: 1}) {
		t.Fatalf("after grace: %+v", rep)
	}
	if got := locked(); got != "0.000000" {
		t.Fatalf("locked after reconciliation = %s, want 0", got)
	}
	if r, err := e.Submit(sell(x, seller, "3", now)); err != nil || r.Order.Reason != engine.ReasonVoided || locked() != "0.000000" {
		t.Fatalf("late retry = %+v, %v; locked %s", r, err, locked())
	}

	// The real ledger answers a replay of the released reservation with 200 and its current state;
	// the client must refuse it rather than accept an unfunded order.
	h := engine.Hold{Kind: "credit", Asset: "gpu_h100", Amount: engine.MustFixed("3")}
	if err := c.Reserve(context.Background(), engine.Order{OrderID: x, TenantID: seller, IsPaper: true}, h); !errors.Is(err, ErrReservationUnusable) {
		t.Fatalf("replay of a released reservation = %v, want ErrReservationUnusable", err)
	}

	// A prompt retry reuses its reservation, and the reconciler leaves it backing the live order.
	y := id(11)
	down.Store(true)
	if _, err := e.Submit(sell(y, seller, "2", now)); !errors.Is(err, engine.ErrJournal) {
		t.Fatal(err)
	}
	down.Store(false)
	if r, err := e.Submit(sell(y, seller, "2", now)); err != nil || r.Order.State != engine.Accepted {
		t.Fatalf("prompt retry = %+v, %v", r, err)
	}
	now = now.Add(time.Minute)
	if rep := rec.RunOnce(context.Background()); rep != (Report{Kept: 1}) || locked() != "2.000000" {
		t.Fatalf("retried order: %+v, locked %s (want 2 still reserved)", rep, locked())
	}

	// Close it the normal way: cancel, and the worker releases on the terminal event.
	if _, err := e.Cancel(engine.CancelCmd{OrderID: y, TenantID: seller, TS: now}); err != nil {
		t.Fatal(err)
	}
	_, evs, err := engine.Replay(engine.Config{Epoch: fmt.Sprint("reconcile-e2e-", stamp)}, e.Journal())
	if err != nil {
		t.Fatal(err)
	}
	if n, err := (&Worker{Ledger: c}).Process(context.Background(), evs); err != nil || n != len(evs) {
		t.Fatalf("worker = %d/%d, %v", n, len(evs), err)
	}
	if got := locked(); got != "0.000000" {
		t.Fatalf("locked at the end = %s, want 0", got)
	}
}
