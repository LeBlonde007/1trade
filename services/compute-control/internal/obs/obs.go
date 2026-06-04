// Package obs is the service's observability surface: a Prometheus /metrics endpoint and a low-
// cardinality HTTP instrumentation middleware (request rate, errors, duration + an in-flight gauge).
// Labels are method + status code only — never per-request paths — so series cardinality stays
// bounded regardless of ids in the URL. This package has no service-specific code: it is copied
// verbatim into each Go service so every module stays self-contained (no shared-module coupling).
package obs

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// requests counts HTTP responses by method + status code.
var requests = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "http_requests_total",
	Help: "Total HTTP requests handled, by method and response code.",
}, []string{"method", "code"})

// duration observes HTTP request latency (seconds) by method + status code.
var duration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "http_request_duration_seconds",
	Help:    "HTTP request latency in seconds, by method and response code.",
	Buckets: prometheus.DefBuckets,
}, []string{"method", "code"})

// inFlight gauges concurrently-served HTTP requests.
var inFlight = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "http_requests_in_flight",
	Help: "HTTP requests currently being served.",
})

// Handler returns the Prometheus exposition handler for /metrics. The default registry already
// carries Go-runtime and process collectors, so memory/GC/fd/CPU metrics ship for free.
func Handler() http.Handler { return promhttp.Handler() }

// Instrument wraps an http.Handler with RED metrics (rate, errors, duration) plus an in-flight gauge.
// Cardinality is intentionally bounded to method × status code — no path label, which an id would
// blow up.
func Instrument(next http.Handler) http.Handler {
	return promhttp.InstrumentHandlerInFlight(inFlight,
		promhttp.InstrumentHandlerDuration(duration,
			promhttp.InstrumentHandlerCounter(requests, next)))
}
