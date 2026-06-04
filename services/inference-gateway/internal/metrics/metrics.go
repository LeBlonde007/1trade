// Package metrics holds inference-gateway domain metrics (Prometheus). HTTP RED metrics live in
// internal/obs (copied verbatim across services); this is the business signal — inference served and
// tokens processed — and is service-specific. Both export on the same /metrics endpoint.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// RequestsTotal counts served inference requests, by model and modality.
var RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "inference_requests_total",
	Help: "Served inference requests, by model and modality.",
}, []string{"model", "modality"})

// TokensTotal counts processed tokens, by model and direction (input|output).
var TokensTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "inference_tokens_total",
	Help: "Processed inference tokens, by model and direction.",
}, []string{"model", "direction"})

// RecordInference records one served request: its model/modality and input/output token counts.
// Called once per request at metering time (after the response completes).
func RecordInference(model, modality string, inputTokens, outputTokens int) {
	RequestsTotal.WithLabelValues(model, modality).Inc()
	if inputTokens > 0 {
		TokensTotal.WithLabelValues(model, "input").Add(float64(inputTokens))
	}
	if outputTokens > 0 {
		TokensTotal.WithLabelValues(model, "output").Add(float64(outputTokens))
	}
}
