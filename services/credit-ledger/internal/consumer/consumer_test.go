package consumer

import (
	"fmt"
	"net"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// TestSuperviseStartsOnceNATSIsUp: the ledger boots before NATS. The consumer must keep retrying and
// start once NATS is up; before, one failed attempt meant no debits until the next restart.
func TestSuperviseStartsOnceNATSIsUp(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	src := Source{Subject: "test.usage.v1", Stream: "TEST_USAGE", Durable: "test-debit"}
	stop := Supervise(fmt.Sprintf("nats://127.0.0.1:%d", port), src, nil, nil, 50*time.Millisecond)
	defer stop()
	time.Sleep(200 * time.Millisecond) // several failed attempts

	srv, err := natsserver.NewServer(&natsserver.Options{Host: "127.0.0.1", Port: port, JetStream: true, StoreDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	go srv.Start()
	defer srv.Shutdown()
	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatal("nats not ready")
	}
	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	defer nc.Close()
	js, _ := nc.JetStream()
	for i := 0; ; i++ {
		if _, err := js.ConsumerInfo(src.Stream, src.Durable); err == nil {
			break
		}
		if i > 100 {
			t.Fatal("the supervised consumer never started")
		}
		time.Sleep(50 * time.Millisecond)
	}
	stop()
	stop() // idempotent
}

// TestZeroUnitsBookNothing: compute served from a prepaid reservation reports zero units; the
// consumer acknowledges it without a ledger movement (the consumer here has no store, so any
// movement would panic).
func TestZeroUnitsBookNothing(t *testing.T) {
	c := &UsageConsumer{}
	c.handle(&nats.Msg{Subject: "compute.usage.v1", Data: []byte(`{"usage_id":"u-1","tenant_id":"00000000-0000-4000-8000-000000000001",
		"credit_type":"gpu_h100","gpu_seconds":"3600.000000","units":"0.000000","reserved":true,"is_paper":true}`)})
}
