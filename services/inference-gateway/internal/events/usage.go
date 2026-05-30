// Package events emits inference.usage.v1 — one event per served request, after the response
// completes. credit-ledger consumes it to debit the sub-credit (idempotent on request_id). Matches
// docs/contracts/events/inference.usage.v1.yaml.
package events

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
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
	PublishUsage(e UsageEvent) error
}

// NatsPublisher publishes inference.usage.v1 to NATS.
type NatsPublisher struct {
	nc *nats.Conn
}

// NewNatsPublisher connects to NATS for usage publication.
func NewNatsPublisher(url string) (*NatsPublisher, error) {
	nc, err := nats.Connect(url, nats.Name("inference-gateway"), nats.Timeout(5*time.Second))
	if err != nil {
		return nil, err
	}
	return &NatsPublisher{nc: nc}, nil
}

// PublishUsage publishes one inference.usage.v1 event.
func (p *NatsPublisher) PublishUsage(e UsageEvent) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return p.nc.Publish(subject, b)
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
func (LogPublisher) PublishUsage(e UsageEvent) error {
	b, _ := json.Marshal(e)
	slog.Info("inference.usage.v1 (log fallback)", "event", string(b))
	return nil
}
