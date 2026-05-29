// Package events publishes ledger events (credit.tx.v1). The publish is downstream of the committed
// DB transaction (the ledger row is the source of truth), so a publish failure never fails a request.
// A no-op LogPublisher is used until the NATS publisher is wired (F05 follow-up; see credit.tx.v1.yaml).
package events

import (
	"log/slog"

	"github.com/exascale/credit-ledger/internal/domain"
)

// Publisher emits ledger events.
type Publisher interface {
	// PublishTx emits one credit.tx.v1 event for a committed transaction. Best-effort.
	PublishTx(t domain.Transaction)
}

// LogPublisher is the default no-op publisher: it only logs (NATS publisher lands next).
type LogPublisher struct{}

// PublishTx logs the event at debug level.
func (LogPublisher) PublishTx(t domain.Transaction) {
	slog.Debug("credit.tx.v1",
		"tx_id", t.TxID, "tenant_id", t.TenantID, "credit_type", t.CreditType,
		"operation", t.Operation, "amount", t.Amount.String(), "is_paper", t.IsPaper)
}
