package settle

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/trade1/matching-engine/internal/engine"
)

// Voider is the engine surface the reconciler needs; *engine.Engine implements it.
type Voider interface {
	Void(c engine.VoidCmd) (engine.Order, bool, error)
}

// Releaser frees what an order's reservation still holds; *Client implements it (a 404, meaning
// nothing was reserved, counts as success).
type Releaser interface {
	Release(ctx context.Context, orderID string) error
}

// Reconciler frees reservations stranded by a submit whose journal write failed after the ledger had
// already reserved the order's hold (SPEC.md §7.3).
//
// The engine reports each such submit (engine.Config.OnUnjournaled → Suspect). Once Grace has passed,
// the reconciler voids the order_id in the engine (a durable tombstone), then releases the reservation:
//   - During the grace period, a client retry with the same order_id reuses the reservation and the
//     order goes ahead. The reconciler then finds the order and leaves its reservation alone.
//   - Once the id is void, no submit can claim the reservation, so the release cannot race an
//     acceptance, and it is retried until the ledger confirms it.
//   - The void goes through the journal. While the journal is down it fails and the suspect waits for
//     the next pass. If the durable journal is ahead of this engine (a write that succeeded without
//     the engine seeing it), the void collides on seq and is refused. So a reservation the journal
//     relies on is never released.
//
// Suspects are held in memory. A tombstone survives a crash and is re-released after recovery (Resume),
// but a suspect lost before its void is not recovered; finding those needs the ledger to list open
// reservations (proposed as credit.yaml v1.3, SPEC.md §8).
type Reconciler struct {
	Engine Voider
	Ledger Releaser
	// Grace is how long a suspect waits before its id is voided (default 2 minutes).
	Grace time.Duration
	// Timeout bounds each release call (default 5 seconds).
	Timeout time.Duration
	// Now is the clock (nil means time.Now).
	Now func() time.Time

	mu       sync.Mutex
	suspects map[string]suspect
}

// suspect is one order_id whose reservation may be orphaned.
type suspect struct {
	tenantID string
	isPaper  bool
	since    time.Time
}

// Report summarises one reconciliation pass.
type Report struct {
	Voided   int // order ids tombstoned this pass
	Released int // reservations freed (voided or rejected-at-entry orders)
	Kept     int // the order exists after all (a retry reused the reservation), or another tenant owns the id
	Waiting  int // still inside the grace period
	Failed   int // the void or the release failed; retried next pass
}

// now returns the reconciler's clock.
func (r *Reconciler) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

// Suspect records an order whose reservation may be orphaned. Its signature matches
// engine.Config.OnUnjournaled. It is safe to call with the engine lock held, because it takes only the
// reconciler's own lock. A repeat report keeps the first time, so the grace period runs from it.
func (r *Reconciler) Suspect(o engine.Order) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.suspects == nil {
		r.suspects = map[string]suspect{}
	}
	if _, ok := r.suspects[o.OrderID]; !ok {
		r.suspects[o.OrderID] = suspect{tenantID: o.TenantID, isPaper: o.IsPaper, since: r.now()}
	}
}

// Resume re-registers tombstones recovered from the journal (engine.Voided) so their reservations are
// released on the next pass, with no grace period: the ids are already void. Release is idempotent, so
// a reservation whose release landed before a crash is a no-op. Call it after journal.Recover.
func (r *Reconciler) Resume(voided []engine.Order) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.suspects == nil {
		r.suspects = map[string]suspect{}
	}
	for _, o := range voided {
		if _, ok := r.suspects[o.OrderID]; !ok {
			r.suspects[o.OrderID] = suspect{tenantID: o.TenantID, isPaper: o.IsPaper} // zero since: due now
		}
	}
}

// Pending returns how many suspects are waiting out their grace period or being retried.
func (r *Reconciler) Pending() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.suspects)
}

// RunOnce handles every suspect whose grace period has passed, in order_id order. It never holds its
// own lock while calling the engine, because the engine calls Suspect with the engine lock held.
func (r *Reconciler) RunOnce(ctx context.Context) Report {
	grace := r.Grace
	if grace <= 0 {
		grace = 2 * time.Minute
	}
	now := r.now()
	var rep Report
	due := map[string]suspect{}
	r.mu.Lock()
	for id, s := range r.suspects {
		if now.Sub(s.since) < grace {
			rep.Waiting++
			continue
		}
		due[id] = s
	}
	r.mu.Unlock()

	ids := make([]string, 0, len(due))
	for id := range due {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if !r.resolve(ctx, id, due[id], now, &rep) {
			continue
		}
		r.mu.Lock()
		if cur, ok := r.suspects[id]; ok && cur == due[id] {
			delete(r.suspects, id)
		}
		r.mu.Unlock()
	}
	return rep
}

// resolve handles one suspect and reports whether it is finished.
func (r *Reconciler) resolve(ctx context.Context, id string, s suspect, now time.Time, rep *Report) bool {
	o, voided, err := r.Engine.Void(engine.VoidCmd{OrderID: id, TenantID: s.tenantID, IsPaper: s.isPaper, TS: now})
	switch {
	case errors.Is(err, engine.ErrOrderIDTaken):
		// Another tenant's order now owns this id, and its reservation (if any) may be backing it. It
		// is never released from here; the worker closes that order's reservation like any other.
		slog.Error("orphan suspect's order_id belongs to another tenant; leaving its reservation alone", "order_id", id, "tenant_id", s.tenantID)
		rep.Kept++
		return true
	case err != nil:
		// The journal still refuses writes, or the durable journal is ahead of this engine.
		slog.Warn("cannot void orphan suspect yet; will retry", "order_id", id, "err", err)
		rep.Failed++
		return false
	case o.State != engine.Rejected:
		// Live or closed: a retry was accepted and reused the reservation. The worker owns it now.
		rep.Kept++
		return true
	}
	// Voided (now or on an earlier pass) or rejected at entry: nothing can claim the reservation.
	if voided {
		rep.Voided++
		slog.Warn("voided an order_id whose submit never reached the journal", "order_id", id, "tenant_id", s.tenantID, "is_paper", s.isPaper)
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := r.Ledger.Release(cctx, id); err != nil {
		slog.Warn("releasing an orphaned reservation failed; will retry", "order_id", id, "err", err)
		rep.Failed++
		return false
	}
	rep.Released++
	slog.Warn("released an orphaned reservation", "order_id", id, "tenant_id", s.tenantID, "is_paper", s.isPaper)
	return true
}

// Run reconciles every interval until ctx ends, logging each pass that did or tried anything.
func (r *Reconciler) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if rep := r.RunOnce(ctx); rep.Voided+rep.Released+rep.Kept+rep.Failed > 0 {
			slog.Info("reservation reconciler pass", "voided", rep.Voided, "released", rep.Released,
				"kept", rep.Kept, "waiting", rep.Waiting, "failed", rep.Failed)
		}
	}
}
