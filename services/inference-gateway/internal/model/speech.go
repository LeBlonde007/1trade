package model

import (
	"context"
	"io"
)

// SpeechRequest is a normalized text-to-speech request. Voice is an OpenAI-compatible voice id (DO's
// Qwen3 TTS overwrites it with "default"); Instructions is the voice-design description ("a clear,
// warm voice"), which Qwen3-TTS-VoiceDesign requires.
type SpeechRequest struct {
	Model        string
	Input        string
	Voice        string
	Instructions string
	Format       string // e.g. "wav" (default downstream)
}

// SpeechBackend is the optional text-to-speech capability (DO's /v1/audio/speech). A backend that can
// serve it implements this; the gateway type-asserts for it. The provider returns raw audio bytes
// (WAV), so the caller streams the body through and closes it.
type SpeechBackend interface {
	Speech(ctx context.Context, req SpeechRequest) (body io.ReadCloser, contentType string, status int, err error)
}
