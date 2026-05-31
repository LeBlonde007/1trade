// Package consumer subscribes to inference.usage.v1 and debits the ledger once per served request.
// The debit reuses store.ApplyMovement and is idempotent on request_id, so JetStream's at-least-once
// delivery can never double-bill. This is the consumer side of
// docs/contracts/events/inference.usage.v1.yaml (compute.usage.v1 will follow the same shape).
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

const (
	subject    = "inference.usage.v1"
	streamName = "INFERENCE_USAGE"
	durable    = "credit-ledger-debit"
)

// usageEvent mirrors the inference.usage.v1 payload (the fields the debit needs).
type usageEvent struct {
	RequestID    string  `json:"request_id"`
	TenantID     string  `json:"tenant_id"`
	SubAccountID *string `json:"sub_account_id"`
	CreditType   string  `json:"credit_type"`
	Units        string  `json:"units"`
	IsPaper      bool    `json:"is_paper"`
}

// UsageConsumer holds the durable JetStream pull subscription that debits the ledger on inference
// usage, plus the fetch loop's stop signal.
type UsageConsumer struct {
	nc   *nats.Conn
	sub  *nats.Subscription
	st   *store.Store
	pub  events.Publisher
	stop chan struct{}
}

// Start connects to NATS/JetStream, ensures the stream capturing inference.usage.v1 exists, and
// begins consuming with a durable PULL subscription (a fetch loop). Pull is deliberate: a push
// durable is exclusive — one active subscription at a time — so a pod restart races the server's
// still-"bound" old deliver subject and fails with "consumer is already bound to a subscription",
// silently stopping all debits. A pull consumer rebinds cleanly across restarts. Close to stop.
func Start(url string, st *store.Store, pub events.Publisher) (*UsageConsumer, error) {
	nc, err := nats.Connect(url, nats.Name("credit-ledger-consumer"), nats.Timeout(5*time.Second))
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	// Ensure the stream exists (create once; tolerate an already-created one). It must exist before
	// the gateway publishes, or core-published events aren't captured.
	if _, err := js.StreamInfo(streamName); err != nil {
		if _, err := js.AddStream(&nats.StreamConfig{
			Name: streamName, Subjects: []string{subject},
			Storage: nats.FileStorage, MaxAge: 14 * 24 * time.Hour,
		}); err != nil {
			nc.Close()
			return nil, fmt.Errorf("add stream: %w", err)
		}
	}
	// Migrate a legacy push-based durable (from the old js.Subscribe path) to pull: a pull
	// subscription cannot bind to a consumer that has a DeliverSubject, so delete it and let
	// PullSubscribe recreate it. The old consumer never acked (it failed to bind), so no progress is
	// lost; the debit is idempotent on request_id regardless.
	if info, err := js.ConsumerInfo(streamName, durable); err == nil && info.Config.DeliverSubject != "" {
		_ = js.DeleteConsumer(streamName, durable)
	}
	c := &UsageConsumer{nc: nc, st: st, pub: pub, stop: make(chan struct{})}
	sub, err := js.PullSubscribe(subject, durable, nats.AckExplicit(), nats.MaxDeliver(5))
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("pull subscribe: %w", err)
	}
	c.sub = sub
	go c.loop()
	slog.Info("consuming inference.usage.v1 → ledger debit (pull)", "stream", streamName, "durable", durable)
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

// handle debits the ledger for one usage event. Idempotent on request_id, so a redelivery is a
// no-op. Unparseable/invalid messages are terminated (poison); transient store errors are NAK'd.
func (c *UsageConsumer) handle(msg *nats.Msg) {
	var e usageEvent
	if err := json.Unmarshal(msg.Data, &e); err != nil {
		slog.Error("usage event unparseable — terminating", "err", err)
		_ = msg.Term()
		return
	}
	if e.RequestID == "" || e.TenantID == "" || e.CreditType == "" {
		slog.Error("usage event missing required fields — terminating", "request_id", e.RequestID)
		_ = msg.Term()
		return
	}
	amt, err := domain.ParseMoney(e.Units)
	if err != nil || amt.Sign() < 0 {
		slog.Error("usage event has bad units — terminating", "request_id", e.RequestID, "units", e.Units)
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
		ReferenceID: e.RequestID, IdempotencyKey: e.RequestID, IsPaper: e.IsPaper,
	})
	if errors.Is(err, domain.ErrInsufficientCredit) {
		// The gateway's pre-flight 402 should prevent this; if it slips through we don't bill and
		// don't redeliver forever. The request was served but is unbilled (logged for reconciliation).
		slog.Warn("usage debit skipped: insufficient credit", "request_id", e.RequestID, "tenant_id", e.TenantID)
		_ = msg.Ack()
		return
	}
	if err != nil {
		slog.Error("usage debit failed — will redeliver", "request_id", e.RequestID, "err", err)
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
