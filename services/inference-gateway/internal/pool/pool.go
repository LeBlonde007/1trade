// Package pool is the gateway side of F11: chat for pooled models is spread across several
// single-model runtime pods that share GPUs, using internal/routing to pick the replica.
//
// The pool routes; it does not start or stop processes. A pooled model with no warm replica returns
// model.ErrModelCold (the handler answers 503 + Retry-After) and the load plan is logged for the compute
// control plane (F12) — the gateway never pretends a cold model is being served.
package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

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
}

// GPUSpec is one GPU in the pool.
type GPUSpec struct {
	ID      string `json:"id"`
	Class   string `json:"class"`
	VRAMMiB int64  `json:"vram_mib"`
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
	Fallback model.Backend
	router   *routing.Router
	pooled   map[string]bool
	clients  map[string]model.Backend // replica URL → client
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
	b := &Backend{Fallback: fallback, router: routing.NewRouter(cfg, gpus, profiles, nil), pooled: pooled, clients: map[string]model.Backend{}}
	running := map[string]bool{}
	for _, r := range s.Replicas {
		if err := b.router.Loaded(r.Model, r.GPU, r.URL); err != nil {
			return nil, err
		}
		b.clients[r.URL] = newClient(r.URL)
		running[r.Model] = true
	}
	for _, p := range static {
		if !running[p.Model] {
			return nil, fmt.Errorf("INFERENCE_POOL: static model %s has no running replica", p.Model)
		}
	}
	return b, nil
}

// Models returns the pooled model ids.
func (b *Backend) Models() []string {
	out := make([]string, 0, len(b.pooled))
	for m := range b.pooled {
		out = append(out, m)
	}
	return out
}

// Router exposes the router (for metrics and, later, the F12 loader).
func (b *Backend) Router() *routing.Router { return b.router }

// Chat serves a pooled model on its least-loaded warm replica; other models go to Fallback.
func (b *Backend) Chat(ctx context.Context, req model.ChatRequest) (model.ChatResult, error) {
	if !b.pooled[req.Model] {
		return b.Fallback.Chat(ctx, req)
	}
	d, err := b.router.Route(req.Model)
	if err != nil {
		return model.ChatResult{}, fmt.Errorf("%w: %w", model.ErrModelCold, err)
	}
	if d.Replica == nil {
		slog.Info("pooled model cold; load needed", "model", req.Model, "gpu", d.LoadGPU, "evict", len(d.Evict))
		return model.ChatResult{}, model.ErrModelCold
	}
	defer b.router.Done(d.Replica.Model, d.Replica.GPU)
	return b.clients[d.Replica.URL].Chat(ctx, req)
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
