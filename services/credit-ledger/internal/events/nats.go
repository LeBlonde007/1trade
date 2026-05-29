package events

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/exascale/credit-ledger/internal/domain"
	"github.com/nats-io/nats.go"
)

// NatsPublisher emits credit.tx.v1 to NATS. Publishing is best-effort: the committed ledger row is
// the source of truth, so a publish failure is logged, never propagated to the request.
type NatsPublisher struct {
	nc      *nats.Conn
	subject string
}

// NewNatsPublisher connects to NATS (credit.tx.v1 subject).
func NewNatsPublisher(url string) (*NatsPublisher, error) {
	nc, err := nats.Connect(url, nats.Name("credit-ledger"), nats.Timeout(5*time.Second))
	if err != nil {
		return nil, err
	}
	return &NatsPublisher{nc: nc, subject: "credit.tx.v1"}, nil
}

// PublishTx publishes one credit.tx.v1 event matching docs/contracts/events/credit.tx.v1.yaml.
func (p *NatsPublisher) PublishTx(t domain.Transaction) {
	b, err := json.Marshal(map[string]any{
		"tx_id":          t.TxID,
		"tenant_id":      t.TenantID,
		"sub_account_id": nilIfEmpty(t.SubAccountID),
		"credit_type":    string(t.CreditType),
		"operation":      string(t.Operation),
		"amount":         t.Amount.String(),
		"reference_id":   nilIfEmpty(t.ReferenceID),
		"balance_after":  t.BalanceAfter.String(),
		"is_paper":       t.IsPaper,
		"created_at":     t.CreatedAt.UTC().Format(time.RFC3339Nano),
		"chain_hash":     t.ChainHash,
	})
	if err != nil {
		slog.Error("marshal credit.tx.v1", "err", err)
		return
	}
	if err := p.nc.Publish(p.subject, b); err != nil {
		slog.Error("publish credit.tx.v1", "err", err, "tx_id", t.TxID)
	}
}

// Close drains and closes the NATS connection.
func (p *NatsPublisher) Close() { _ = p.nc.Drain() }

// nilIfEmpty returns nil for "" so the field serialises as JSON null (the event schema allows null).
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
