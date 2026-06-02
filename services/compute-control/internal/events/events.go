// Package events publishes compute.usage.v1 to NATS JetStream — one event per metered GPU interval,
// consumed by credit-ledger (drives the gpu_* debit, idempotent on usage_id) and settlement-trust
// (payout attribution via supply_source_id). Mirrors events/compute.usage.v1.yaml.
package events

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// Subject is the NATS subject for billable GPU usage.
const Subject = "compute.usage.v1"

// ComputeUsage is the compute.usage.v1 payload.
type ComputeUsage struct {
	UsageID        string  `json:"usage_id"`
	TenantID       string  `json:"tenant_id"`
	SubAccountID   *string `json:"sub_account_id"`
	InstanceID     string  `json:"instance_id"`
	CreditType     string  `json:"credit_type"`
	GPUSeconds     string  `json:"gpu_seconds"`
	Units          string  `json:"units"`
	Reserved       bool    `json:"reserved"`
	SupplySourceID string  `json:"supply_source_id"`
	IsPaper        bool    `json:"is_paper"`
	TS             string  `json:"ts"`
}

// Publisher emits billable GPU usage.
type Publisher interface {
	PublishUsage(u ComputeUsage)
}

// NewTimestamp returns the current time formatted for the event's ts field.
func NewTimestamp() string { return time.Now().UTC().Format(time.RFC3339) }

// NoopPublisher logs instead of publishing (used when NATS is unconfigured / in tests).
type NoopPublisher struct{}

// PublishUsage logs the usage event.
func (NoopPublisher) PublishUsage(u ComputeUsage) {
	slog.Info("compute.usage.v1 (noop)", "usage_id", u.UsageID, "units", u.Units, "credit_type", u.CreditType)
}

// NATSPublisher publishes to JetStream. Connect with Open; Close to drain.
type NATSPublisher struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

// Open connects to NATS and returns a JetStream publisher.
func Open(url string) (*NATSPublisher, error) {
	nc, err := nats.Connect(url, nats.Name("compute-control"), nats.Timeout(5*time.Second))
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}
	return &NATSPublisher{nc: nc, js: js}, nil
}

// PublishUsage marshals + publishes one usage event. Failures are logged, never fatal — metering is
// best-effort here; the durable record + idempotent debit live in the ledger consumer.
func (p *NATSPublisher) PublishUsage(u ComputeUsage) {
	b, err := json.Marshal(u)
	if err != nil {
		slog.Error("marshal compute.usage.v1", "err", err)
		return
	}
	if _, err := p.js.Publish(Subject, b); err != nil {
		slog.Error("publish compute.usage.v1", "err", err, "usage_id", u.UsageID)
	}
}

// Close drains the connection.
func (p *NATSPublisher) Close() {
	if p.nc != nil {
		_ = p.nc.Drain()
	}
}
