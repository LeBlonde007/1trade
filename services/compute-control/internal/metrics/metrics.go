// Package metrics holds compute-control domain metrics (Prometheus). HTTP RED metrics live in
// internal/obs (copied verbatim across services); this is the business signal — billable GPU usage —
// and is service-specific. Both export on the same /metrics endpoint.
package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// GPUSecondsTotal counts metered GPU-seconds, by credit type and whether the capacity is reserved.
var GPUSecondsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "compute_gpu_seconds_total",
	Help: "Metered GPU-seconds, by credit type and reservation.",
}, []string{"credit_type", "reserved"})

// UsageEventsTotal counts compute.usage.v1 events emitted, by credit type.
var UsageEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "compute_usage_events_total",
	Help: "compute.usage.v1 events emitted, by credit type.",
}, []string{"credit_type"})

// RecordComputeUsage records one metered interval: a usage-event count and the GPU-seconds (parsed
// best-effort from the fixed-point string — metrics are advisory, not the billing source of truth),
// labelled by credit type + reservation. Called at each compute.usage.v1 emit site.
func RecordComputeUsage(creditType, gpuSeconds string, reserved bool) {
	UsageEventsTotal.WithLabelValues(creditType).Inc()
	if secs, err := strconv.ParseFloat(gpuSeconds, 64); err == nil && secs > 0 {
		GPUSecondsTotal.WithLabelValues(creditType, strconv.FormatBool(reserved)).Add(secs)
	}
}
