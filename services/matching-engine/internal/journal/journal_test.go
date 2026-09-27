package journal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trade1/matching-engine/internal/engine"
)

var t0 = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

// freshSchema creates an isolated schema with the journal migration applied and returns a DSN pinned
// to it. Skips unless DATABASE_URL is set.
func freshSchema(t testing.TB) string {
	t.Helper()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		t.Skip("DATABASE_URL not set; skipping journal integration test")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("jt_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		p, err := pgxpool.New(context.Background(), base)
		if err == nil {
			_, _ = p.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
			p.Close()
		}
	})
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	dsn := base + sep + "search_path=" + schema
	p, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	for _, f := range []string{"../../migrations/0001_journal.sql", "../../migrations/0002_meta.sql"} {
		sql, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("migrate %s: %v", f, err)
		}
	}
	return dsn
}

// open opens a store and recovers an engine from it.
func open(t *testing.T, dsn string) (*Store, *engine.Engine) {
	t.Helper()
	s, err := Open(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	e, err := Recover(context.Background(), s, engine.Config{})
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	return s, e
}

// order builds a paper limit order; n spaces timestamps.
func order(id, tenant string, side engine.Side, price, qty string, n int) engine.SubmitCmd {
	return engine.SubmitCmd{OrderID: id, TenantID: tenant, ProductID: "EAI-IDX", Side: side, Type: engine.Limit,
		Price: engine.MustFixed(price), Quantity: engine.MustFixed(qty), IsPaper: true, TS: t0.Add(time.Duration(n) * time.Millisecond)}
}

// drive submits a mixed stream that trades, rests, cancels and expires.
func drive(t *testing.T, e *engine.Engine) {
	t.Helper()
	n := 0
	for i := range 30 {
		side, price := engine.Sell, fmt.Sprintf("0.0010%02d", i%7)
		if i%2 == 0 {
			side, price = engine.Buy, fmt.Sprintf("0.0010%02d", (i+3)%7)
		}
		n++
		c := order(fmt.Sprintf("o%d", i), fmt.Sprintf("t%d", i%3), side, price, fmt.Sprintf("%d", 5+i), n)
		if i%5 == 0 {
			c.TIF = engine.Day
		}
		if _, err := e.Submit(c); err != nil {
			t.Fatalf("submit o%d: %v", i, err)
		}
		if i%7 == 6 {
			n++
			_, _ = e.Cancel(engine.CancelCmd{OrderID: fmt.Sprintf("o%d", i-1), TenantID: fmt.Sprintf("t%d", (i-1)%3), TS: t0.Add(time.Duration(n) * time.Millisecond)})
		}
	}
	n++
	if _, err := e.ExpireDay(engine.ExpireDayCmd{TS: t0.Add(time.Duration(n) * time.Millisecond)}); err != nil {
		t.Fatal(err)
	}
}

// TestCrashRecovery runs an engine against the journal, drops it, recovers a new one from Postgres,
// and checks the books, orders and journal are identical and that new commands continue the chain.
func TestCrashRecovery(t *testing.T) {
	dsn := freshSchema(t)
	s1, e1 := open(t, dsn)
	drive(t, e1)
	wantBids, wantAsks := e1.Depth("EAI-IDX", true, 0)
	wantJournal := e1.Journal()
	s1.Close() // "crash": the process and its memory are gone

	s2, e2 := open(t, dsn)
	defer s2.Close()
	gotBids, gotAsks := e2.Depth("EAI-IDX", true, 0)
	if !reflect.DeepEqual(gotBids, wantBids) || !reflect.DeepEqual(gotAsks, wantAsks) {
		t.Fatalf("recovered book differs:\n bids %v vs %v\n asks %v vs %v", gotBids, wantBids, gotAsks, wantAsks)
	}
	if !reflect.DeepEqual(e2.Journal(), wantJournal) {
		t.Fatal("recovered journal differs")
	}
	for _, tenant := range []string{"t0", "t1", "t2"} {
		if len(e2.OpenOrders(tenant, true)) == 0 && len(e1.OpenOrders(tenant, true)) != 0 {
			t.Errorf("open orders for %s lost", tenant)
		}
	}
	if len(wantBids)+len(wantAsks) == 0 {
		t.Fatal("test stream left an empty book; it proves nothing")
	}

	// New work appends at n+1 and survives a second restart.
	if _, err := e2.Submit(order("after", "t9", engine.Buy, "0.000500", "1", 1000)); err != nil {
		t.Fatalf("submit after recovery: %v", err)
	}
	s2.Close()
	s3, e3 := open(t, dsn)
	defer s3.Close()
	if o, ok := e3.Order("after", "t9"); !ok || o.State != engine.Accepted {
		t.Errorf("post-recovery order not recovered: %+v %v", o, ok)
	}
}

// TestAppendOnlyAndTamperDetection checks the DB refuses edits, and that a journal altered anyway
// (triggers disabled by a superuser) refuses to load.
func TestAppendOnlyAndTamperDetection(t *testing.T) {
	dsn := freshSchema(t)
	s, e := open(t, dsn)
	drive(t, e)
	s.Close()

	ctx := context.Background()
	p, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	for _, q := range []string{
		`UPDATE engine_journal SET command_json = command_json WHERE seq = 1`,
		`DELETE FROM engine_journal WHERE seq = 1`,
		`TRUNCATE engine_journal`,
	} {
		if _, err := p.Exec(ctx, q); err == nil || !strings.Contains(err.Error(), "append-only") {
			t.Errorf("%q was not refused: %v", q, err)
		}
	}

	// Bypass the guard, as only a superuser could, and alter one quantity.
	if _, err := p.Exec(ctx, `ALTER TABLE engine_journal DISABLE TRIGGER USER`); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Exec(ctx, `UPDATE engine_journal SET command_json = replace(command_json, '"Quantity":5000000', '"Quantity":6000000') WHERE seq = 1`); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	if _, err := Recover(ctx, s2, engine.Config{}); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("tampered journal recover err = %v, want ErrCorrupt", err)
	}

	// A gap is also refused.
	if _, err := p.Exec(ctx, `DELETE FROM engine_journal WHERE seq = 2`); err != nil {
		t.Fatal(err)
	}
	if _, err := s2.Load(ctx); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("gapped journal load err = %v, want ErrCorrupt", err)
	}
}

// TestSecondWriterFails checks two engines recovered from the same journal cannot both write: the
// second append at the same seq fails and its command is refused.
func TestSecondWriterFails(t *testing.T) {
	dsn := freshSchema(t)
	sa, ea := open(t, dsn)
	defer sa.Close()
	sb, eb := open(t, dsn)
	defer sb.Close()
	if _, err := ea.Submit(order("a", "t1", engine.Buy, "0.001000", "1", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := eb.Submit(order("b", "t2", engine.Sell, "0.001000", "1", 2)); !errors.Is(err, engine.ErrJournal) {
		t.Fatalf("second writer err = %v, want ErrJournal", err)
	}
	if _, ok := eb.Order("b", "t2"); ok {
		t.Error("second writer applied a command it could not journal")
	}
}

// TestDatabaseDownRefusesCommands checks an unreachable journal refuses commands instead of letting
// the books run ahead of what can be recovered.
func TestDatabaseDownRefusesCommands(t *testing.T) {
	dsn := freshSchema(t)
	s, e := open(t, dsn)
	s.Close() // pool closed: every write fails
	if _, err := e.Submit(order("x", "t1", engine.Buy, "0.001000", "1", 1)); !errors.Is(err, engine.ErrJournal) {
		t.Fatalf("submit with journal down err = %v, want ErrJournal", err)
	}
	if bids, _ := e.Depth("EAI-IDX", true, 0); len(bids) != 0 {
		t.Errorf("book changed without a journal write: %v", bids)
	}
}

// BenchmarkSubmitJournaled measures order acceptance including the synchronous Postgres append — the
// cost the write-ahead rule adds against the < 10 ms acceptance budget (SPEC.md §7).
func BenchmarkSubmitJournaled(b *testing.B) {
	dsn := freshSchema(b)
	s, err := Open(context.Background(), dsn)
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	e, err := Recover(context.Background(), s, engine.Config{})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := range b.N {
		side, price := engine.Buy, "0.000900"
		if i%2 == 1 {
			side, price = engine.Sell, "0.001100"
		}
		if _, err := e.Submit(order(fmt.Sprintf("b%d", i), fmt.Sprintf("t%d", i%5), side, price, "1", i+1)); err != nil {
			b.Fatal(err)
		}
	}
}

// TestEpochIsStableAndDistinct checks a journal keeps one epoch across restarts (so replayed trade ids
// match the originals) and two journals get different epochs (so their trade ids never collide).
func TestEpochIsStableAndDistinct(t *testing.T) {
	dsnA, dsnB := freshSchema(t), freshSchema(t)
	sa, ea := open(t, dsnA)
	if _, err := ea.Submit(order("s", "t1", engine.Sell, "0.001000", "1", 1)); err != nil {
		t.Fatal(err)
	}
	r, err := ea.Submit(order("b", "t2", engine.Buy, "0.001000", "1", 2))
	if err != nil {
		t.Fatal(err)
	}
	var tradeA string
	for _, ev := range r.Events {
		if ev.Trade != nil {
			tradeA = ev.Trade.TradeID
		}
	}
	epochA, _ := sa.Epoch(context.Background())
	sa.Close()

	sa2, ea2 := open(t, dsnA)
	defer sa2.Close()
	if again, _ := sa2.Epoch(context.Background()); again != epochA || epochA == "" {
		t.Fatalf("epoch changed across restart: %q → %q", epochA, again)
	}
	var replayed string
	_, evs, err := engine.Replay(engine.Config{Epoch: epochA}, ea2.Journal())
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range evs {
		if ev.Trade != nil {
			replayed = ev.Trade.TradeID
		}
	}
	if replayed != tradeA {
		t.Errorf("replayed trade id %s, original %s", replayed, tradeA)
	}

	sb, eb := open(t, dsnB)
	defer sb.Close()
	if _, err := eb.Submit(order("s", "t1", engine.Sell, "0.001000", "1", 1)); err != nil {
		t.Fatal(err)
	}
	r, err = eb.Submit(order("b", "t2", engine.Buy, "0.001000", "1", 2))
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range r.Events {
		if ev.Trade != nil && ev.Trade.TradeID == tradeA {
			t.Error("a second journal minted the same trade id for the same history")
		}
	}
}

// TestReadAfterVerifiesContinuity checks the relay's reader: it follows the chain from a known head,
// and refuses a head that does not match what is stored.
func TestReadAfterVerifiesContinuity(t *testing.T) {
	dsn := freshSchema(t)
	s, e := open(t, dsn)
	defer s.Close()
	drive(t, e)
	all, err := s.ReadAfter(context.Background(), 0, "", 0)
	if err != nil || len(all) != len(e.Journal()) {
		t.Fatalf("read all = %d entries, %v; want %d", len(all), err, len(e.Journal()))
	}
	tail, err := s.ReadAfter(context.Background(), 3, all[2].ChainHash, 0)
	if err != nil || len(tail) != len(all)-3 || tail[0].Seq != 4 {
		t.Fatalf("read after 3 = %d entries starting %v, %v", len(tail), tail, err)
	}
	if _, err := s.ReadAfter(context.Background(), 3, "not-the-head", 0); !errors.Is(err, ErrCorrupt) {
		t.Errorf("wrong head err = %v, want ErrCorrupt", err)
	}
}
