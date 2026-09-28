// Package routing is F11 multi-model-per-GPU packing: where models live, and which replica serves a
// request.
//
// On this stack one runtime process serves one model (inference-runtime README), so "several models
// per GPU" means several single-model processes sharing a GPU's memory. This package decides
// placement and routing; starting/stopping processes is the compute control plane's job (F12), behind
// the Loader interface.
//
// Two rules the spec makes non-negotiable:
//   - VRAM math is real: every decision uses a model's MEASURED peak VRAM. A model with no measurement
//     is refused ("should fit" is not an input), and every GPU keeps a headroom fraction free.
//   - Hot-swap must not thrash: static (popular) models are never evicted; a replica serving requests
//     is never evicted; a replica younger than MinResidency is never evicted.
package routing

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Errors.
var (
	ErrUnmeasured   = errors.New("routing: model has no measured peak VRAM")
	ErrUnknownModel = errors.New("routing: model has no profile")
	ErrNoCapacity   = errors.New("routing: no GPU can host the model without evicting protected replicas")
	ErrDoesNotFit   = errors.New("routing: model is larger than any GPU's usable VRAM")
	ErrLoading      = errors.New("routing: model is already being loaded")
	ErrWarm         = errors.New("routing: model already has a live replica")
)

// GPU is one GPU (or MIG slice) that runtime processes can share.
type GPU struct {
	ID      string
	Class   string // e.g. "h100-80gb"
	VRAMMiB int64
}

// Profile is what is known about a model's footprint. PeakVRAMMiB must come from a measurement (the
// runtime's peak allocation under load, incl. KV cache at the configured max batch) — never a nominal
// "weights size".
type Profile struct {
	Model       string
	PeakVRAMMiB int64
	MeasuredAt  time.Time // zero = not measured → refused
	Static      bool      // popular: pinned by the planner, never evicted
}

// Replica is one running model process on a GPU.
type Replica struct {
	Model    string
	GPU      string
	URL      string
	Static   bool
	LoadedAt time.Time
	LastUsed time.Time
	Inflight int
	// Loading: the process is starting; its VRAM is claimed but it takes no requests yet.
	Loading bool
	// Draining: chosen for eviction; it takes no new requests and is about to be stopped.
	Draining bool
}

// routable reports whether a replica may take a request.
func (rep *Replica) routable() bool { return !rep.Loading && !rep.Draining }

// Config tunes packing and eviction.
type Config struct {
	HeadroomPPM  int64         // VRAM kept free on every GPU, parts per million (default 50_000 = 5%)
	MinResidency time.Duration // a replica cannot be evicted before this age (default 2 min)
}

// withDefaults fills unset fields.
func (c Config) withDefaults() Config {
	if c.HeadroomPPM <= 0 {
		c.HeadroomPPM = 50_000
	}
	if c.MinResidency <= 0 {
		c.MinResidency = 2 * time.Minute
	}
	return c
}

// usable is a GPU's budget after headroom.
func (c Config) usable(g GPU) int64 { return g.VRAMMiB - g.VRAMMiB*c.HeadroomPPM/1_000_000 }

// Placement is the planner's answer: GPU id → models pinned there.
type Placement map[string][]string

// Plan pins the static models onto GPUs, most popular first (the order of static), each on the GPU
// where it fits most tightly (best-fit decreasing keeps big contiguous room for the tail). Every model
// must be measured; every static model must be placed, or Plan fails — a popular model that is not
// resident would silently become a cold start.
func Plan(cfg Config, gpus []GPU, static []Profile) (Placement, error) {
	cfg = cfg.withDefaults()
	free := map[string]int64{}
	for _, g := range gpus {
		free[g.ID] = cfg.usable(g)
	}
	order := append([]Profile(nil), static...)
	for _, p := range order {
		if p.MeasuredAt.IsZero() || p.PeakVRAMMiB <= 0 {
			return nil, fmt.Errorf("%w: %s", ErrUnmeasured, p.Model)
		}
	}
	// Largest first; popularity order breaks ties (stable sort keeps the caller's order).
	sort.SliceStable(order, func(i, j int) bool { return order[i].PeakVRAMMiB > order[j].PeakVRAMMiB })
	out := Placement{}
	for _, p := range order {
		best := ""
		for _, g := range gpus {
			if free[g.ID] >= p.PeakVRAMMiB && (best == "" || free[g.ID] < free[best]) {
				best = g.ID
			}
		}
		if best == "" {
			return nil, fmt.Errorf("%w: static model %s (%d MiB)", ErrNoCapacity, p.Model, p.PeakVRAMMiB)
		}
		free[best] -= p.PeakVRAMMiB
		out[best] = append(out[best], p.Model)
	}
	return out, nil
}

// Decision is the router's answer for one request.
type Decision struct {
	// Replica is set when a warm replica can serve now.
	Replica *Replica
	// Otherwise the model must be loaded on LoadGPU after evicting Evict (tail replicas, LRU first).
	LoadGPU string
	Evict   []Replica
	// Loading is set when a load of this model is already underway: wait, do not plan another.
	Loading bool
}

// Router tracks the live replicas and routes requests. Safe for concurrent use.
type Router struct {
	mu       sync.Mutex
	cfg      Config
	gpus     map[string]GPU
	profiles map[string]Profile
	replicas []*Replica
	now      func() time.Time
}

// NewRouter builds a router over a fleet. now is injectable for tests (nil = time.Now).
func NewRouter(cfg Config, gpus []GPU, profiles []Profile, now func() time.Time) *Router {
	r := &Router{cfg: cfg.withDefaults(), gpus: map[string]GPU{}, profiles: map[string]Profile{}, now: now}
	if r.now == nil {
		r.now = time.Now
	}
	for _, g := range gpus {
		r.gpus[g.ID] = g
	}
	for _, p := range profiles {
		r.profiles[p.Model] = p
	}
	return r
}

// used is the measured VRAM committed on a GPU.
func (r *Router) used(gpu string) int64 {
	var n int64
	for _, rep := range r.replicas {
		if rep.GPU == gpu {
			n += r.profiles[rep.Model].PeakVRAMMiB
		}
	}
	return n
}

// Loaded registers a replica that is up and ready. (model, gpu) identifies a replica, so a second
// copy of a model on the same GPU is refused — it would only split the same memory bandwidth.
func (r *Router) Loaded(model, gpu, url string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.profiles[model]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownModel, model)
	}
	g, ok := r.gpus[gpu]
	if !ok {
		return fmt.Errorf("routing: unknown GPU %s", gpu)
	}
	for _, rep := range r.replicas {
		if rep.Model == model && rep.GPU == gpu {
			return fmt.Errorf("routing: %s already loaded on %s", model, gpu)
		}
	}
	if r.used(gpu)+p.PeakVRAMMiB > r.cfg.usable(g) {
		return fmt.Errorf("%w: %s on %s would exceed usable VRAM", ErrNoCapacity, model, gpu)
	}
	t := r.now()
	r.replicas = append(r.replicas, &Replica{Model: model, GPU: gpu, URL: url, Static: p.Static, LoadedAt: t, LastUsed: t})
	return nil
}

// Unloaded removes a live or draining replica (evicted, or its process died). A load in progress is
// not touched: that is AbortLoad.
func (r *Router) Unloaded(model, gpu string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, rep := range r.replicas {
		if rep.Model == model && rep.GPU == gpu && !rep.Loading {
			r.replicas = append(r.replicas[:i], r.replicas[i+1:]...)
			return
		}
	}
}

// Route decides how to serve a request for model. A warm replica is chosen by fewest in-flight
// requests (then least recently used, to spread load) and its in-flight count is incremented — call
// Done when the request ends. A cold model gets a load plan instead.
func (r *Router) Route(model string) (Decision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.profiles[model]
	if !ok {
		return Decision{}, fmt.Errorf("%w: %s", ErrUnknownModel, model)
	}
	var best *Replica
	loading := false
	for _, rep := range r.replicas {
		if rep.Model != model {
			continue
		}
		if rep.Loading {
			loading = true
		}
		if !rep.routable() {
			continue
		}
		if best == nil || rep.Inflight < best.Inflight || (rep.Inflight == best.Inflight && rep.LastUsed.Before(best.LastUsed)) {
			best = rep
		}
	}
	if best != nil {
		best.Inflight++
		best.LastUsed = r.now()
		cp := *best
		return Decision{Replica: &cp}, nil
	}
	if loading {
		return Decision{Loading: true}, nil
	}
	if p.MeasuredAt.IsZero() || p.PeakVRAMMiB <= 0 {
		return Decision{}, fmt.Errorf("%w: %s", ErrUnmeasured, model)
	}
	return r.planLoad(p)
}

// planLoad finds where a cold model can go: a GPU with enough free VRAM (tightest fit), otherwise the
// GPU needing the fewest evictions of evictable replicas (LRU order), ties broken by oldest LastUsed.
func (r *Router) planLoad(p Profile) (Decision, error) {
	fits := false
	ids := make([]string, 0, len(r.gpus))
	for id := range r.gpus {
		ids = append(ids, id)
	}
	sort.Strings(ids) // deterministic
	bestFree, bestFreeGPU := int64(-1), ""
	for _, id := range ids {
		usable := r.cfg.usable(r.gpus[id])
		if usable >= p.PeakVRAMMiB {
			fits = true
		}
		free := usable - r.used(id)
		if free >= p.PeakVRAMMiB && (bestFreeGPU == "" || free < bestFree) {
			bestFree, bestFreeGPU = free, id
		}
	}
	if !fits {
		return Decision{}, fmt.Errorf("%w: %s needs %d MiB", ErrDoesNotFit, p.Model, p.PeakVRAMMiB)
	}
	if bestFreeGPU != "" {
		return Decision{LoadGPU: bestFreeGPU}, nil
	}
	now := r.now()
	type option struct {
		gpu    string
		evict  []Replica
		oldest time.Time
	}
	var best *option
	for _, id := range ids {
		var cands []*Replica
		for _, rep := range r.replicas {
			if rep.GPU == id && rep.routable() && !rep.Static && rep.Inflight == 0 && now.Sub(rep.LoadedAt) >= r.cfg.MinResidency {
				cands = append(cands, rep)
			}
		}
		sort.Slice(cands, func(i, j int) bool { return cands[i].LastUsed.Before(cands[j].LastUsed) })
		free := r.cfg.usable(r.gpus[id]) - r.used(id)
		var ev []Replica
		for _, c := range cands {
			if free >= p.PeakVRAMMiB {
				break
			}
			free += r.profiles[c.Model].PeakVRAMMiB
			ev = append(ev, *c)
		}
		if free < p.PeakVRAMMiB || len(ev) == 0 {
			continue
		}
		o := option{gpu: id, evict: ev, oldest: ev[0].LastUsed}
		if best == nil || len(o.evict) < len(best.evict) || (len(o.evict) == len(best.evict) && o.oldest.Before(best.oldest)) {
			best = &o
		}
	}
	if best == nil {
		return Decision{}, fmt.Errorf("%w: %s", ErrNoCapacity, p.Model)
	}
	return Decision{LoadGPU: best.gpu, Evict: best.evict}, nil
}

// Load is a planned load: start Model on GPU once Evict have been stopped.
type Load struct {
	Model string
	GPU   string
	Evict []Replica
}

// BeginLoad plans a load of a cold model and commits to it: the new model's VRAM is claimed on the
// chosen GPU (a Loading replica that takes no requests), and the eviction victims are set Draining so
// no new request lands on them. Only one load per model runs at a time (ErrLoading). The caller stops
// the victims (Unloaded, or Undrain if that fails), starts the process, then calls FinishLoad — or
// AbortLoad if it could not.
func (r *Router) BeginLoad(model string) (Load, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.profiles[model]
	if !ok {
		return Load{}, fmt.Errorf("%w: %s", ErrUnknownModel, model)
	}
	for _, rep := range r.replicas {
		if rep.Model != model {
			continue
		}
		if rep.Loading {
			return Load{}, ErrLoading
		}
		if rep.routable() {
			return Load{}, ErrWarm
		}
	}
	if p.MeasuredAt.IsZero() || p.PeakVRAMMiB <= 0 {
		return Load{}, fmt.Errorf("%w: %s", ErrUnmeasured, model)
	}
	d, err := r.planLoad(p)
	if err != nil {
		return Load{}, err
	}
	for _, v := range d.Evict {
		for _, rep := range r.replicas {
			if rep.Model == v.Model && rep.GPU == v.GPU {
				rep.Draining = true
			}
		}
	}
	t := r.now()
	r.replicas = append(r.replicas, &Replica{Model: model, GPU: d.LoadGPU, Static: p.Static, LoadedAt: t, LastUsed: t, Loading: true})
	return Load{Model: model, GPU: d.LoadGPU, Evict: d.Evict}, nil
}

// FinishLoad makes a loading replica live at url.
func (r *Router) FinishLoad(model, gpu, url string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rep := range r.replicas {
		if rep.Model == model && rep.GPU == gpu && rep.Loading {
			t := r.now()
			rep.Loading, rep.URL, rep.LoadedAt, rep.LastUsed = false, url, t, t
			return
		}
	}
}

// AbortLoad drops a load that failed, releasing the VRAM it claimed.
func (r *Router) AbortLoad(model, gpu string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, rep := range r.replicas {
		if rep.Model == model && rep.GPU == gpu && rep.Loading {
			r.replicas = append(r.replicas[:i], r.replicas[i+1:]...)
			return
		}
	}
}

// Undrain puts a replica chosen for eviction back into service (its process could not be stopped).
func (r *Router) Undrain(model, gpu string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rep := range r.replicas {
		if rep.Model == model && rep.GPU == gpu {
			rep.Draining = false
		}
	}
}

// Done ends a request on a replica (decrements its in-flight count).
func (r *Router) Done(model, gpu string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rep := range r.replicas {
		if rep.Model == model && rep.GPU == gpu && rep.Inflight > 0 {
			rep.Inflight--
			return
		}
	}
}

// Replicas returns a snapshot of the live replicas.
func (r *Router) Replicas() []Replica {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Replica, 0, len(r.replicas))
	for _, rep := range r.replicas {
		out = append(out, *rep)
	}
	return out
}

// Headroom returns a GPU's free VRAM after committed replicas, and its size — for metrics.
func (r *Router) Headroom(gpu string) (freeMiB, totalMiB int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	g := r.gpus[gpu]
	return g.VRAMMiB - r.used(gpu), g.VRAMMiB
}
