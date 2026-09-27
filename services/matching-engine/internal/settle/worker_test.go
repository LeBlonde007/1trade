package settle

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/trade1/matching-engine/internal/engine"
)

// fakeLedger records calls and can fail a call a set number of times.
type fakeLedger struct {
	calls    []string
	failures map[string][]error // per call key, errors to return before succeeding
}

// next pops the next scripted error for key.
func (f *fakeLedger) next(key string) error {
	f.calls = append(f.calls, key)
	if errs := f.failures[key]; len(errs) > 0 {
		f.failures[key] = errs[1:]
		return errs[0]
	}
	return nil
}

// Settle records "settle:<trade>".
func (f *fakeLedger) Settle(_ context.Context, t engine.Trade) error {
	return f.next("settle:" + t.TradeID)
}

// Release records "release:<order>".
func (f *fakeLedger) Release(_ context.Context, id string) error { return f.next("release:" + id) }

// stream runs a real engine: a held sell rests, a held buy fills it partially and rests, then the
// buyer cancels. Returns the events in emission order.
func stream(t *testing.T) []engine.Event {
	t.Helper()
	e := engine.New(engine.Config{})
	ts := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	var evs []engine.Event
	sub := func(id, tenant string, side engine.Side, q string) {
		ts = ts.Add(time.Millisecond)
		r, err := e.Submit(engine.SubmitCmd{OrderID: id, TenantID: tenant, ProductID: "H100-SPOT", Side: side,
			Type: engine.Limit, Price: engine.MustFixed("3.00"), Quantity: engine.MustFixed(q), IsPaper: true, TS: ts})
		if err != nil {
			t.Fatal(err)
		}
		evs = append(evs, r.Events...)
	}
	sub("sell", "s", engine.Sell, "4")
	sub("buy", "b", engine.Buy, "10")
	ts = ts.Add(time.Millisecond)
	r, err := e.Cancel(engine.CancelCmd{OrderID: "buy", TenantID: "b", TS: ts})
	if err != nil {
		t.Fatal(err)
	}
	return append(evs, r.Events...)
}

// TestWorkerOrderAndReleases checks the trade settles before either reservation is released, and each
// held order is released exactly once, when it closes.
func TestWorkerOrderAndReleases(t *testing.T) {
	evs := stream(t)
	f := &fakeLedger{}
	n, err := (&Worker{Ledger: f}).Process(context.Background(), evs)
	if err != nil || n != len(evs) {
		t.Fatalf("process = %d, %v", n, err)
	}
	var trade string
	for _, ev := range evs {
		if ev.Trade != nil {
			trade = ev.Trade.TradeID
		}
	}
	want := []string{"settle:" + trade, "release:sell", "release:buy"}
	if fmt.Sprint(f.calls) != fmt.Sprint(want) {
		t.Errorf("calls = %v, want %v", f.calls, want)
	}
}

// TestWorkerRetriesTransient checks a transient failure is retried until it succeeds.
func TestWorkerRetriesTransient(t *testing.T) {
	evs := stream(t)
	f := &fakeLedger{failures: map[string][]error{"release:sell": {errors.New("503"), errors.New("timeout")}}}
	if _, err := (&Worker{Ledger: f, Backoff: time.Millisecond}).Process(context.Background(), evs); err != nil {
		t.Fatalf("process: %v", err)
	}
	count := 0
	for _, c := range f.calls {
		if c == "release:sell" {
			count++
		}
	}
	if count != 3 {
		t.Errorf("release:sell called %d times, want 3 (2 failures + success)", count)
	}
}

// TestWorkerHaltsOnConflict checks a conflict stops the stream at that event, so nothing after it —
// in particular no release — runs out of order.
func TestWorkerHaltsOnConflict(t *testing.T) {
	evs := stream(t)
	var trade string
	idx := -1
	for i, ev := range evs {
		if ev.Trade != nil {
			trade, idx = ev.Trade.TradeID, i
		}
	}
	f := &fakeLedger{failures: map[string][]error{"settle:" + trade: {ErrConflict}}}
	n, err := (&Worker{Ledger: f, Backoff: time.Millisecond}).Process(context.Background(), evs)
	if !errors.Is(err, ErrHalt) || !errors.Is(err, ErrConflict) || n != idx {
		t.Fatalf("process = %d, %v; want halt at %d with ErrConflict", n, err, idx)
	}
	for _, c := range f.calls {
		if c != "settle:"+trade {
			t.Errorf("ran %s after a halting conflict", c)
		}
	}
}

// TestWorkerContinuesPastUnsettleable checks a permanently refused trade is reported and the stream
// continues (it is an alert, not a stop).
func TestWorkerContinuesPastUnsettleable(t *testing.T) {
	evs := stream(t)
	var trade string
	for _, ev := range evs {
		if ev.Trade != nil {
			trade = ev.Trade.TradeID
		}
	}
	f := &fakeLedger{failures: map[string][]error{"settle:" + trade: {fmt.Errorf("%w: 402", ErrUnsettleable)}}}
	var alerted []string
	w := &Worker{Ledger: f, OnUnsettleable: func(tr engine.Trade, _ error) { alerted = append(alerted, tr.TradeID) }}
	if n, err := w.Process(context.Background(), evs); err != nil || n != len(evs) {
		t.Fatalf("process = %d, %v", n, err)
	}
	if len(alerted) != 1 || alerted[0] != trade {
		t.Errorf("alerted = %v", alerted)
	}
}
