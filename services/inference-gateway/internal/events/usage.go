// Package events emits inference.usage.v1 — one event per served request, after the response
// completes. credit-ledger consumes it to debit the sub-credit (idempotent on request_id). Matches
// docs/contracts/events/inference.usage.v1.yaml.
package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/trade1/inference-gateway/internal/obs"
)

// subject is the NATS subject for inference usage (versioned).
const subject = "inference.usage.v1"

// UsageEvent is the inference.usage.v1 payload. `Units` is a fixed-point decimal string; nullable
// fields are pointers so they serialize as JSON null when unset (per the schema).
type UsageEvent struct {
	RequestID      string  `json:"request_id"`
	TenantID       string  `json:"tenant_id"`
	SubAccountID   *string `json:"sub_account_id"`
	Model          string  `json:"model"`
	Modality       string  `json:"modality"`
	CreditType     string  `json:"credit_type"`
	InputTokens    *int    `json:"input_tokens"`
	OutputTokens   *int    `json:"output_tokens"`
	Units          string  `json:"units"`
	LatencyMS      *int    `json:"latency_ms"`
	ServedByPod    *string `json:"served_by_pod"`
	SupplySourceID *string `json:"supply_source_id"`
	IsPaper        bool    `json:"is_paper"`
	TS             string  `json:"ts"`
}

// Publisher emits usage events.
type Publisher interface {
	PublishUsage(ctx context.Context, e UsageEvent) error
}

// Stream is the JetStream stream that captures inference.usage.v1 for the ledger's debit consumer.
// Its config must match credit-ledger's consumer.Inference source: either side may create it first.
var Stream = nats.StreamConfig{
	Name: "INFERENCE_USAGE", Subjects: []string{subject}, Storage: nats.FileStorage, MaxAge: 14 * 24 * time.Hour,
}

// NatsPublisher publishes inference.usage.v1 to NATS.
type NatsPublisher struct {
	nc      *nats.Conn
	ensured atomic.Bool // the capturing stream is known to exist
}

// NewNatsPublisher connects to NATS for usage publication. NATS being down at boot is not fatal: the
// connection keeps retrying in the background and buffers publishes until it is up. The old behaviour
// (fall back to logging for the life of the process) silently stopped every debit. Only a malformed
// URL returns an error.
func NewNatsPublisher(url string) (*NatsPublisher, error) {
	nc, err := nats.Connect(url, nats.Name("inference-gateway"), nats.Timeout(5*time.Second),
		nats.RetryOnFailedConnect(true), nats.MaxReconnects(-1), nats.ReconnectWait(time.Second),
		nats.ConnectHandler(func(*nats.Conn) { slog.Info("usage publisher connected to NATS") }),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			slog.Warn("usage publisher disconnected from NATS; buffering events", "err", err)
		}))
	if err != nil {
		return nil, err
	}
	p := &NatsPublisher{nc: nc}
	p.ensureStream()
	return p, nil
}

// ensureStream creates the capturing stream if it is missing, so events published before the ledger
// first starts are kept rather than dropped. Best effort: while NATS is down it fails fast and is
// retried on the next publish.
func (p *NatsPublisher) ensureStream() {
	if p.ensured.Load() || !p.nc.IsConnected() {
		return
	}
	js, err := p.nc.JetStream(nats.MaxWait(time.Second))
	if err != nil {
		return
	}
	if _, err := js.StreamInfo(Stream.Name); err != nil {
		cfg := Stream
		if _, err := js.AddStream(&cfg); err != nil {
			slog.Warn("cannot ensure the usage stream; will retry", "stream", Stream.Name, "err", err)
			return
		}
	}
	p.ensured.Store(true)
}

// PublishUsage publishes one inference.usage.v1 event. The request's trace context rides in the
// message headers, so the ledger's debit joins the request's trace.
func (p *NatsPublisher) PublishUsage(ctx context.Context, e UsageEvent) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	p.ensureStream()
	msg := nats.NewMsg(subject)
	msg.Data = b
	obs.InjectHeader(ctx, msg.Header)
	return p.nc.PublishMsg(msg)
}

// Close drains and closes the connection.
func (p *NatsPublisher) Close() {
	if p.nc != nil {
		_ = p.nc.Drain()
	}
}

// LogPublisher is the fallback when NATS is unavailable: it logs the event so local dev still sees
// usage (the debit won't happen until a real bus is wired, which is acceptable in dev).
type LogPublisher struct{}

// PublishUsage logs the event as JSON.
func (LogPublisher) PublishUsage(_ context.Context, e UsageEvent) error {
	b, _ := json.Marshal(e)
	slog.Info("inference.usage.v1 (log fallback)", "event", string(b))
	return nil
}
