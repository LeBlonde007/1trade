// Package consumer subscribes to the platform's usage events and debits the ledger once per metered
// unit of work — inference.usage.v1 (per request) and compute.usage.v1 (per metered GPU interval).
// Each debit reuses store.ApplyMovement and is idempotent on the event's id (request_id / usage_id),
// so JetStream's at-least-once delivery can never double-bill. This is the consumer side of
// docs/contracts/events/inference.usage.v1.yaml and events/compute.usage.v1.yaml.
package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/exascale/credit-ledger/internal/domain"
	"github.com/exascale/credit-ledger/internal/events"
	"github.com/exascale/credit-ledger/internal/store"
	"github.com/nats-io/nats.go"
)

// Source describes one usage stream this consumer drains: its NATS subject, the JetStream stream that
// captures it, and the durable consumer name. One UsageConsumer is started per source.
type Source struct {
	Subject string
	Stream  string
	Durable string
}

// Inference is the inference.usage.v1 source (debit on request_id).
var Inference = Source{Subject: "inference.usage.v1", Stream: "INFERENCE_USAGE", Durable: "credit-ledger-debit"}

// Compute is the compute.usage.v1 source (debit the gpu_* credit on usage_id).
var Compute = Source{Subject: "compute.usage.v1", Stream: "COMPUTE_USAGE", Durable: "credit-ledger-compute-debit"}

// usageEvent mirrors the fields the debit needs from either inference.usage.v1 (request_id) or
// compute.usage.v1 (usage_id). id() returns whichever idempotency key is present.
type usageEvent struct {
	RequestID    string  `json:"request_id"`
	UsageID      string  `json:"usage_id"`
	TenantID     string  `json:"tenant_id"`
	SubAccountID *string `json:"sub_account_id"`
	CreditType   string  `json:"credit_type"`
	Units        string  `json:"units"`
	IsPaper      bool    `json:"is_paper"`
}

// id returns the event's idempotency key — request_id for inference, usage_id for compute.
func (e usageEvent) id() string {
	if e.RequestID != "" {
		return e.RequestID
	}
	return e.UsageID
}

// UsageConsumer holds the durable JetStream pull subscription that debits the ledger on usage events
// from one Source, plus the fetch loop's stop signal.
type UsageConsumer struct {
	nc   *nats.Conn
	sub  *nats.Subscription
	st   *store.Store
	pub  events.Publisher
	src  Source
	stop chan struct{}
}

// Start connects to NATS/JetStream, ensures the stream capturing src.Subject exists, and begins
// consuming with a durable PULL subscription (a fetch loop). Pull is deliberate: a push durable is
// exclusive — one active subscription at a time — so a pod restart races the server's still-"bound"
// old deliver subject and fails with "consumer is already bound to a subscription", silently stopping
// all debits. A pull consumer rebinds cleanly across restarts. Close to stop. Start one per Source.
func Start(url string, src Source, st *store.Store, pub events.Publisher) (*UsageConsumer, error) {
	nc, err := nats.Connect(url, nats.Name("credit-ledger-consumer-"+src.Durable), nats.Timeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	// Ensure the stream exists (create once; tolerate an already-created one). It must exist before
	// the producer publishes, or events aren't captured.
	if _, err := js.StreamInfo(src.Stream); err != nil {
		if _, err := js.AddStream(&nats.StreamConfig{
			Name: src.Stream, Subjects: []string{src.Subject},
			Storage: nats.FileStorage, MaxAge: 14 * 24 * time.Hour,
		}); err != nil {
			nc.Close()
			return nil, fmt.Errorf("add stream: %w", err)
		}
	}
	// Migrate a legacy push-based durable (from the old js.Subscribe path) to pull: a pull
	// subscription cannot bind to a consumer that has a DeliverSubject, so delete it and let
	// PullSubscribe recreate it. The old consumer never acked (it failed to bind), so no progress is
	// lost; the debit is idempotent on the event id regardless.
	if info, err := js.ConsumerInfo(src.Stream, src.Durable); err == nil && info.Config.DeliverSubject != "" {
		_ = js.DeleteConsumer(src.Stream, src.Durable)
	}
	c := &UsageConsumer{nc: nc, st: st, pub: pub, src: src, stop: make(chan struct{})}
	sub, err := js.PullSubscribe(src.Subject, src.Durable, nats.AckExplicit(), nats.MaxDeliver(5))
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("pull subscribe: %w", err)
	}
	c.sub = sub
	go c.loop()
	slog.Info("consuming usage → ledger debit (pull)", "subject", src.Subject, "stream", src.Stream, "durable", src.Durable)
	return c, nil
}

// loop fetches batches of usage events and debits each, until Close. A fetch timeout (no messages in
// the window) is normal and just polls again; other fetch errors back off briefly then retry.
func (c *UsageConsumer) loop() {
	for {
		select {
		case <-c.stop:
			return
		default:
		}
		msgs, err := c.sub.Fetch(10, nats.MaxWait(2*time.Second))
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				continue // no messages this window — poll again
			}
			slog.Error("usage fetch failed — retrying", "err", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		for _, m := range msgs {
			c.handle(m)
		}
	}
}

// handle debits the ledger for one usage event. Idempotent on the event id (request_id / usage_id),
// so a redelivery is a no-op. Unparseable/invalid messages are terminated (poison); transient store
// errors are NAK'd.
func (c *UsageConsumer) handle(msg *nats.Msg) {
	var e usageEvent
	if err := json.Unmarshal(msg.Data, &e); err != nil {
		slog.Error("usage event unparseable — terminating", "err", err)
		_ = msg.Term()
		return
	}
	id := e.id()
	if id == "" || e.TenantID == "" || e.CreditType == "" {
		slog.Error("usage event missing required fields — terminating", "id", id)
		_ = msg.Term()
		return
	}
	amt, err := domain.ParseMoney(e.Units)
	if err != nil || amt.Sign() < 0 {
		slog.Error("usage event has bad units — terminating", "id", id, "units", e.Units)
		_ = msg.Term()
		return
	}
	sub := ""
	if e.SubAccountID != nil {
		sub = *e.SubAccountID
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := c.st.ApplyMovement(ctx, store.Movement{
		TenantID: e.TenantID, SubAccountID: sub, CreditType: domain.CreditType(e.CreditType),
		Operation: domain.OpConsumption, Amount: domain.Zero().Sub(amt), // debit = negative delta
		ReferenceID: id, IdempotencyKey: id, IsPaper: e.IsPaper,
	})
	if errors.Is(err, domain.ErrInsufficientCredit) {
		// The producer's pre-flight (e.g. the gateway's 402) should prevent this; if it slips through
		// we don't bill and don't redeliver forever. The work was done but is unbilled (logged for
		// reconciliation).
		slog.Warn("usage debit skipped: insufficient credit", "id", id, "tenant_id", e.TenantID)
		_ = msg.Ack()
		return
	}
	if err != nil {
		slog.Error("usage debit failed — will redeliver", "id", id, "err", err)
		_ = msg.Nak()
		return
	}
	c.pub.PublishTx(tx)
	_ = msg.Ack()
}

// Close stops the fetch loop, then drains the subscription and connection.
func (c *UsageConsumer) Close() {
	if c.stop != nil {
		close(c.stop)
	}
	if c.sub != nil {
		_ = c.sub.Drain()
	}
	if c.nc != nil {
		_ = c.nc.Drain()
	}
}
