// Package pool is the gateway side of F11: chat for pooled models is spread across several
// single-model runtime pods that share GPUs, using internal/routing to pick the replica.
//
// With a Loader configured, the pool also starts and stops the runtime processes: a request for a
// cold pooled model commits a load plan (the router claims the VRAM and drains the eviction victims),
// stops the victims, starts the model and routes to it once it answers ready. One load per model runs
// at a time; requests wait for it up to load_wait and otherwise get model.ErrModelCold (503 +
// Retry-After). Static models missing at boot are loaded before the gateway serves. Without a Loader
// the pool only routes, and a cold model is always ErrModelCold — the gateway never pretends a cold
// model is being served.
package pool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/trade1/inference-gateway/internal/metrics"
	"github.com/trade1/inference-gateway/internal/model"
	"github.com/trade1/inference-gateway/internal/routing"
)

// Spec is the INFERENCE_POOL JSON: the fleet, measured model profiles, and the replicas running now.
type Spec struct {
	HeadroomPPM  int64      `json:"headroom_ppm"`
	MinResidency string     `json:"min_residency"` // Go duration, e.g. "2m"
	GPUs         []GPUSpec  `json:"gpus"`
	Profiles     []ProfSpec `json:"profiles"`
	Replicas     []ReplSpec `json:"replicas"`
	// Loader starts and stops runtime processes (hot-swap). Unset: the pool only routes.
	Loader *LoaderSpec `json:"loader"`
	// LoadWait is how long a request for a cold model waits for its load (Go duration; "0s" = answer
	// 503 at once while the load runs in the background).
	LoadWait string `json:"load_wait"`
}

// LoaderSpec configures the process loader.
type LoaderSpec struct {
	Kind         string   `json:"kind"`          // "process"
	Command      []string `json:"command"`       // argv with {model} {port} {gpu} {device}
	ReadyTimeout string   `json:"ready_timeout"` // Go duration (default 10m)
	Host         string   `json:"host"`
}

// GPUSpec is one GPU in the pool.
type GPUSpec struct {
	ID      string `json:"id"`
	Class   string `json:"class"`
	VRAMMiB int64  `json:"vram_mib"`
	Device  string `json:"device"` // CUDA device index for processes placed on this GPU
}

// ProfSpec is a model's measured footprint. measured_at is required: an unmeasured model is refused.
type ProfSpec struct {
	Model       string    `json:"model"`
	PeakVRAMMiB int64     `json:"peak_vram_mib"`
	MeasuredAt  time.Time `json:"measured_at"`
	Static      bool      `json:"static"`
}

// ReplSpec is a running runtime pod: which model, on which GPU, at which URL.
type ReplSpec struct {
	Model string `json:"model"`
	GPU   string `json:"gpu"`
	URL   string `json:"url"`
}

// Backend serves chat for pooled models via the router and delegates everything else to Fallback.
type Backend struct {
	Fallback  model.Backend
	router    *routing.Router
	pooled    map[string]bool
	loader    Loader
	loadWait  time.Duration
	newClient func(url string) model.Backend

	mu      sync.RWMutex
	clients map[string]model.Backend // replica URL → client
	loads   map[string]chan struct{} // model → closed when its load ends
}

// ParseSpec decodes INFERENCE_POOL, rejecting unknown fields so a typo cannot silently drop a limit.
func ParseSpec(raw string) (Spec, error) {
	var s Spec
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return Spec{}, fmt.Errorf("INFERENCE_POOL: %w", err)
	}
	return s, nil
}

// New validates the spec and builds the pool. It fails when a profile is unmeasured, the static models
// cannot all be placed, a replica would overcommit its GPU, or a static model has no replica — a
// popular model that is not resident would quietly become a cold start. newClient builds the HTTP
// client for one replica URL.
func New(s Spec, fallback model.Backend, newClient func(url string) model.Backend) (*Backend, error) {
	cfg := routing.Config{HeadroomPPM: s.HeadroomPPM}
	if s.MinResidency != "" {
		d, err := time.ParseDuration(s.MinResidency)
		if err != nil {
			return nil, fmt.Errorf("INFERENCE_POOL min_residency: %w", err)
		}
		cfg.MinResidency = d
	}
	gpus := make([]routing.GPU, len(s.GPUs))
	for i, g := range s.GPUs {
		gpus[i] = routing.GPU{ID: g.ID, Class: g.Class, VRAMMiB: g.VRAMMiB}
	}
	profiles := make([]routing.Profile, len(s.Profiles))
	var static []routing.Profile
	pooled := map[string]bool{}
	for i, p := range s.Profiles {
		profiles[i] = routing.Profile{Model: p.Model, PeakVRAMMiB: p.PeakVRAMMiB, MeasuredAt: p.MeasuredAt, Static: p.Static}
		if p.MeasuredAt.IsZero() || p.PeakVRAMMiB <= 0 {
			return nil, fmt.Errorf("%w: %s", routing.ErrUnmeasured, p.Model)
		}
		if p.Static {
			static = append(static, profiles[i])
		}
		pooled[p.Model] = true
	}
	if _, err := routing.Plan(cfg, gpus, static); err != nil {
		return nil, err
	}
	b := &Backend{Fallback: fallback, router: routing.NewRouter(cfg, gpus, profiles, nil), pooled: pooled,
		clients: map[string]model.Backend{}, loads: map[string]chan struct{}{}, newClient: newClient}
	if s.LoadWait != "" {
		d, err := time.ParseDuration(s.LoadWait)
		if err != nil || d < 0 {
			return nil, fmt.Errorf("INFERENCE_POOL load_wait: %q is not a duration", s.LoadWait)
		}
		b.loadWait = d
	}
	if s.Loader != nil {
		l, err := newLoader(*s.Loader, s.GPUs)
		if err != nil {
			return nil, err
		}
		l.OnExit = b.processExited
		b.loader = l
	}
	running := map[string]bool{}
	for _, r := range s.Replicas {
		if err := b.router.Loaded(r.Model, r.GPU, r.URL); err != nil {
			return nil, err
		}
		b.clients[r.URL] = newClient(r.URL)
		running[r.Model] = true
	}
	// Static (popular) models must be resident before the gateway serves: load the missing ones,
	// largest first (the order Plan proved feasible), or refuse to start.
	sort.SliceStable(static, func(i, j int) bool { return static[i].PeakVRAMMiB > static[j].PeakVRAMMiB })
	for _, p := range static {
		if running[p.Model] {
			continue
		}
		if b.loader == nil {
			return nil, fmt.Errorf("INFERENCE_POOL: static model %s has no running replica", p.Model)
		}
		if done := b.ensureLoad(context.Background(), p.Model); done != nil {
			<-done
		}
		if !b.warm(p.Model) {
			return nil, fmt.Errorf("INFERENCE_POOL: static model %s could not be loaded", p.Model)
		}
	}
	return b, nil
}

// newLoader builds the configured loader.
func newLoader(ls LoaderSpec, gpus []GPUSpec) (*ProcessLoader, error) {
	if ls.Kind != "process" || len(ls.Command) == 0 {
		return nil, errors.New(`INFERENCE_POOL loader: kind must be "process" with a command`)
	}
	l := &ProcessLoader{Command: ls.Command, Host: ls.Host, Devices: map[string]string{}}
	if ls.ReadyTimeout != "" {
		d, err := time.ParseDuration(ls.ReadyTimeout)
		if err != nil || d <= 0 {
			return nil, fmt.Errorf("INFERENCE_POOL loader ready_timeout: %q is not a duration", ls.ReadyTimeout)
		}
		l.ReadyTimeout = d
	}
	for _, g := range gpus {
		l.Devices[g.ID] = g.Device
	}
	return l, nil
}

// SetLoader replaces the loader (tests, or a control-plane-backed loader).
func (b *Backend) SetLoader(l Loader) { b.loader = l }

// warm reports whether a model has a live replica.
func (b *Backend) warm(m string) bool {
	for _, r := range b.router.Replicas() {
		if r.Model == m && !r.Loading && !r.Draining {
			return true
		}
	}
	return false
}

// ensureLoad starts a load of a cold model unless one is running, and returns a channel closed when it
// ends. nil means no load could be planned (the model does not fit without evicting protected
// replicas); a closed channel means the model is already warm.
func (b *Backend) ensureLoad(ctx context.Context, m string) <-chan struct{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ch, ok := b.loads[m]; ok {
		return ch
	}
	ld, err := b.router.BeginLoad(m)
	if errors.Is(err, routing.ErrWarm) {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	if err != nil {
		slog.Warn("pooled model cannot be loaded now", "model", m, "err", err)
		return nil
	}
	ch := make(chan struct{})
	b.loads[m] = ch
	go b.runLoad(context.WithoutCancel(ctx), ld, ch) // a load outlives the request that triggered it
	return ch
}

// runLoad carries out a committed load: stop the victims, start the model, route to it. On any failure
// the claimed VRAM is released and victims that are still running go back into service.
func (b *Backend) runLoad(ctx context.Context, ld routing.Load, done chan struct{}) {
	defer func() {
		b.mu.Lock()
		delete(b.loads, ld.Model)
		b.mu.Unlock()
		close(done)
	}()
	for i, v := range ld.Evict {
		if err := b.loader.Unload(ctx, v.Model, v.GPU, v.URL); err != nil {
			slog.Error("evicting a replica failed; load abandoned", "model", ld.Model, "victim", v.Model, "err", err)
			for _, rest := range ld.Evict[i:] {
				b.router.Undrain(rest.Model, rest.GPU)
			}
			b.router.AbortLoad(ld.Model, ld.GPU)
			metrics.ModelLoadsTotal.WithLabelValues(ld.Model, "failed").Inc()
			return
		}
		b.router.Unloaded(v.Model, v.GPU)
		b.dropClient(v.URL)
		metrics.ModelEvictionsTotal.WithLabelValues(v.Model).Inc()
		slog.Info("replica evicted", "model", v.Model, "gpu", v.GPU, "for", ld.Model)
	}
	start := time.Now()
	url, err := b.loader.Load(ctx, ld.Model, ld.GPU)
	if err != nil {
		slog.Error("model load failed", "model", ld.Model, "gpu", ld.GPU, "err", err)
		b.router.AbortLoad(ld.Model, ld.GPU)
		metrics.ModelLoadsTotal.WithLabelValues(ld.Model, "failed").Inc()
		return
	}
	b.mu.Lock()
	b.clients[url] = b.newClient(url)
	b.mu.Unlock()
	b.router.FinishLoad(ld.Model, ld.GPU, url)
	took := time.Since(start)
	metrics.ModelLoadSeconds.WithLabelValues(ld.Model).Observe(took.Seconds())
	metrics.ModelLoadsTotal.WithLabelValues(ld.Model, "ok").Inc()
	slog.Info("model loaded", "model", ld.Model, "gpu", ld.GPU, "url", url, "cold_start_ms", took.Milliseconds())
}

// processExited drops a replica whose process died, so no request is routed to it.
func (b *Backend) processExited(m, gpu string) {
	for _, r := range b.router.Replicas() {
		if r.Model == m && r.GPU == gpu && !r.Loading {
			b.router.Unloaded(m, gpu)
			b.dropClient(r.URL)
		}
	}
}

// dropClient forgets a replica's client.
func (b *Backend) dropClient(url string) {
	b.mu.Lock()
	delete(b.clients, url)
	b.mu.Unlock()
}

// client returns a replica's client.
func (b *Backend) client(url string) model.Backend {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.clients[url]
}

// Models returns the pooled model ids.
func (b *Backend) Models() []string {
	out := make([]string, 0, len(b.pooled))
	for m := range b.pooled {
		out = append(out, m)
	}
	return out
}

// Router exposes the router (for metrics and tests).
func (b *Backend) Router() *routing.Router { return b.router }

// Chat serves a pooled model on its least-loaded warm replica; other models go to Fallback. A cold
// model is loaded (with a Loader) and the request waits up to load_wait for it; otherwise, or if the
// load takes longer, it gets model.ErrModelCold.
func (b *Backend) Chat(ctx context.Context, req model.ChatRequest) (model.ChatResult, error) {
	if !b.pooled[req.Model] {
		return b.Fallback.Chat(ctx, req)
	}
	d, err := b.router.Route(req.Model)
	if err != nil {
		return model.ChatResult{}, fmt.Errorf("%w: %w", model.ErrModelCold, err)
	}
	if d.Replica == nil {
		if b.loader == nil {
			slog.Info("pooled model cold; no loader configured", "model", req.Model, "gpu", d.LoadGPU, "evict", len(d.Evict))
			return model.ChatResult{}, model.ErrModelCold
		}
		done := b.ensureLoad(ctx, req.Model)
		if done == nil || b.loadWait <= 0 {
			return model.ChatResult{}, model.ErrModelCold
		}
		t := time.NewTimer(b.loadWait)
		defer t.Stop()
		select {
		case <-done:
		case <-t.C:
			return model.ChatResult{}, model.ErrModelCold
		case <-ctx.Done():
			return model.ChatResult{}, ctx.Err()
		}
		if d, err = b.router.Route(req.Model); err != nil || d.Replica == nil {
			return model.ChatResult{}, model.ErrModelCold
		}
	}
	defer b.router.Done(d.Replica.Model, d.Replica.GPU)
	c := b.client(d.Replica.URL)
	if c == nil {
		return model.ChatResult{}, model.ErrModelCold // evicted between routing and now
	}
	return c.Chat(ctx, req)
}

// Image delegates to Fallback (pooling covers chat models only).
func (b *Backend) Image(ctx context.Context, req model.ImageRequest) (model.ImageResult, error) {
	f, ok := b.Fallback.(model.ImageBackend)
	if !ok {
		return model.ImageResult{}, model.ErrCapabilityUnavailable
	}
	return f.Image(ctx, req)
}

// Speech delegates to Fallback.
func (b *Backend) Speech(ctx context.Context, req model.SpeechRequest) (io.ReadCloser, string, int, error) {
	f, ok := b.Fallback.(model.SpeechBackend)
	if !ok {
		return nil, "", 0, model.ErrCapabilityUnavailable
	}
	return f.Speech(ctx, req)
}

// Vision delegates to Fallback.
func (b *Backend) Vision(ctx context.Context, req model.VisionRequest) (model.VisionResult, error) {
	f, ok := b.Fallback.(model.VisionBackend)
	if !ok {
		return model.VisionResult{}, model.ErrCapabilityUnavailable
	}
	return f.Vision(ctx, req)
}

// SubmitVideo delegates to Fallback.
func (b *Backend) SubmitVideo(ctx context.Context, req model.VideoRequest) (model.VideoJob, error) {
	f, ok := b.Fallback.(model.VideoBackend)
	if !ok {
		return model.VideoJob{}, model.ErrCapabilityUnavailable
	}
	return f.SubmitVideo(ctx, req)
}

// GetVideo delegates to Fallback.
func (b *Backend) GetVideo(ctx context.Context, id string) (model.VideoJob, error) {
	f, ok := b.Fallback.(model.VideoBackend)
	if !ok {
		return model.VideoJob{}, model.ErrCapabilityUnavailable
	}
	return f.GetVideo(ctx, id)
}

// GetVideoContent delegates to Fallback.
func (b *Backend) GetVideoContent(ctx context.Context, id string) (io.ReadCloser, string, int, error) {
	f, ok := b.Fallback.(model.VideoBackend)
	if !ok {
		return nil, "", 0, model.ErrCapabilityUnavailable
	}
	return f.GetVideoContent(ctx, id)
}

// Embed delegates to Fallback (pooling covers chat models only).
func (b *Backend) Embed(ctx context.Context, req model.EmbedRequest) (model.EmbedResult, error) {
	f, ok := b.Fallback.(model.EmbeddingsBackend)
	if !ok {
		return model.EmbedResult{}, model.ErrCapabilityUnavailable
	}
	return f.Embed(ctx, req)
}

// Transcribe delegates to Fallback.
func (b *Backend) Transcribe(ctx context.Context, req model.TranscribeRequest) (model.TranscribeResult, error) {
	f, ok := b.Fallback.(model.TranscriptionBackend)
	if !ok {
		return model.TranscribeResult{}, model.ErrCapabilityUnavailable
	}
	return f.Transcribe(ctx, req)
}

// Close stops every runtime process the pool started (gateway shutdown).
func (b *Backend) Close() {
	if l, ok := b.loader.(*ProcessLoader); ok {
		l.StopAll()
	}
}
