// Package metrics holds credit-ledger domain metrics (Prometheus). HTTP RED metrics live in
// internal/obs (copied verbatim across services); this is the business signal — credits actually
// moved — and is service-specific, so it lives on its own. Both export on the same /metrics endpoint
// via the default registry.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// TransactionsTotal counts ledger transactions actually written, labelled by operation and credit
// type. It is incremented at insert time, so idempotent replays — which return before the insert —
// are excluded. Cardinality is bounded by the operation set × the credit-types enum.
var TransactionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "ledger_transactions_total",
	Help: "Credit-ledger transactions written, by operation and credit type.",
}, []string{"operation", "credit_type"})
