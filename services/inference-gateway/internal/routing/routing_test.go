package routing

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var t0 = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

// clock is a settable test clock.
type clock struct{ t time.Time }

// now returns the clock's current time.
func (c *clock) now() time.Time { return c.t }

// prof builds a measured profile.
func prof(model string, mib int64, static bool) Profile {
	return Profile{Model: model, PeakVRAMMiB: mib, MeasuredAt: t0, Static: static}
}

// TestPlanPacksBestFitWithinHeadroom checks every GPU stays within usable VRAM and big models go first.
func TestPlanPacksBestFitWithinHeadroom(t *testing.T) {
	gpus := []GPU{{ID: "g0", VRAMMiB: 81920}, {ID: "g1", VRAMMiB: 81920}}
	static := []Profile{prof("small", 7000, true), prof("big", 70000, true), prof("mid", 40000, true), prof("mid2", 30000, true)}
	pl, err := Plan(Config{}, gpus, static)
	if err != nil {
		t.Fatal(err)
	}
	size := map[string]int64{}
	for _, p := range static {
		size[p.Model] = p.PeakVRAMMiB
	}
	usable := Config{}.withDefaults().usable(gpus[0]) // 77824
	placed := 0
	for g, ms := range pl {
		var sum int64
		for _, m := range ms {
			sum += size[m]
			placed++
		}
		if sum > usable {
			t.Fatalf("%s holds %d MiB > usable %d", g, sum, usable)
		}
	}
	if placed != len(static) {
		t.Fatalf("placed %d of %d: %v", placed, len(static), pl)
	}
	// big (70000) leaves 7824 on g0; mid+mid2 leave 7824 on g1; small (7000) takes the tightest gap,
	// which ties — the first GPU wins.
	if fmt.Sprint(pl["g0"]) != "[big small]" || fmt.Sprint(pl["g1"]) != "[mid mid2]" {
		t.Fatalf("unexpected placement %v", pl)
	}
}

// TestPlanHeadroomIsEnforced: a model that fits raw VRAM but not VRAM minus headroom is refused.
func TestPlanHeadroomIsEnforced(t *testing.T) {
	gpus := []GPU{{ID: "g0", VRAMMiB: 100000}}
	if _, err := Plan(Config{}, gpus, []Profile{prof("m", 96000, true)}); !errors.Is(err, ErrNoCapacity) {
		t.Fatalf("want ErrNoCapacity, got %v", err)
	}
	if _, err := Plan(Config{}, gpus, []Profile{prof("m", 95000, true)}); err != nil {
		t.Fatalf("95%% should fit: %v", err)
	}
}

// TestPlanPrefersTightestGPU keeps the big GPU's room free for a big model later.
func TestPlanPrefersTightestGPU(t *testing.T) {
	pl, err := Plan(Config{}, []GPU{{ID: "big", VRAMMiB: 100000}, {ID: "small", VRAMMiB: 50000}}, []Profile{prof("m", 40000, true)})
	if err != nil || len(pl["small"]) != 1 {
		t.Fatalf("m belongs on the small GPU: %v %v", pl, err)
	}
}

// TestPlanRefusesUnmeasured: "should fit" is not an input.
func TestPlanRefusesUnmeasured(t *testing.T) {
	gpus := []GPU{{ID: "g0", VRAMMiB: 81920}}
	for _, p := range []Profile{{Model: "a", PeakVRAMMiB: 1000}, {Model: "b", MeasuredAt: t0}} {
		if _, err := Plan(Config{}, gpus, []Profile{p}); !errors.Is(err, ErrUnmeasured) {
			t.Fatalf("%s: want ErrUnmeasured, got %v", p.Model, err)
		}
	}
}

// TestPlanRandomNeverOvercommits: across random fleets, a successful plan never exceeds usable VRAM.
func TestPlanRandomNeverOvercommits(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	ok := 0
	for i := 0; i < 500; i++ {
		var gpus []GPU
		for g := 0; g < 1+rng.Intn(4); g++ {
			gpus = append(gpus, GPU{ID: fmt.Sprint("g", g), VRAMMiB: int64(20000 + rng.Intn(80000))})
		}
		var static []Profile
		size := map[string]int64{}
		for m := 0; m < 1+rng.Intn(8); m++ {
			p := prof(fmt.Sprint("m", m), int64(1000+rng.Intn(50000)), true)
			static = append(static, p)
			size[p.Model] = p.PeakVRAMMiB
		}
		pl, err := Plan(Config{}, gpus, static)
		if err != nil {
			continue
		}
		ok++
		for _, g := range gpus {
			var sum int64
			for _, m := range pl[g.ID] {
				sum += size[m]
			}
			if sum > (Config{}).withDefaults().usable(g) {
				t.Fatalf("case %d: %s overcommitted", i, g.ID)
			}
		}
	}
	if ok < 100 {
		t.Fatalf("only %d plans succeeded — generator too tight to test anything", ok)
	}
}

// fleet is a two-GPU router with a static model resident.
func fleet(c *clock) *Router {
	gpus := []GPU{{ID: "g0", VRAMMiB: 81920}, {ID: "g1", VRAMMiB: 81920}}
	r := NewRouter(Config{}, gpus, []Profile{
		prof("pop", 60000, true),
		prof("a", 30000, false), prof("b", 30000, false), prof("c", 30000, false),
		prof("huge", 90000, false), prof("tiny", 10000, false),
		{Model: "raw", PeakVRAMMiB: 1000}, // unmeasured
	}, c.now)
	if err := r.Loaded("pop", "g0", "http://pop-g0"); err != nil {
		panic(err)
	}
	return r
}

// TestRouteWarmPrefersFewestInflight spreads load across replicas of the same model.
func TestRouteWarmPrefersFewestInflight(t *testing.T) {
	c := &clock{t0}
	r := NewRouter(Config{}, []GPU{{ID: "g0", VRAMMiB: 81920}, {ID: "g1", VRAMMiB: 81920}}, []Profile{prof("a", 30000, false)}, c.now)
	_ = r.Loaded("a", "g0", "u0")
	_ = r.Loaded("a", "g1", "u1")
	seen := map[string]int{}
	for i := 0; i < 4; i++ {
		c.t = c.t.Add(time.Second)
		d, err := r.Route("a")
		if err != nil || d.Replica == nil {
			t.Fatal(d, err)
		}
		seen[d.Replica.GPU]++
	}
	if seen["g0"] != 2 || seen["g1"] != 2 {
		t.Fatalf("uneven spread: %v", seen)
	}
	r.Done("a", "g1")
	r.Done("a", "g1")
	if d, _ := r.Route("a"); d.Replica.GPU != "g1" {
		t.Fatalf("idle replica should win, got %s", d.Replica.GPU)
	}
}

// TestRouteColdLoadsOnTightestFreeGPU picks the GPU with the least free room that still fits.
func TestRouteColdLoadsOnTightestFreeGPU(t *testing.T) {
	c := &clock{t0}
	r := fleet(c) // g0: 60000 used of 77824 → 17824 free; g1: empty
	d, err := r.Route("a")
	if err != nil || d.Replica != nil || d.LoadGPU != "g1" || len(d.Evict) != 0 {
		t.Fatalf("got %+v %v", d, err)
	}
	if d, _ := r.Route("tiny"); d.LoadGPU != "g0" {
		t.Fatalf("tiny fits beside pop on g0 (tightest); got %+v", d)
	}
	_ = r.Loaded("a", "g1", "ua") // g1: 47824 free
	d, _ = r.Route("b")
	if d.LoadGPU != "g1" || len(d.Evict) != 0 {
		t.Fatalf("b should fit beside a: %+v", d)
	}
}

// TestRouteRefusals: unknown, unmeasured, and larger-than-any-GPU models.
func TestRouteRefusals(t *testing.T) {
	r := fleet(&clock{t0})
	for model, want := range map[string]error{"nope": ErrUnknownModel, "raw": ErrUnmeasured, "huge": ErrDoesNotFit} {
		if _, err := r.Route(model); !errors.Is(err, want) {
			t.Fatalf("%s: want %v, got %v", model, want, err)
		}
	}
}

// TestEvictionProtections: static, busy, and young replicas are never evicted; LRU goes first.
func TestEvictionProtections(t *testing.T) {
	c := &clock{t0}
	r := fleet(c)
	_ = r.Loaded("a", "g1", "ua")
	c.t = c.t.Add(10 * time.Second)
	_ = r.Loaded("b", "g1", "ub") // g1 full-ish: 17824 free
	// c needs 30000: g0 has only static pop; g1 replicas are younger than MinResidency.
	if _, err := r.Route("c"); !errors.Is(err, ErrNoCapacity) {
		t.Fatalf("young replicas must not be evicted: %v", err)
	}
	c.t = t0.Add(3 * time.Minute)
	// a is busy (a long request started at t0+3m) — and is the LRU once b serves a request later.
	if d, _ := r.Route("a"); d.Replica == nil {
		t.Fatal("a should be warm")
	}
	c.t = t0.Add(4 * time.Minute)
	if d, _ := r.Route("b"); d.Replica == nil {
		t.Fatal("b should be warm")
	}
	r.Done("b", "g1")
	d, err := r.Route("c")
	if err != nil || d.LoadGPU != "g1" || len(d.Evict) != 1 || d.Evict[0].Model != "b" {
		t.Fatalf("busy a must survive; b is the victim: %+v %v", d, err)
	}
	r.Done("a", "g1")
	// Now a (used t0+3m) and b (used t0+4m) are both idle and old: LRU is a.
	d, _ = r.Route("c")
	if len(d.Evict) != 1 || d.Evict[0].Model != "a" {
		t.Fatalf("LRU victim should be a: %+v", d)
	}
	for _, e := range d.Evict {
		if e.Static {
			t.Fatal("static replica chosen for eviction")
		}
	}
}

// TestEvictionPrefersFewestVictims picks the GPU where one eviction suffices over one needing two.
func TestEvictionPrefersFewestVictims(t *testing.T) {
	c := &clock{t0}
	gpus := []GPU{{ID: "g0", VRAMMiB: 81920}, {ID: "g1", VRAMMiB: 81920}}
	r := NewRouter(Config{}, gpus, []Profile{
		prof("s1", 25000, false), prof("s2", 25000, false), prof("s3", 25000, false),
		prof("l1", 70000, false), prof("new", 50000, false),
	}, c.now)
	_ = r.Loaded("s1", "g0", "")
	_ = r.Loaded("s2", "g0", "")
	_ = r.Loaded("s3", "g0", "")
	c.t = c.t.Add(time.Second)
	_ = r.Loaded("l1", "g1", "") // l1 used more recently than s*, yet one victim beats two
	c.t = c.t.Add(time.Hour)
	d, err := r.Route("new")
	if err != nil || d.LoadGPU != "g1" || len(d.Evict) != 1 || d.Evict[0].Model != "l1" {
		t.Fatalf("got %+v %v", d, err)
	}
}

// TestNoThrashUnderAlternatingDemand: two cold models competing for one slot cannot swap each other
// out faster than MinResidency.
func TestNoThrashUnderAlternatingDemand(t *testing.T) {
	c := &clock{t0}
	r := NewRouter(Config{MinResidency: time.Minute}, []GPU{{ID: "g0", VRAMMiB: 40000}},
		[]Profile{prof("x", 30000, false), prof("y", 30000, false)}, c.now)
	loads := 0
	for i := 0; i < 600; i++ { // 10 minutes at 1 req/s, alternating
		c.t = t0.Add(time.Duration(i) * time.Second)
		m := []string{"x", "y"}[i%2]
		d, err := r.Route(m)
		if err != nil {
			continue // honest 503 while the resident replica is protected
		}
		if d.Replica != nil {
			r.Done(m, d.Replica.GPU)
			continue
		}
		for _, e := range d.Evict {
			r.Unloaded(e.Model, e.GPU)
		}
		if err := r.Loaded(m, d.LoadGPU, ""); err != nil {
			t.Fatal(err)
		}
		loads++
	}
	if loads > 11 { // at most one swap per MinResidency (+ the first load)
		t.Fatalf("thrash: %d loads in 10 minutes", loads)
	}
}

// TestLoadedRefusesOvercommitAndDuplicates guards the registry itself.
func TestLoadedRefusesOvercommitAndDuplicates(t *testing.T) {
	r := fleet(&clock{t0})
	if err := r.Loaded("a", "g0", ""); !errors.Is(err, ErrNoCapacity) {
		t.Fatalf("g0 has 17824 free; a needs 30000: %v", err)
	}
	if err := r.Loaded("pop", "g0", ""); err == nil {
		t.Fatal("duplicate replica accepted")
	}
	if err := r.Loaded("a", "gX", ""); err == nil {
		t.Fatal("unknown GPU accepted")
	}
	if free, total := r.Headroom("g0"); free != 81920-60000 || total != 81920 {
		t.Fatalf("headroom %d/%d", free, total)
	}
}

// TestConcurrentRouting exercises the router under -race and checks in-flight accounting balances.
func TestConcurrentRouting(t *testing.T) {
	r := NewRouter(Config{}, []GPU{{ID: "g0", VRAMMiB: 81920}, {ID: "g1", VRAMMiB: 81920}}, []Profile{prof("a", 30000, false)}, nil)
	_ = r.Loaded("a", "g0", "")
	_ = r.Loaded("a", "g1", "")
	var wg sync.WaitGroup
	for w := 0; w < 16; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				d, err := r.Route("a")
				if err != nil || d.Replica == nil {
					t.Error(d, err)
					return
				}
				_ = r.Replicas()
				r.Done("a", d.Replica.GPU)
			}
		}()
	}
	wg.Wait()
	for _, rep := range r.Replicas() {
		if rep.Inflight != 0 {
			t.Fatalf("leaked inflight on %s: %d", rep.GPU, rep.Inflight)
		}
	}
}
