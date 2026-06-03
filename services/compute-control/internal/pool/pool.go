// Package pool is the shared, in-memory GPU capacity that both the internal job scheduler (F12) and
// the customer instance manager (F13) draw from — so a GPU held by a running instance is unavailable
// to a job, and vice versa. Counts are per credit-type tier (gpu_h100 / gpu_h200). In M3 this is the
// mock-GPU pool; the real backend swaps it for live capacity reported by the GPU Operator. Safe for
// concurrent use.
package pool

import (
	"maps"
	"sync"
)

// Pool tracks per-tier GPU capacity and how many of each are currently reserved.
type Pool struct {
	mu       sync.Mutex
	capacity map[string]int
	reserved map[string]int
}

// New builds a pool with the given per-tier capacity (e.g. {"gpu_h100": 8, "gpu_h200": 0}).
func New(perTier map[string]int) *Pool {
	c := make(map[string]int, len(perTier))
	maps.Copy(c, perTier)
	return &Pool{capacity: c, reserved: make(map[string]int, len(perTier))}
}

// Reserve atomically takes n GPUs of a tier if enough are free; it returns false (reserving nothing)
// when the request would exceed capacity. n<=0 is a no-op that succeeds.
func (p *Pool) Reserve(tier string, n int) bool {
	if n <= 0 {
		return true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.reserved[tier]+n > p.capacity[tier] {
		return false
	}
	p.reserved[tier] += n
	return true
}

// Release returns n GPUs of a tier to the pool, clamped so reserved never goes below zero.
func (p *Pool) Release(tier string, n int) {
	if n <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reserved[tier] -= n
	if p.reserved[tier] < 0 {
		p.reserved[tier] = 0
	}
}

// Available is the number of free GPUs of a tier (capacity − reserved).
func (p *Pool) Available(tier string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.capacity[tier] - p.reserved[tier]
}

// Capacity is the total GPUs of a tier.
func (p *Pool) Capacity(tier string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.capacity[tier]
}

// InUse is the number of reserved GPUs of a tier.
func (p *Pool) InUse(tier string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.reserved[tier]
}
