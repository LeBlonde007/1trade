package model

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
)

// EmbedRequest asks for one vector per input string.
type EmbedRequest struct {
	Model string
	Input []string
}

// EmbedResult is one vector per input, in order, plus the tokens billed.
type EmbedResult struct {
	Vectors      [][]float64
	PromptTokens int
}

// EmbeddingsBackend creates embeddings (POST /v1/embeddings).
type EmbeddingsBackend interface {
	Embed(ctx context.Context, req EmbedRequest) (EmbedResult, error)
}

// TranscribeRequest is one audio file to transcribe.
type TranscribeRequest struct {
	Model    string
	Filename string
	Audio    []byte
	Language string // optional ISO-639-1 hint
}

// TranscribeResult is the transcript and the audio's duration, which is what speech-to-text bills.
type TranscribeResult struct {
	Text        string
	DurationSec float64
}

// TranscriptionBackend transcribes audio (POST /v1/audio/transcriptions).
type TranscriptionBackend interface {
	Transcribe(ctx context.Context, req TranscribeRequest) (TranscribeResult, error)
}

// mockDims is the mock embedding width.
const mockDims = 16

// Embed returns deterministic unit vectors derived from each input's hash, so identical inputs embed
// identically — enough for dev and tests; the vLLM backend returns real ones.
func (MockBackend) Embed(_ context.Context, req EmbedRequest) (EmbedResult, error) {
	res := EmbedResult{Vectors: make([][]float64, len(req.Input))}
	for i, in := range req.Input {
		sum := sha256.Sum256([]byte(in))
		v := make([]float64, mockDims)
		var norm float64
		for d := range v {
			x := float64(binary.BigEndian.Uint16(sum[2*d:]))/math.MaxUint16*2 - 1 // in [-1, 1]
			v[d], norm = x, norm+x*x
		}
		for d := range v {
			v[d] /= math.Sqrt(norm)
		}
		res.Vectors[i] = v
		res.PromptTokens += CountTokens(in)
	}
	return res, nil
}

// mockBytesPerSecond treats mock audio as 16 kHz, 16-bit mono PCM when estimating its duration.
const mockBytesPerSecond = 32000

// Transcribe returns a placeholder transcript and a duration estimated from the file size.
func (MockBackend) Transcribe(_ context.Context, req TranscribeRequest) (TranscribeResult, error) {
	return TranscribeResult{
		Text:        fmt.Sprintf("This is a mock transcript of %s (%d bytes).", req.Filename, len(req.Audio)),
		DurationSec: float64(len(req.Audio)) / mockBytesPerSecond,
	}, nil
}

// Embed calls an OpenAI-compatible POST /v1/embeddings.
func (b *VLLMBackend) Embed(ctx context.Context, req EmbedRequest) (EmbedResult, error) {
	body, _ := json.Marshal(map[string]any{"model": b.providerModel(req.Model), "input": req.Input})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return EmbedResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if b.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)
	}
	resp, err := b.http.Do(httpReq)
	if err != nil {
		return EmbedResult{}, fmt.Errorf("embeddings provider call: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		excerpt, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return EmbedResult{}, fmt.Errorf("embeddings provider returned %d for model %q: %s", resp.StatusCode, b.providerModel(req.Model), bytes.TrimSpace(excerpt))
	}
	var out struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
		} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return EmbedResult{}, fmt.Errorf("decode embeddings response: %w", err)
	}
	if len(out.Data) != len(req.Input) {
		return EmbedResult{}, fmt.Errorf("embeddings provider returned %d vectors for %d inputs", len(out.Data), len(req.Input))
	}
	res := EmbedResult{Vectors: make([][]float64, len(req.Input)), PromptTokens: out.Usage.PromptTokens}
	for _, d := range out.Data {
		if d.Index < 0 || d.Index >= len(req.Input) || res.Vectors[d.Index] != nil {
			return EmbedResult{}, fmt.Errorf("embeddings provider returned a bad index %d", d.Index)
		}
		res.Vectors[d.Index] = d.Embedding
	}
	if res.PromptTokens == 0 { // some providers omit usage; bill our own estimate rather than nothing
		for _, in := range req.Input {
			res.PromptTokens += CountTokens(in)
		}
	}
	return res, nil
}

// Transcribe calls an OpenAI-compatible POST /v1/audio/transcriptions, asking for verbose_json so
// the audio's duration (what the request bills) comes back with the text.
func (b *VLLMBackend) Transcribe(ctx context.Context, req TranscribeRequest) (TranscribeResult, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("model", b.providerModel(req.Model))
	_ = mw.WriteField("response_format", "verbose_json")
	if req.Language != "" {
		_ = mw.WriteField("language", req.Language)
	}
	fw, err := mw.CreateFormFile("file", req.Filename)
	if err != nil {
		return TranscribeResult{}, err
	}
	if _, err := fw.Write(req.Audio); err != nil {
		return TranscribeResult{}, err
	}
	if err := mw.Close(); err != nil {
		return TranscribeResult{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/v1/audio/transcriptions", &buf)
	if err != nil {
		return TranscribeResult{}, err
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	if b.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)
	}
	resp, err := b.http.Do(httpReq)
	if err != nil {
		return TranscribeResult{}, fmt.Errorf("transcription provider call: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		excerpt, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return TranscribeResult{}, fmt.Errorf("transcription provider returned %d for model %q: %s", resp.StatusCode, b.providerModel(req.Model), bytes.TrimSpace(excerpt))
	}
	var out struct {
		Text     string  `json:"text"`
		Duration float64 `json:"duration"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return TranscribeResult{}, fmt.Errorf("decode transcription response: %w", err)
	}
	if out.Duration <= 0 {
		return TranscribeResult{}, fmt.Errorf("transcription provider reported no duration; refusing to bill a guess")
	}
	return TranscribeResult{Text: out.Text, DurationSec: out.Duration}, nil
}

// Embed routes by model, like Chat.
func (r *RoutedBackend) Embed(ctx context.Context, req EmbedRequest) (EmbedResult, error) {
	b, ok := r.pick(req.Model).(EmbeddingsBackend)
	if !ok {
		return EmbedResult{}, ErrCapabilityUnavailable
	}
	return b.Embed(ctx, req)
}

// Transcribe routes by model, like Chat.
func (r *RoutedBackend) Transcribe(ctx context.Context, req TranscribeRequest) (TranscribeResult, error) {
	b, ok := r.pick(req.Model).(TranscriptionBackend)
	if !ok {
		return TranscribeResult{}, ErrCapabilityUnavailable
	}
	return b.Transcribe(ctx, req)
}
