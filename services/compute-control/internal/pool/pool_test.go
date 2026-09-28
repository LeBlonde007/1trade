package pool

import (
	"fmt"
	"sync"
	"testing"
)

const h100 = "gpu_h100"

// TestPlacementIsBlindToOwnership: the source with the most free GPUs wins, owned or partner alike;
// ties go to the lower id; a gang is never split across datacenters.
func TestPlacementIsBlindToOwnership(t *testing.T) {
	p := New("dc-owned-1", map[string]int{h100: 4})
	p.SetSource("partner-a", map[string]int{h100: 6}, true)
	if src, ok := p.Reserve(h100, 2); !ok || src != "partner-a" {
		t.Fatalf("first placement on %q, want partner-a (most free)", src)
	}
	// Now 4 free on each: the tie goes to the lower id.
	if src, _ := p.Reserve(h100, 1); src != "dc-owned-1" {
		t.Fatalf("tie went to %q", src)
	}
	// 3 + 4 free, but no single datacenter has 5.
	if src, ok := p.Reserve(h100, 5); ok {
		t.Fatalf("a gang of 5 was split or overbooked onto %q", src)
	}
	if p.Available(h100) != 7 || p.MaxPlaceable(h100) != 4 {
		t.Fatalf("available %d, max placeable %d", p.Available(h100), p.MaxPlaceable(h100))
	}
}

// TestSuspendedSourceDrains: a source that stops accepting takes no new work, keeps what it runs,
// and can be removed once that work is released.
func TestSuspendedSourceDrains(t *testing.T) {
	p := New("dc-owned-1", map[string]int{h100: 1})
	p.SetSource("partner-a", map[string]int{h100: 8}, true)
	src, _ := p.Reserve(h100, 3)
	if src != "partner-a" {
		t.Fatal(src)
	}
	p.SetSource("partner-a", map[string]int{h100: 8}, false) // suspended
	if s, ok := p.Reserve(h100, 2); ok {
		t.Fatalf("a suspended source took new work (%s)", s)
	}
	if p.InUse(h100) != 3 || p.Capacity(h100) != 1 {
		t.Fatalf("in use %d (running work must keep its GPUs), capacity %d", p.InUse(h100), p.Capacity(h100))
	}
	if p.RemoveSource("partner-a") {
		t.Fatal("removed a source still running work")
	}
	p.Release("partner-a", h100, 3)
	if !p.RemoveSource("partner-a") || len(p.Sources()) != 1 {
		t.Fatalf("drained source not removed: %+v", p.Sources())
	}
	p.Release("partner-a", h100, 1) // late release onto a removed source is harmless
}

// TestShrinkBelowReserved: capacity can drop under what is running; nothing is evicted and no new
// work lands until the source is back under capacity.
func TestShrinkBelowReserved(t *testing.T) {
	p := New("dc", map[string]int{h100: 4})
	p.Reserve(h100, 4)
	p.SetSource("dc", map[string]int{h100: 2}, true)
	if _, ok := p.Reserve(h100, 1); ok || p.Available(h100) != 0 {
		t.Fatal("an over-committed source took more work")
	}
	p.Release("dc", h100, 3)
	if _, ok := p.Reserve(h100, 1); !ok {
		t.Fatal("back under capacity but still refusing")
	}
}

// TestReserveOn pins work to one source, respecting its state and capacity.
func TestReserveOn(t *testing.T) {
	p := New("dc", map[string]int{h100: 2})
	p.SetSource("partner-a", map[string]int{h100: 2}, false)
	if p.ReserveOn("partner-a", h100, 1) || p.ReserveOn("nope", h100, 1) || p.ReserveOn("dc", h100, 3) {
		t.Fatal("reserved on a suspended, unknown or too-small source")
	}
	if !p.ReserveOn("dc", h100, 2) || p.Available(h100) != 0 {
		t.Fatal("pinned reservation failed")
	}
}

// TestConcurrentNeverOverbooks: racing reservers across several sources never exceed any source's
// capacity, and every GPU comes back.
func TestConcurrentNeverOverbooks(t *testing.T) {
	p := New("dc-0", map[string]int{h100: 10})
	for i := 1; i < 4; i++ {
		p.SetSource(fmt.Sprint("dc-", i), map[string]int{h100: 10}, true)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	held := map[string]int{}
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				n := 1 + (g+i)%4
				src, ok := p.Reserve(h100, n)
				if !ok {
					continue
				}
				mu.Lock()
				held[src] += n
				if held[src] > 10 {
					t.Errorf("%s overbooked: %d", src, held[src])
				}
				held[src] -= n
				mu.Unlock()
				p.Release(src, h100, n)
			}
		}()
	}
	wg.Wait()
	if p.InUse(h100) != 0 || p.Available(h100) != 40 {
		t.Fatalf("leaked: in use %d, available %d", p.InUse(h100), p.Available(h100))
	}
}
