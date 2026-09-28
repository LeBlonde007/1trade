package pool

import "testing"

// TestHoldsSetCapacityAside: a hold is sold only against free capacity, on-demand work cannot take
// held GPUs, and the owner's work draws from its hold first, then on-demand.
func TestHoldsSetCapacityAside(t *testing.T) {
	p := New("own", map[string]int{"gpu_h100": 16})
	a := HoldKey{Tenant: "a", IsPaper: true, Tier: "gpu_h100"}
	if p.Hold(a, 17) || !p.Hold(a, 10) {
		t.Fatal("hold must be sold only against free capacity")
	}
	if p.Available("gpu_h100") != 6 || p.MaxPlaceable("gpu_h100") != 6 {
		t.Fatalf("available %d / placeable %d, want 6", p.Available("gpu_h100"), p.MaxPlaceable("gpu_h100"))
	}
	if _, ok := p.Reserve("gpu_h100", 7); ok {
		t.Fatal("on-demand took held GPUs")
	}
	if p.Hold(HoldKey{Tenant: "b", Tier: "gpu_h100"}, 7) {
		t.Fatal("a second reservation oversold the pool")
	}
	// The owner: 8 from the hold, then 2 more from the hold, then on-demand.
	if _, fromHold, ok := p.ReserveFor(a, 8); !ok || !fromHold {
		t.Fatal("owner work should come from its hold")
	}
	if _, fromHold, ok := p.ReserveFor(a, 4); !ok || fromHold {
		t.Fatal("4 > the 2 left on the hold: should be on-demand")
	}
	if p.Available("gpu_h100") != 2 {
		t.Fatalf("available %d, want 16 - 8 - 4 - 2 unused held", p.Available("gpu_h100"))
	}
	// The paper hold does not serve the tenant's real work.
	if _, fromHold, _ := p.ReserveFor(HoldKey{Tenant: "a", Tier: "gpu_h100"}, 2); fromHold {
		t.Fatal("a paper reservation served real work")
	}
	p.ReleaseFor(a, "own", 8, true)
	if h := p.Holds()[a]; h != [2]int{10, 0} {
		t.Fatalf("holds %v", h)
	}
}

// TestHoldEndsOverdrawn: when a reservation ends while its work runs, the running work is overdrawn
// and moves to on-demand.
func TestHoldEndsOverdrawn(t *testing.T) {
	p := New("own", map[string]int{"gpu_h100": 8})
	a := HoldKey{Tenant: "a", Tier: "gpu_h100"}
	p.Hold(a, 8)
	if _, fromHold, ok := p.ReserveFor(a, 8); !ok || !fromHold {
		t.Fatal("place")
	}
	p.SetHold(a, 0)
	if p.Overdrawn(a) != 8 {
		t.Fatalf("overdrawn %d", p.Overdrawn(a))
	}
	p.Detach(a, 8)
	if p.Overdrawn(a) != 0 || len(p.Holds()) != 0 {
		t.Fatalf("after detach: %d %v", p.Overdrawn(a), p.Holds())
	}
	p.Release("own", "gpu_h100", 8)
	if p.Available("gpu_h100") != 8 {
		t.Fatal("capacity not back")
	}
}
