package events

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace"

	"github.com/trade1/inference-gateway/internal/obs"
)

// freePort returns a TCP port nothing is listening on.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// TestPublisherSurvivesNATSStartingLate: the gateway boots before NATS. It must not fall back to
// logging for good (which silently stopped every debit); once NATS is up, an event lands in the
// usage stream (created by the gateway if the ledger has not yet) carrying the request's trace.
func TestPublisherSurvivesNATSStartingLate(t *testing.T) {
	port := freePort(t)
	p, err := NewNatsPublisher(fmt.Sprintf("nats://127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("NATS down at boot must not be fatal: %v", err)
	}
	defer p.Close()

	srv, err := natsserver.NewServer(&natsserver.Options{Host: "127.0.0.1", Port: port, JetStream: true, StoreDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	go srv.Start()
	defer srv.Shutdown()
	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats not ready")
	}
	for i := 0; !p.nc.IsConnected(); i++ {
		if i > 100 {
			t.Fatal("publisher never reconnected")
		}
		time.Sleep(50 * time.Millisecond)
	}

	if _, err := obs.InitTracing(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	tid, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	sid, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{TraceID: tid, SpanID: sid, TraceFlags: trace.FlagsSampled}))
	if err := p.PublishUsage(ctx, UsageEvent{RequestID: "infreq_1", TenantID: "t", Model: "m", CreditType: "text", Units: "1.000000", IsPaper: true}); err != nil {
		t.Fatal(err)
	}

	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	js, _ := nc.JetStream()
	sub, err := js.SubscribeSync(subject, nats.BindStream(Stream.Name), nats.DeliverAll())
	if err != nil {
		t.Fatalf("usage stream missing: %v", err)
	}
	m, err := sub.NextMsg(3 * time.Second)
	if err != nil {
		t.Fatalf("event not captured by the stream: %v", err)
	}
	var got UsageEvent
	_ = json.Unmarshal(m.Data, &got)
	if got.RequestID != "infreq_1" {
		t.Fatalf("captured %+v", got)
	}
	if tp := m.Header.Get("traceparent"); tp != "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
		t.Fatalf("traceparent header = %q", tp)
	}
}
