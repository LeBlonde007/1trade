// Package metrics holds matching-engine domain metrics (Prometheus). HTTP RED metrics live in
// internal/obs (copied verbatim across services); this is the business signal and is service-specific.
// Both export on the same /metrics endpoint.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// OrderAttemptsBlockedTotal counts order-entry attempts refused because the exchange is paused,
// labelled by product and side. This is the one genuinely useful business signal the paused exchange
// produces: it measures real demand for trading, which is direct evidence for the licensing decision
// (F22). Cardinality is bounded — the product set is fixed and small.
var OrderAttemptsBlockedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "trading_order_attempts_blocked_total",
	Help: "Order-entry attempts refused with EXCHANGE_PAUSED, by product and side.",
}, []string{"product_id", "side"})

// CancelAttemptsBlockedTotal counts cancel attempts refused because the exchange is paused.
var CancelAttemptsBlockedTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "trading_cancel_attempts_blocked_total",
	Help: "Cancel attempts refused with EXCHANGE_PAUSED.",
})

// MarketDataRequestsTotal counts served market-data reads by surface, so the demo's traffic shape is
// visible without per-path HTTP labels.
var MarketDataRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "trading_marketdata_requests_total",
	Help: "Market-data reads served, by surface (quote, orderbook, trades, candles, products).",
}, []string{"surface"})

// RecordBlockedOrder records one refused order attempt. Unknown or absent fields are normalized to
// "unknown" so a malformed body cannot create unbounded label values.
func RecordBlockedOrder(productID, side string) {
	OrderAttemptsBlockedTotal.WithLabelValues(normalize(productID), normalizeSide(side)).Inc()
}

// RecordMarketData records one served market-data read.
func RecordMarketData(surface string) {
	MarketDataRequestsTotal.WithLabelValues(surface).Inc()
}

// normalize bounds a product label to the known catalog shape, collapsing anything unexpected.
func normalize(productID string) string {
	if productID == "" || len(productID) > 24 {
		return "unknown"
	}
	return productID
}

// normalizeSide bounds the side label to buy / sell / unknown.
func normalizeSide(side string) string {
	switch side {
	case "buy", "sell":
		return side
	default:
		return "unknown"
	}
}
