// Package pipeline feeds engine events to the detectors and delivers what they raise: every alert is
// stored first (the record of truth), and published on surveillance.alert.v1 only if it was new — so
// replaying the stream after a restart rebuilds detector state without re-announcing anything.
package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/trade1/surveillance/internal/detect"
	"github.com/trade1/surveillance/internal/metrics"
)

// Subjects consumed and produced (docs/contracts/events).
const (
	SubjectOrders = "orders.state.v1"
	SubjectTrades = "trades.executed.v1"
	SubjectAlerts = "surveillance.alert.v1"
)

// Store saves an alert and reports whether it was new (false for an alert already stored).
type Store interface {
	SaveAlert(ctx context.Context, a detect.Alert) (bool, error)
}

// Publisher announces a new alert.
type Publisher interface {
	PublishAlert(a detect.Alert) error
}

// Pipeline owns one detector. Feed it one ordered stream from a single goroutine.
type Pipeline struct {
	det   *detect.Detector
	store Store
	pub   Publisher
}

// New wires a detector to its sinks.
func New(cfg detect.Config, st Store, pub Publisher) *Pipeline {
	return &Pipeline{det: detect.New(cfg), store: st, pub: pub}
}

// Handle processes one message. A payload that does not decode is skipped with an error (a poison
// message must not stop surveillance of everything after it); a store failure is returned so the
// caller can retry the same message — the detector already deduplicates the redelivery, and the
// alert id is deterministic, so a retry re-saves the same alert.
func (p *Pipeline) Handle(ctx context.Context, subject string, data []byte) (int, error) {
	var alerts []detect.Alert
	switch subject {
	case SubjectOrders:
		var o detect.Order
		if err := json.Unmarshal(data, &o); err != nil {
			metrics.EventsTotal.WithLabelValues(subject, "undecodable").Inc()
			return 0, fmt.Errorf("pipeline: undecodable %s: %w", subject, err)
		}
		alerts = p.det.OnOrder(o)
	case SubjectTrades:
		var t detect.Trade
		if err := json.Unmarshal(data, &t); err != nil {
			metrics.EventsTotal.WithLabelValues(subject, "undecodable").Inc()
			return 0, fmt.Errorf("pipeline: undecodable %s: %w", subject, err)
		}
		alerts = p.det.OnTrade(t)
	default:
		return 0, nil
	}
	published := 0
	for _, a := range alerts {
		isNew, err := p.store.SaveAlert(ctx, a)
		if err != nil {
			metrics.EventsTotal.WithLabelValues(subject, "error").Inc()
			return published, fmt.Errorf("pipeline: save %s: %w", a.AlertID, err)
		}
		if !isNew {
			continue // raised before this restart; already announced
		}
		metrics.AlertsTotal.WithLabelValues(a.Rule, a.Severity).Inc()
		if err := p.pub.PublishAlert(a); err != nil {
			// Stored is what counts; the review API serves it. Publishing is best-effort.
			return published, fmt.Errorf("pipeline: publish %s (stored): %w", a.AlertID, err)
		}
		published++
	}
	metrics.EventsTotal.WithLabelValues(subject, "ok").Inc()
	return published, nil
}
