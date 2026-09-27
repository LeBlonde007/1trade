package settle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/trade1/matching-engine/internal/engine"
)

// Ledger is what the worker needs from credit-ledger. *Client implements it; tests fake it.
type Ledger interface {
	Settle(ctx context.Context, t engine.Trade) error
	Release(ctx context.Context, orderID string) error
}

// Worker applies engine output to the ledger strictly in emission order: each trade is settled, and
// each order that closes with a reservation has its remainder released. Order matters — an order's
// terminal event is always emitted after its trades, so processing in order guarantees every fill is
// settled (consuming the reservation) before the leftover is released.
type Worker struct {
	Ledger Ledger
	// Attempts bounds retries of a transient failure per call (default 5); Backoff is the first
	// delay, doubled each retry (default 100ms).
	Attempts int
	Backoff  time.Duration
	// OnUnsettleable is told about a trade the ledger refused for good. With reservations in place
	// this should never happen; it is an alert, not a normal path.
	OnUnsettleable func(t engine.Trade, err error)
}

// ErrHalt wraps a failure the worker must not skip past (a conflict, or transient failures that
// outlasted every retry). The caller must stop and resume from the failed event: skipping it would
// settle later trades out of order or release a reservation before its fills settle.
var ErrHalt = errors.New("settle: worker halted")

// Process applies events in order. It returns the index of the first event it could not apply,
// wrapped in ErrHalt, or (len(evs), nil) when everything applied. Safe to re-run from any index:
// settle and release are both idempotent.
func (w *Worker) Process(ctx context.Context, evs []engine.Event) (int, error) {
	for i, ev := range evs {
		var err error
		switch {
		case ev.Trade != nil:
			err = w.retry(ctx, func(ctx context.Context) error { return w.Ledger.Settle(ctx, *ev.Trade) })
			if errors.Is(err, ErrUnsettleable) {
				if w.OnUnsettleable != nil {
					w.OnUnsettleable(*ev.Trade, err)
				}
				slog.Error("trade refused by ledger despite reservation", "trade_id", ev.Trade.TradeID, "err", err)
				err = nil // recorded and alerted; the stream continues
			}
		case ev.Order != nil && ev.Order.Held && terminal(ev.Order.State):
			id := ev.Order.OrderID
			err = w.retry(ctx, func(ctx context.Context) error { return w.Ledger.Release(ctx, id) })
		}
		if err != nil {
			return i, fmt.Errorf("%w at event %d: %w", ErrHalt, i, err)
		}
	}
	return len(evs), nil
}

// terminal reports whether an order state ends the order (its reservation can be released).
func terminal(s engine.State) bool {
	return s == engine.Filled || s == engine.Cancelled || s == engine.Rejected
}

// retry runs call until it succeeds, fails permanently (unsettleable / conflict), or exhausts the
// attempts on transient errors, backing off exponentially.
func (w *Worker) retry(ctx context.Context, call func(context.Context) error) error {
	attempts, delay := w.Attempts, w.Backoff
	if attempts <= 0 {
		attempts = 5
	}
	if delay <= 0 {
		delay = 100 * time.Millisecond
	}
	var err error
	for a := 1; ; a++ {
		if err = call(ctx); err == nil || errors.Is(err, ErrUnsettleable) || errors.Is(err, ErrConflict) || a >= attempts {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w (last error: %w)", ctx.Err(), err)
		case <-time.After(delay):
		}
		delay *= 2
	}
}
