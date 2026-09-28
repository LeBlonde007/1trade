package engine

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// TestVoidTombstonesUnknownID: a void is journaled, emits nothing, and makes the id unusable — a
// same-tenant resubmit gets the tombstone back without the risk check (so without a new reservation),
// and another tenant cannot take the id.
func TestVoidTombstonesUnknownID(t *testing.T) {
	var risked atomic.Int32
	e := New(Config{Risk: func(Order, Hold) string { risked.Add(1); return "" }})
	o, voided, err := e.Void(VoidCmd{OrderID: "x1", TenantID: "cust", IsPaper: true, TS: at(1)})
	if err != nil || !voided || o.State != Rejected || o.Reason != ReasonVoided || !o.IsPaper {
		t.Fatalf("void = %+v voided=%v err=%v", o, voided, err)
	}
	if j := e.Journal(); len(j) != 1 || j[0].Void == nil || j[0].Void.OrderID != "x1" {
		t.Fatalf("journal = %+v, want the void", j)
	}
	evs, err := New(Config{}).Apply(Command{Void: &VoidCmd{OrderID: "x9", TenantID: "t", TS: at(1)}})
	if err != nil || len(evs) != 0 {
		t.Fatalf("a void emitted %d events (%v)", len(evs), err)
	}

	r, err := e.Submit(limit("x1", "cust", Buy, "0.001000", "10", 2))
	if err != nil || !r.Duplicate || r.Order.State != Rejected || r.Order.Reason != ReasonVoided || len(r.Events) != 0 {
		t.Fatalf("resubmit of a voided id = %+v, %v", r, err)
	}
	if risked.Load() != 0 {
		t.Fatal("a voided id reached the risk check (it would reserve again)")
	}
	if _, err := e.Submit(limit("x1", "other", Buy, "0.001000", "10", 3)); !errors.Is(err, ErrOrderIDTaken) {
		t.Fatalf("another tenant took a voided id: %v", err)
	}
	if o, voided, err := e.Void(VoidCmd{OrderID: "x1", TenantID: "cust", TS: at(4)}); err != nil || voided || o.Reason != ReasonVoided {
		t.Fatalf("second void = %+v voided=%v err=%v, want the existing tombstone", o, voided, err)
	}
	if len(e.Journal()) != 1 {
		t.Fatalf("journal grew to %d; duplicates and repeat voids must not be journaled", len(e.Journal()))
	}
	if open := e.OpenOrders("cust", true); len(open) != 0 {
		t.Fatalf("tombstone listed as open: %+v", open)
	}
	if b, a := e.Depth("EAI-IDX", true, 0); len(b)+len(a) != 0 {
		t.Fatal("tombstone reached a book")
	}
}

// TestVoidOfKnownIDChangesNothing: an accepted order is returned as it stands, never tombstoned.
func TestVoidOfKnownIDChangesNothing(t *testing.T) {
	e := New(Config{})
	mustSubmit(t, e, limit("a1", "mm", Sell, "0.001000", "10", 1))
	o, voided, err := e.Void(VoidCmd{OrderID: "a1", TenantID: "mm", TS: at(2)})
	if err != nil || voided || o.State != Accepted {
		t.Fatalf("void of a live order = %+v voided=%v err=%v", o, voided, err)
	}
	if _, _, err := e.Void(VoidCmd{OrderID: "a1", TenantID: "intruder", TS: at(3)}); !errors.Is(err, ErrOrderIDTaken) {
		t.Fatalf("void naming another tenant = %v, want ErrOrderIDTaken", err)
	}
	if len(e.Journal()) != 1 {
		t.Fatalf("journal = %d commands, want only the submit", len(e.Journal()))
	}
	if _, a := e.Depth("EAI-IDX", true, 0); len(a) != 1 {
		t.Fatal("the live order left the book")
	}
}

// TestVoidValidationAndJournalFailure: bad voids and failed writes change nothing.
func TestVoidValidationAndJournalFailure(t *testing.T) {
	fail := true
	e := New(Config{Persist: func(uint64, Command) error {
		if fail {
			return errors.New("db down")
		}
		return nil
	}})
	for _, c := range []VoidCmd{{TenantID: "t", TS: at(1)}, {OrderID: "x", TS: at(1)}} {
		if _, _, err := e.Void(c); !errors.Is(err, ErrMissingID) {
			t.Fatalf("void %+v = %v, want ErrMissingID", c, err)
		}
	}
	if _, _, err := e.Void(VoidCmd{OrderID: "x", TenantID: "t"}); !errors.Is(err, ErrMissingTimestamp) {
		t.Fatalf("void without TS = %v", err)
	}
	if _, _, err := e.Void(VoidCmd{OrderID: "x", TenantID: "t", TS: at(1)}); !errors.Is(err, ErrJournal) {
		t.Fatalf("void with a failing journal = %v, want ErrJournal", err)
	}
	fail = false
	if r := mustSubmit(t, e, limit("x", "t", Buy, "0.001000", "1", 2)); r.Duplicate || r.Order.State != Accepted {
		t.Fatalf("an unjournaled void still took effect: %+v", r)
	}
}

// TestOnUnjournaledReportsOnlyFailedHeldSubmits: the hook fires once per submit whose write failed
// after the risk check, with the order and its hold — and never for journaled submits, voids, or
// orders with nothing to hold.
func TestOnUnjournaledReportsOnlyFailedHeldSubmits(t *testing.T) {
	var got []Order
	fail := false
	e := New(Config{
		Persist: func(uint64, Command) error {
			if fail {
				return errors.New("db down")
			}
			return nil
		},
		OnUnjournaled: func(o Order) { got = append(got, o) },
	})
	mustSubmit(t, e, limit("ok", "t1", Sell, "0.001000", "1", 1))
	if len(got) != 0 {
		t.Fatalf("hook fired for a journaled submit: %+v", got)
	}
	fail = true
	if _, err := e.Submit(limit("lost", "t2", Buy, "0.002000", "5", 2)); !errors.Is(err, ErrJournal) {
		t.Fatal(err)
	}
	// A market buy on a book with no asks holds nothing (Reserve skips it), so it is not reported.
	if _, err := e.Submit(SubmitCmd{OrderID: "mkt", TenantID: "t3", ProductID: "H100-SPOT", Side: Buy, Type: Market, Quantity: qty("1"), IsPaper: true, TS: at(3)}); !errors.Is(err, ErrJournal) {
		t.Fatal(err)
	}
	if _, _, err := e.Void(VoidCmd{OrderID: "v", TenantID: "t", TS: at(4)}); !errors.Is(err, ErrJournal) {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].OrderID != "lost" || got[0].TenantID != "t2" || !got[0].IsPaper || got[0].Hold.Amount != MustFixed("0.010100") {
		t.Fatalf("hook calls = %+v, want only 'lost' with its 0.0101 hold", got)
	}
}

// TestSubmitAndVoidAreAtomic races a submit and a void on each of 200 ids: exactly one of them may
// reach the journal per id, and the survivor decides the id's fate for good.
func TestSubmitAndVoidAreAtomic(t *testing.T) {
	e := New(Config{})
	var wg sync.WaitGroup
	for i := range 200 {
		id := fmt.Sprintf("r%d", i)
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _ = e.Submit(limit(id, "t", Sell, "0.001000", "1", i+1))
		}()
		go func() {
			defer wg.Done()
			_, _, _ = e.Void(VoidCmd{OrderID: id, TenantID: "t", TS: at(i + 1)})
		}()
	}
	wg.Wait()
	seen := map[string]int{}
	for _, c := range e.Journal() {
		switch {
		case c.Submit != nil:
			seen[c.Submit.OrderID]++
		case c.Void != nil:
			seen[c.Void.OrderID]++
		}
	}
	for i := range 200 {
		id := fmt.Sprintf("r%d", i)
		if seen[id] != 1 {
			t.Fatalf("%s journaled %d times; want exactly one of submit or void", id, seen[id])
		}
		o, _ := e.Order(id, "t")
		if (o.Reason == ReasonVoided) == (o.State == Accepted) {
			t.Fatalf("%s ended as %+v", id, o)
		}
	}
}
