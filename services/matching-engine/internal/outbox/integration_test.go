package outbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/trade1/matching-engine/internal/engine"
	"github.com/trade1/matching-engine/internal/journal"
)

// TestRelayOverPostgresAndJetStream is the real path: engine → Postgres journal → relay (Postgres
// cursor) → JetStream. Every event lands once, in order; losing the cursor and relaying everything
// again adds nothing, because JetStream drops the republished message ids. Skips without DATABASE_URL.
func TestRelayOverPostgresAndJetStream(t *testing.T) {
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("ob_%d", time.Now().UnixNano())
	admin, err := pgxpool.New(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE") //nolint:errcheck // test cleanup
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	dsn := base + sep + "search_path=" + schema
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	files, err := filepath.Glob("../../migrations/*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("no migrations found: %v", err)
	}
	sort.Strings(files)
	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("migrate %s: %v", f, err)
		}
	}

	// The live engine, journaled to Postgres.
	js, err := journal.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer js.Close()
	e, err := journal.Recover(ctx, js, engine.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var wantIDs []string
	ts := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	for i := range 12 {
		ts = ts.Add(time.Second)
		r, err := e.Submit(engine.SubmitCmd{OrderID: fmt.Sprintf("o%d", i), TenantID: fmt.Sprintf("t%d", i%3), ProductID: "EAI-IDX",
			Side: []engine.Side{engine.Buy, engine.Sell}[i%2], Type: engine.Limit, Price: engine.MustFixed("0.001000"),
			Quantity: engine.MustFixed("10"), IsPaper: true, TS: ts})
		if err != nil {
			t.Fatal(err)
		}
		for _, ev := range r.Events {
			wantIDs = append(wantIDs, msgID(ev))
		}
		if i == 5 { // a void row mid-journal: chain-verified and stepped over like any command
			if _, _, err := e.Void(engine.VoidCmd{OrderID: "v5", TenantID: "t0", IsPaper: true, TS: ts}); err != nil {
				t.Fatal(err)
			}
		}
	}
	epoch, _ := js.Epoch(ctx)

	// JetStream.
	srv, err := natsserver.NewServer(&natsserver.Options{Port: -1, JetStream: true, StoreDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats not ready")
	}
	defer srv.Shutdown()
	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	sink, err := NewJetStreamSink(nc)
	if err != nil {
		t.Fatal(err)
	}

	relay := func() int {
		n, err := New(engine.Config{Epoch: epoch}, js, NewPGCursor(pool, "nats"), sink).Step(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := relay(); n != len(wantIDs) {
		t.Fatalf("relayed %d, want %d", n, len(wantIDs))
	}

	// Read the stream back in order.
	jsc, _ := nc.JetStream()
	sub, err := jsc.SubscribeSync(">", nats.BindStream(Stream), nats.OrderedConsumer(), nats.DeliverAll())
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(wantIDs))
	for range wantIDs {
		m, err := sub.NextMsg(3 * time.Second)
		if err != nil {
			t.Fatalf("after %d messages: %v", len(got), err)
		}
		got = append(got, m.Header.Get(nats.MsgIdHdr))
	}
	if fmt.Sprint(got) != fmt.Sprint(wantIDs) {
		t.Fatal("stream order differs from engine emission order")
	}

	// Lose the cursor and relay everything again: JetStream must drop every duplicate.
	if _, err := pool.Exec(ctx, `DELETE FROM engine_outbox_cursor`); err != nil {
		t.Fatal(err)
	}
	if n := relay(); n != len(wantIDs) {
		t.Fatalf("second relay attempted %d, want %d", n, len(wantIDs))
	}
	info, err := jsc.StreamInfo(Stream)
	if err != nil {
		t.Fatal(err)
	}
	if info.State.Msgs != uint64(len(wantIDs)) {
		t.Errorf("stream holds %d messages after a full republish, want %d (dedupe failed)", info.State.Msgs, len(wantIDs))
	}
}
