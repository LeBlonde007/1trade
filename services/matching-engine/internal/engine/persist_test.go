package engine

import (
	"errors"
	"reflect"
	"testing"
)

// TestPersistIsWriteAhead checks every state-changing command is persisted with a contiguous 1-based
// sequence, and that a failed write refuses the command with no trace in the books or journal.
func TestPersistIsWriteAhead(t *testing.T) {
	var seqs []uint64
	fail := false
	e := New(Config{Persist: func(seq uint64, _ Command) error {
		if fail {
			return errors.New("disk full")
		}
		seqs = append(seqs, seq)
		return nil
	}})
	mustSubmit(t, e, limit("a1", "mm", Sell, "0.001000", "10", 1))
	if _, err := e.Cancel(CancelCmd{OrderID: "a1", TenantID: "mm", TS: at(2)}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.ExpireDay(ExpireDayCmd{TS: at(3)}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(seqs, []uint64{1, 2, 3}) {
		t.Fatalf("persisted seqs = %v, want [1 2 3]", seqs)
	}

	fail = true
	if _, err := e.Submit(limit("b1", "cust", Buy, "0.001000", "10", 4)); !errors.Is(err, ErrJournal) {
		t.Fatalf("submit with failing journal err = %v, want ErrJournal", err)
	}
	if _, err := e.Submit(SubmitCmd{OrderID: "h1", TenantID: "cust", ProductID: "H100-SPOT", Side: Buy, Type: Limit, Price: px("2.99"), Quantity: qty("1"), IsPaper: true, TS: at(5)}); !errors.Is(err, ErrJournal) {
		t.Fatalf("first submit on a new book err = %v, want ErrJournal", err)
	}
	if _, ok := e.Order("b1", "cust"); ok {
		t.Error("refused order is known to the engine")
	}
	if _, ok := e.books[bookKey{"H100-SPOT", true}]; ok {
		t.Error("refused command registered a book")
	}
	if len(e.Journal()) != 3 {
		t.Errorf("journal has %d commands, want 3", len(e.Journal()))
	}

	fail = false
	r := mustSubmit(t, e, limit("b1", "cust", Buy, "0.001000", "10", 6)) // the same id works after recovery
	if r.Duplicate || seqs[len(seqs)-1] != 4 {
		t.Errorf("retry after failure: duplicate=%v last seq=%d, want fresh order at seq 4", r.Duplicate, seqs[len(seqs)-1])
	}
}

// TestReplayDoesNotRepersist checks replayed commands are not written again, and that the returned
// engine persists new commands at the next sequence.
func TestReplayDoesNotRepersist(t *testing.T) {
	src := New(Config{})
	mustSubmit(t, src, limit("a1", "mm", Sell, "0.001000", "10", 1))
	mustSubmit(t, src, limit("b1", "cust", Buy, "0.001000", "4", 2))

	var seqs []uint64
	e, _, err := Replay(Config{Persist: func(seq uint64, _ Command) error { seqs = append(seqs, seq); return nil }}, src.Journal())
	if err != nil {
		t.Fatal(err)
	}
	if len(seqs) != 0 {
		t.Fatalf("replay re-persisted %v", seqs)
	}
	mustSubmit(t, e, limit("b2", "cust", Buy, "0.001000", "1", 3))
	if !reflect.DeepEqual(seqs, []uint64{3}) {
		t.Errorf("post-replay seqs = %v, want [3]", seqs)
	}
}
