package model

import "context"

// ImageRequest is a normalized text-to-image request handed to an image-capable backend.
type ImageRequest struct {
	Model  string
	Prompt string
	N      int    // number of images (defaults to 1 downstream)
	Size   string // e.g. "1024x1024" (defaults downstream)
}

// ImageResult carries the generated images as base64-encoded PNGs (data is inline; the caller may
// persist them to object storage and hand back URLs).
type ImageResult struct {
	B64 []string
}

// ImageBackend is the optional text-to-image capability. A backend that can serve image generation
// (the hosted multimodal provider) implements it; the gateway type-asserts for it, so backends without
// image support (the GPU-free mock) don't have to — they simply yield a clear "unavailable" error.
type ImageBackend interface {
	Image(ctx context.Context, req ImageRequest) (ImageResult, error)
}
