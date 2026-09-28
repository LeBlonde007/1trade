// Package pool is the shared GPU capacity that the internal job scheduler (F12) and the customer
// instance manager (F13) draw from, so a GPU held by an instance is unavailable to a job and vice
// versa.
//
// F16: capacity is held per SUPPLY SOURCE — 1Trade's own datacenter and each partner datacenter —
// and all of it is one pool. Placement never special-cases owned versus partner: a request goes to
// whichever accepting source has the most free GPUs of its tier (ties by source id), and the chosen
// source is returned so every job and instance, and so every usage event, records who supplied it.
// A source that stops accepting (suspended, retired, unhealthy) takes no new work but keeps what it
// is running until it is released: that is the drain.
//
// Counts are per credit-type tier (gpu_h100 / gpu_h200). Safe for concurrent use.
package pool

import (
	"maps"
	"sort"
	"sync"
)

// source is one supplier's capacity and what is reserved on it, per tier.
type source struct {
	capacity  map[string]int
	reserved  map[string]int
	accepting bool
}

// Pool tracks capacity and reservations across supply sources.
type Pool struct {
	mu      sync.Mutex
	sources map[string]*source
}

// SourceStat is a snapshot of one source, for the API and the datacenter dashboard.
type SourceStat struct {
	ID        string
	Capacity  map[string]int
	InUse     map[string]int
	Accepting bool
}

// New builds a pool with one accepting source (normally 1Trade's own datacenter) of the given
// per-tier capacity, e.g. New("dc-owned-1", {"gpu_h100": 8}).
func New(ownedID string, perTier map[string]int) *Pool {
	p := &Pool{sources: map[string]*source{}}
	p.SetSource(ownedID, perTier, true)
	return p
}

// SetSource adds a source or updates its capacity and whether it accepts new work. Shrinking below
// what is reserved is allowed: running work keeps its GPUs, and the source takes no more until it
// is back under capacity.
func (p *Pool) SetSource(id string, perTier map[string]int, accepting bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s, ok := p.sources[id]
	if !ok {
		s = &source{reserved: map[string]int{}}
		p.sources[id] = s
	}
	s.capacity = maps.Clone(perTier)
	if s.capacity == nil {
		s.capacity = map[string]int{}
	}
	s.accepting = accepting
}

// RemoveSource drops a source that holds no reservations. A source still running work is instead
// set to stop accepting, and is dropped by a later call once it has drained. It reports whether the
// source is gone.
func (p *Pool) RemoveSource(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	s, ok := p.sources[id]
	if !ok {
		return true
	}
	for _, n := range s.reserved {
		if n > 0 {
			s.accepting = false
			return false
		}
	}
	delete(p.sources, id)
	return true
}

// ids returns source ids in a fixed order (placement must be deterministic).
func (p *Pool) ids() []string {
	out := make([]string, 0, len(p.sources))
	for id := range p.sources {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// free is a source's free GPUs of a tier (0 when over-committed after a shrink).
func (s *source) free(tier string) int {
	return max(0, s.capacity[tier]-s.reserved[tier])
}

// Reserve atomically takes n GPUs of a tier from ONE accepting source (a gang is never split across
// datacenters) and returns that source. It picks the source with the most free GPUs of the tier, ties
// broken by id. ok=false means no single source can fit n; nothing is reserved. n<=0 is a no-op that
// succeeds with no source.
func (p *Pool) Reserve(tier string, n int) (sourceID string, ok bool) {
	if n <= 0 {
		return "", true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	best, bestFree := "", 0
	for _, id := range p.ids() {
		s := p.sources[id]
		if f := s.free(tier); s.accepting && f >= n && f > bestFree {
			best, bestFree = id, f
		}
	}
	if best == "" {
		return "", false
	}
	p.sources[best].reserved[tier] += n
	return best, true
}

// ReserveOn takes n GPUs of a tier from a specific source (e.g. a reservation pinned to a partner). It
// fails if the source is unknown, not accepting, or short.
func (p *Pool) ReserveOn(sourceID, tier string, n int) bool {
	if n <= 0 {
		return true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	s, ok := p.sources[sourceID]
	if !ok || !s.accepting || s.free(tier) < n {
		return false
	}
	s.reserved[tier] += n
	return true
}

// Release returns n GPUs of a tier to the source that supplied them, clamped at zero. Releasing onto
// a source that has been removed is a no-op.
func (p *Pool) Release(sourceID, tier string, n int) {
	if n <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	s, ok := p.sources[sourceID]
	if !ok {
		return
	}
	s.reserved[tier] = max(0, s.reserved[tier]-n)
}

// Available is the free GPUs of a tier across accepting sources.
func (p *Pool) Available(tier string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := 0
	for _, s := range p.sources {
		if s.accepting {
			n += s.free(tier)
		}
	}
	return n
}

// MaxPlaceable is the largest request of a tier that one source could take right now (a gang must
// fit on a single source, so this can be less than Available).
func (p *Pool) MaxPlaceable(tier string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	best := 0
	for _, s := range p.sources {
		if s.accepting {
			best = max(best, s.free(tier))
		}
	}
	return best
}

// Capacity is the total GPUs of a tier across accepting sources.
func (p *Pool) Capacity(tier string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := 0
	for _, s := range p.sources {
		if s.accepting {
			n += s.capacity[tier]
		}
	}
	return n
}

// InUse is the reserved GPUs of a tier across every source, draining ones included.
func (p *Pool) InUse(tier string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := 0
	for _, s := range p.sources {
		n += s.reserved[tier]
	}
	return n
}

// Sources snapshots every source, in id order.
func (p *Pool) Sources() []SourceStat {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]SourceStat, 0, len(p.sources))
	for _, id := range p.ids() {
		s := p.sources[id]
		out = append(out, SourceStat{ID: id, Capacity: maps.Clone(s.capacity), InUse: maps.Clone(s.reserved), Accepting: s.accepting})
	}
	return out
}
