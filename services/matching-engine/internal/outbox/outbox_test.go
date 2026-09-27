package outbox

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/trade1/matching-engine/internal/engine"
	"github.com/trade1/matching-engine/internal/events"
	"github.com/trade1/matching-engine/internal/journal"
)

// memSource serves a fixed journal.
type memSource struct{ entries []journal.Entry }

// ReadAfter returns entries after seq.
func (m *memSource) ReadAfter(_ context.Context, after uint64, _ string, _ int) ([]journal.Entry, error) {
	var out []journal.Entry
	for _, e := range m.entries {
		if e.Seq > after {
			out = append(out, e)
		}
	}
	return out, nil
}

// memCursor keeps the position in memory.
type memCursor struct{ p Position }

// Load returns the position.
func (c *memCursor) Load(context.Context) (Position, error) { return c.p, nil }

// Save stores the position.
func (c *memCursor) Save(_ context.Context, p Position) error { c.p = p; return nil }

// msg is one published message.
type msg struct{ subject, id, data string }

// memSink records publishes and can fail the Nth one.
type memSink struct {
	got    []msg
	failAt int // fail the publish with this 1-based index (0 = never)
	calls  int
}

// Publish records or fails.
func (s *memSink) Publish(_ context.Context, subject, id string, data []byte) error {
	s.calls++
	if s.calls == s.failAt {
		return errors.New("nats down")
	}
	s.got = append(s.got, msg{subject, id, string(data)})
	return nil
}

// live runs a real engine through a busy stream and returns its journal and the messages it emitted.
func live(t *testing.T) ([]journal.Entry, []msg, engine.Config) {
	t.Helper()
	cfg := engine.Config{Epoch: "relay-test"}
	e := engine.New(cfg)
	ts := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	var want []msg
	add := func(evs []engine.Event) {
		for _, ev := range evs {
			subj, data, err := events.Encode(ev)
			if err != nil {
				t.Fatal(err)
			}
			want = append(want, msg{subj, msgID(ev), string(data)})
		}
	}
	for i := range 40 {
		ts = ts.Add(time.Second)
		side := []engine.Side{engine.Buy, engine.Sell}[i%2]
		r, err := e.Submit(engine.SubmitCmd{OrderID: fmt.Sprintf("o%d", i), TenantID: fmt.Sprintf("t%d", i%5), ProductID: "EAI-IDX",
			Side: side, Type: engine.Limit, Price: engine.MustFixed(fmt.Sprintf("0.0010%02d", i%7)),
			Quantity: engine.MustFixed(fmt.Sprint(5 + i)), IsPaper: true, TS: ts})
		if err != nil {
			t.Fatal(err)
		}
		add(r.Events)
		if i%9 == 8 {
			if r, err := e.Cancel(engine.CancelCmd{OrderID: fmt.Sprintf("o%d", i-1), TenantID: fmt.Sprintf("t%d", (i-1)%5), TS: ts}); err == nil {
				add(r.Events)
			}
		}
	}
	entries := make([]journal.Entry, 0, len(e.Journal()))
	for i, c := range e.Journal() {
		entries = append(entries, journal.Entry{Seq: uint64(i + 1), Command: c, ChainHash: fmt.Sprint("h", i+1)})
	}
	if len(want) < 50 {
		t.Fatalf("stream too thin: %d events", len(want))
	}
	return entries, want, cfg
}

// TestRelayPublishesExactlyWhatTheEngineEmitted: the relay's output is byte-identical to the live
// engine's, in the same order, with the deterministic ids as message ids.
func TestRelayPublishesExactlyWhatTheEngineEmitted(t *testing.T) {
	entries, want, cfg := live(t)
	sink := &memSink{}
	n, err := New(cfg, &memSource{entries}, &memCursor{}, sink).Step(context.Background())
	if err != nil || n != len(want) {
		t.Fatalf("step = %d, %v; want %d", n, err, len(want))
	}
	if !reflect.DeepEqual(sink.got, want) {
		t.Fatal("relay output differs from the engine's emission")
	}
}

// TestRelayResumesWithoutDuplicates: a restarted relay (new shadow engine, same cursor) publishes only
// what came after, and a failure mid-command resumes at exactly the failed event.
func TestRelayResumesWithoutDuplicates(t *testing.T) {
	entries, want, cfg := live(t)
	cur := &memCursor{}
	half := len(entries) / 2
	first := &memSink{}
	if _, err := New(cfg, &memSource{entries[:half]}, cur, first).Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	failing := &memSink{failAt: 3} // dies on its 3rd publish, mid-way through a command's events
	if _, err := New(cfg, &memSource{entries}, cur, failing).Step(context.Background()); err == nil {
		t.Fatal("expected the injected publish failure")
	}
	last := &memSink{}
	if _, err := New(cfg, &memSource{entries}, cur, last).Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := append(append(append([]msg{}, first.got...), failing.got...), last.got...)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resumed output: %d messages, want %d, and must match exactly with no gaps or duplicates", len(got), len(want))
	}
	if again, _ := New(cfg, &memSource{entries}, cur, &memSink{}).Step(context.Background()); again != 0 {
		t.Errorf("fully caught-up relay published %d more", again)
	}
}

// TestRelayFollowsNewEntries: stepping again after the journal grows publishes only the new tail.
func TestRelayFollowsNewEntries(t *testing.T) {
	entries, want, cfg := live(t)
	src := &memSource{entries[:10]}
	sink := &memSink{}
	r := New(cfg, src, &memCursor{}, sink)
	if _, err := r.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	src.entries = entries // the journal grew
	if _, err := r.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sink.got, want) {
		t.Fatalf("following output: %d messages, want %d", len(sink.got), len(want))
	}
}
