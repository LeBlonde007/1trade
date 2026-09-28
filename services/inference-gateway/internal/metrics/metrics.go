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

// ModelLoadSeconds is how long a pooled model took to become ready after a cold start (F11) — the
// measured cold-start latency.
var ModelLoadSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "inference_model_load_seconds",
	Help:    "Time from starting a pooled model's runtime process to it answering ready.",
	Buckets: []float64{0.5, 1, 2, 5, 10, 20, 30, 60, 120, 300, 600},
}, []string{"model"})

// ModelLoadsTotal counts pooled model loads by outcome (ok | failed).
var ModelLoadsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "inference_model_loads_total",
	Help: "Pooled model loads, by model and outcome.",
}, []string{"model", "result"})

// ModelEvictionsTotal counts replicas stopped to make room for another model.
var ModelEvictionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "inference_model_evictions_total",
	Help: "Pooled model replicas evicted to make room, by model.",
}, []string{"model"})
