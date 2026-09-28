package supply

import (
	"context"
	"log/slog"
	"time"

	"github.com/trade1/compute-control/internal/events"
	"github.com/trade1/compute-control/internal/pool"
)

// StaleAfter is how long a source may go without a heartbeat before it stops taking new work.
const StaleAfter = 2 * time.Minute

// Schedulable reports whether a source should take new work at now: active, a fresh heartbeat, and
// at least one healthy GPU.
func Schedulable(src Source, now time.Time) bool {
	return src.State == Active && src.LastHeartbeatAt != nil && now.Sub(*src.LastHeartbeatAt) <= StaleAfter &&
		src.GPUsHealthy != nil && *src.GPUsHealthy > 0
}

// capacityOf is what a source offers: its healthy GPUs from the latest heartbeat, never more than
// it registered.
func capacityOf(src Source) int {
	if src.GPUsHealthy == nil {
		return src.GPUCount
	}
	return min(*src.GPUsHealthy, src.GPUCount)
}

// Syncer mirrors the registry into the scheduler's pool. A source that is not schedulable stays in
// the pool as not accepting, so its running work keeps its GPUs (the drain); a retired source is
// removed once it has drained.
type Syncer struct {
	Store *Store
	Pool  *pool.Pool
	Now   func() time.Time // nil = time.Now
}

// Sync applies the registry to the pool once.
func (y *Syncer) Sync(ctx context.Context) error {
	now := time.Now()
	if y.Now != nil {
		now = y.Now()
	}
	srcs, err := y.Store.List(ctx, "")
	if err != nil {
		return err
	}
	for _, src := range srcs {
		if src.State == Retired {
			y.Pool.SetSource(src.ID, map[string]int{src.GPUType: capacityOf(src)}, false)
			y.Pool.RemoveSource(src.ID) // no-op until its running work has drained
			continue
		}
		y.Pool.SetSource(src.ID, map[string]int{src.GPUType: capacityOf(src)}, Schedulable(src, now))
	}
	return nil
}

// Run syncs every interval until ctx ends (heartbeat staleness is time-driven, so it must tick).
func (y *Syncer) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		if err := y.Sync(ctx); err != nil {
			slog.Warn("supply sync failed; the pool keeps its last view", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Recorder is an events.Publisher that records each metered interval per supply source, then passes
// it on. The record is what payouts (F18) reconcile against, so it is written before publishing.
type Recorder struct {
	Next  events.Publisher
	Store *Store
}

// PublishUsage records u, then publishes it.
func (r Recorder) PublishUsage(u events.ComputeUsage) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := r.Store.RecordUsage(ctx, u.UsageID, u.SupplySourceID, u.TenantID, u.CreditType, u.GPUSeconds, u.Units, u.IsPaper); err != nil {
		slog.Error("supply usage not recorded", "usage_id", u.UsageID, "source", u.SupplySourceID, "err", err)
	}
	cancel()
	r.Next.PublishUsage(u)
}
