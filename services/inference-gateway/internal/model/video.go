package model

import (
	"context"
	"io"
)

// VideoRequest is a normalized text-to-video request handed to a video-capable backend.
type VideoRequest struct {
	Model  string
	Prompt string
	Size   string // e.g. "1280x720" (defaults downstream)
}

// VideoJob is the async job handle returned by submit/poll. Status moves queued → in_progress →
// completed (or failed). Raw carries the provider's full object for the client to inspect.
type VideoJob struct {
	ID     string
	Status string
	Raw    map[string]any
}

// VideoBackend is the optional async text-to-video capability (DigitalOcean's /v1/videos). A backend
// that can serve it implements this; the gateway type-asserts for it so backends without video support
// (the mock) don't have to. The flow is: SubmitVideo → poll GetVideo until status=="completed" →
// GetVideoContent streams the mp4.
type VideoBackend interface {
	SubmitVideo(ctx context.Context, req VideoRequest) (VideoJob, error)
	GetVideo(ctx context.Context, id string) (VideoJob, error)
	// GetVideoContent returns the mp4 body stream + its content-type + the upstream status. The caller
	// copies it through and closes the reader.
	GetVideoContent(ctx context.Context, id string) (body io.ReadCloser, contentType string, status int, err error)
}
