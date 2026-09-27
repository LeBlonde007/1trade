// Package metrics holds surveillance's domain metrics. Labels are bounded enums (subject, rule,
// severity) — never tenant or product ids, which would make series cardinality unbounded.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// EventsTotal counts engine events processed, by subject and outcome (ok | undecodable | error).
var EventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "surveillance_events_total",
	Help: "Engine events processed by surveillance, by subject and outcome.",
}, []string{"subject", "outcome"})

// AlertsTotal counts NEW alerts raised (replays of stored alerts are not counted), by rule and severity.
var AlertsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "surveillance_alerts_total",
	Help: "New surveillance alerts raised, by rule and severity.",
}, []string{"rule", "severity"})
