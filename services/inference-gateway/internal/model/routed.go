package model

import (
	"context"
	"errors"
	"io"
)

// ErrCapabilityUnavailable is returned when the routed backend can't serve a requested modality.
var ErrCapabilityUnavailable = errors.New("the selected backend does not support this modality")

// RoutedBackend dispatches each request to a per-model backend: model ids in OpenAIModels go to the
// OpenAI backend (frontier image / vision / text), everything else to Default (the DigitalOcean
// multimodal runtime). Video always uses Default — its poll/content steps key off an opaque job id (no
// model is available there) and OpenAI has no video API. RoutedBackend satisfies Backend plus the
// Image/Speech/Vision/Video capability interfaces, so the API handlers route transparently through their
// existing type assertions — no handler changes. When OpenAI isn't configured the gateway uses the
// Default backend directly (this type isn't created), so the no-key path is unchanged.
type RoutedBackend struct {
	Default      Backend
	OpenAI       Backend
	OpenAIModels map[string]bool
}

// pick selects the backend for a catalog model id.
func (r *RoutedBackend) pick(model string) Backend {
	if r.OpenAI != nil && r.OpenAIModels[model] {
		return r.OpenAI
	}
	return r.Default
}

// Chat routes a chat completion.
func (r *RoutedBackend) Chat(ctx context.Context, req ChatRequest) (ChatResult, error) {
	return r.pick(req.Model).Chat(ctx, req)
}

// Image routes text-to-image generation.
func (r *RoutedBackend) Image(ctx context.Context, req ImageRequest) (ImageResult, error) {
	b, ok := r.pick(req.Model).(ImageBackend)
	if !ok {
		return ImageResult{}, ErrCapabilityUnavailable
	}
	return b.Image(ctx, req)
}

// Speech routes text-to-speech.
func (r *RoutedBackend) Speech(ctx context.Context, req SpeechRequest) (io.ReadCloser, string, int, error) {
	b, ok := r.pick(req.Model).(SpeechBackend)
	if !ok {
		return nil, "", 0, ErrCapabilityUnavailable
	}
	return b.Speech(ctx, req)
}

// Vision routes image understanding.
func (r *RoutedBackend) Vision(ctx context.Context, req VisionRequest) (VisionResult, error) {
	b, ok := r.pick(req.Model).(VisionBackend)
	if !ok {
		return VisionResult{}, ErrCapabilityUnavailable
	}
	return b.Vision(ctx, req)
}

// SubmitVideo starts an async video job — always on the default backend (OpenAI has no video API).
func (r *RoutedBackend) SubmitVideo(ctx context.Context, req VideoRequest) (VideoJob, error) {
	b, ok := r.Default.(VideoBackend)
	if !ok {
		return VideoJob{}, ErrCapabilityUnavailable
	}
	return b.SubmitVideo(ctx, req)
}

// GetVideo polls an async video job — always on the default backend.
func (r *RoutedBackend) GetVideo(ctx context.Context, id string) (VideoJob, error) {
	b, ok := r.Default.(VideoBackend)
	if !ok {
		return VideoJob{}, ErrCapabilityUnavailable
	}
	return b.GetVideo(ctx, id)
}

// GetVideoContent streams a finished video — always on the default backend.
func (r *RoutedBackend) GetVideoContent(ctx context.Context, id string) (io.ReadCloser, string, int, error) {
	b, ok := r.Default.(VideoBackend)
	if !ok {
		return nil, "", 0, ErrCapabilityUnavailable
	}
	return b.GetVideoContent(ctx, id)
}
