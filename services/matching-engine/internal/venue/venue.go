// Package venue is the live paper exchange: the matching engine plus the read models the trading API
// serves (a tenant's orders, fills and positions; each product's prints, candles and 24h summary).
//
// Every command goes through the Venue, which applies it to the engine and folds the resulting events
// into the view under one lock, so a read never sees an order the view has not caught up with. On
// start the view is rebuilt by replaying the journal (Recover), so it needs no storage of its own and
// can never drift from the journal.
//
// Only paper books are served. Real-money order entry stays refused at the API (licence-gated, F22);
// nothing here opens it.
package venue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/trade1/matching-engine/internal/engine"
	"github.com/trade1/matching-engine/internal/journal"
)

// Venue owns the engine and its view.
type Venue struct {
	mu   sync.Mutex
	eng  *engine.Engine
	view *View
}

// New wraps an engine whose history is events (what engine.Replay returned) and journal (its commands,
// for each order's time in force). liquidityTenant is the paper liquidity account: its orders and
// fills are kept out of the per-tenant views (it is not a customer), but its trades are market prints.
func New(eng *engine.Engine, journal []engine.Command, events []engine.Event, liquidityTenant string) *Venue {
	v := &Venue{eng: eng, view: newView(liquidityTenant)}
	for _, c := range journal {
		if c.Submit != nil {
			v.view.tif[c.Submit.OrderID] = c.Submit.TIF
		}
	}
	v.view.apply(events)
	return v
}

// Recover loads and verifies the journal, replays it into an engine with cfg (the store becomes the
// engine's write-ahead hook, and the epoch comes from the journal), and builds the view from the
// replayed events. It is journal.Recover plus the view; call it before serving.
func Recover(ctx context.Context, s *journal.Store, cfg engine.Config, liquidityTenant string) (*Venue, engine.Config, error) {
	epoch, err := s.Epoch(ctx)
	if err != nil {
		return nil, cfg, err
	}
	cmds, err := s.Load(ctx)
	if err != nil {
		return nil, cfg, err
	}
	cfg.Epoch, cfg.Persist = epoch, s.Append
	eng, evs, err := engine.Replay(cfg, cmds)
	if err != nil {
		return nil, cfg, fmt.Errorf("venue: replay: %w", err)
	}
	return New(eng, cmds, evs, liquidityTenant), cfg, nil
}

// Submit places an order and folds its events into the view.
func (v *Venue) Submit(c engine.SubmitCmd) (engine.Result, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	r, err := v.eng.Submit(c)
	if err != nil {
		return r, err
	}
	if !r.Duplicate {
		v.view.tif[r.Order.OrderID] = r.Order.TIF
	}
	v.view.apply(r.Events)
	return r, nil
}

// Cancel cancels a tenant's open order and folds the event into the view.
func (v *Venue) Cancel(c engine.CancelCmd) (engine.Result, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	r, err := v.eng.Cancel(c)
	if err != nil {
		return r, err
	}
	v.view.apply(r.Events)
	return r, nil
}

// ExpireDay cancels every resting day order (the session boundary) and folds the events in.
func (v *Venue) ExpireDay(ts time.Time) (int, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	evs, err := v.eng.ExpireDay(engine.ExpireDayCmd{TS: ts})
	if err != nil {
		return 0, err
	}
	v.view.apply(evs)
	return len(evs), nil
}

// Void tombstones an order id for the reservation reconciler (settle.Voider). A void emits no event.
func (v *Venue) Void(c engine.VoidCmd) (engine.Order, bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.eng.Void(c)
}

// Voided lists tombstoned orders, for the reconciler's resume after recovery.
func (v *Venue) Voided() []engine.Order { return v.eng.Voided() }

// Order returns a tenant's order as the engine holds it.
func (v *Venue) Order(orderID, tenantID string) (engine.Order, bool) {
	return v.eng.Order(orderID, tenantID)
}

// OpenOrders returns a tenant's resting orders on paper books.
func (v *Venue) OpenOrders(tenantID string) []engine.Order { return v.eng.OpenOrders(tenantID, true) }

// Depth returns a paper book's aggregated levels, best first.
func (v *Venue) Depth(productID string, n int) (bids, asks []engine.DepthLevel) {
	return v.eng.Depth(productID, true, n)
}

// Read runs fn against the view under the venue lock, so it sees a consistent snapshot.
func (v *Venue) Read(fn func(*View)) {
	v.mu.Lock()
	defer v.mu.Unlock()
	fn(v.view)
}
