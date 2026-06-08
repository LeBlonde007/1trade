package model

import "context"

// VisionRequest is a normalized vision (image-understanding) request: a text prompt plus one image
// (a data URI or https URL) sent to a VLM that returns a text answer.
type VisionRequest struct {
	Model     string
	Prompt    string
	ImageURL  string
	MaxTokens int
}

// VisionResult is the VLM's text answer plus token accounting for billing (in text credits).
type VisionResult struct {
	Text             string
	PromptTokens     int
	CompletionTokens int
}

// VisionBackend is the optional image-understanding capability (a VLM over /v1/chat/completions with an
// image_url content part). A backend that can serve it implements this; the gateway type-asserts for it.
type VisionBackend interface {
	Vision(ctx context.Context, req VisionRequest) (VisionResult, error)
}
