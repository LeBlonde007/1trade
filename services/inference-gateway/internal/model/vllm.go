package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// VLLMBackend serves inference by calling any OpenAI-compatible chat API over HTTP — either our own
// vLLM runtime pod (keyless, in-cluster) or a hosted provider such as OpenRouter/Groq (bearer key).
// It implements model.Backend, so swapping it in for MockBackend gives the gateway real model output
// with no change to the customer API. See services/inference-runtime/README.md.
type VLLMBackend struct {
	baseURL  string
	apiKey   string            // bearer key for a hosted provider; empty for the keyless in-cluster runtime
	modelMap map[string]string // optional catalog-id → provider-id translation (e.g. OpenRouter slugs)
	http     *http.Client
}

// NewVLLMBackend builds a backend pointed at an OpenAI-compatible base URL (e.g.
// http://inference-runtime:8000 for the local runtime, or https://openrouter.ai/api/v1 for a hosted
// provider). apiKey is sent as a bearer token when non-empty; modelMap (may be nil) translates our
// catalog ids to the provider's model slugs.
func NewVLLMBackend(baseURL, apiKey string, modelMap map[string]string, timeout time.Duration) *VLLMBackend {
	return &VLLMBackend{
		baseURL:  strings.TrimRight(baseURL, "/"),
		apiKey:   apiKey,
		modelMap: modelMap,
		http:     &http.Client{Timeout: timeout},
	}
}

// providerModel maps a catalog model id to the upstream provider's slug via modelMap, falling back to
// the id unchanged when there's no mapping (so the in-cluster runtime, which serves by raw id, works
// untouched).
func (b *VLLMBackend) providerModel(id string) string {
	if m, ok := b.modelMap[id]; ok {
		return m
	}
	return id
}

// vllmChatResponse is the slice of the runtime's OpenAI chat.completion response the gateway needs.
type vllmChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Chat forwards the request to the runtime's /v1/chat/completions and maps the response back. Token
// counts come from the runtime's usage (the real tokenizer); if absent they fall back to an estimate
// so billing is never zero. A non-200 from the runtime is surfaced as an error (the gateway 500s).
func (b *VLLMBackend) Chat(ctx context.Context, req ChatRequest) (ChatResult, error) {
	msgs := make([]map[string]string, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = map[string]string{"role": m.Role, "content": m.Content}
	}
	reqBody := map[string]any{"model": b.providerModel(req.Model), "messages": msgs, "stream": false}
	if req.MaxTokens > 0 {
		reqBody["max_tokens"] = req.MaxTokens
	}
	body, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ChatResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if b.apiKey != "" { // hosted provider (OpenRouter/Groq/…); the in-cluster runtime is keyless
		httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)
	}

	resp, err := b.http.Do(httpReq)
	if err != nil {
		return ChatResult{}, fmt.Errorf("runtime call: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Surface the provider's reason (truncated) — a hosted provider 402 (out of credits), 404
		// (unknown model slug), or 401 (bad key) otherwise hides behind a blind 500. This lands in the
		// gateway's error log and helps the operator fix the model map / top up the provider.
		excerpt, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return ChatResult{}, fmt.Errorf("provider returned %d for model %q: %s", resp.StatusCode, b.providerModel(req.Model), bytes.TrimSpace(excerpt))
	}
	var out vllmChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ChatResult{}, fmt.Errorf("decode provider response: %w", err)
	}
	if len(out.Choices) == 0 {
		return ChatResult{}, fmt.Errorf("provider returned no choices for model %q (check the model slug / availability)", b.providerModel(req.Model))
	}

	content := out.Choices[0].Message.Content
	pt, ct := out.Usage.PromptTokens, out.Usage.CompletionTokens
	if pt == 0 { // runtime omitted usage — estimate so billing isn't zero
		var b strings.Builder
		for _, m := range req.Messages {
			b.WriteString(m.Content)
			b.WriteString("\n")
		}
		pt = CountTokens(b.String())
	}
	if ct == 0 {
		ct = CountTokens(content)
	}
	finish := out.Choices[0].FinishReason
	if finish == "" {
		finish = "stop"
	}
	return ChatResult{Content: content, PromptTokens: pt, CompletionTokens: ct, FinishReason: finish}, nil
}

// imageResponse is the slice of the provider's OpenAI-compatible image-generation response we need.
type imageResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
	} `json:"data"`
}

// Image forwards a text-to-image request to the provider's /v1/images/generations (OpenAI-compatible,
// e.g. DigitalOcean's multimodal inference) and returns the generated images as base64 PNGs. Requires a
// hosted provider (apiKey set); a non-200 surfaces the provider's reason, like Chat. It satisfies
// model.ImageBackend, so the gateway only offers image generation when this backend is in use.
func (b *VLLMBackend) Image(ctx context.Context, req ImageRequest) (ImageResult, error) {
	n := req.N
	if n < 1 {
		n = 1
	}
	size := req.Size
	if size == "" {
		size = "1024x1024"
	}
	reqBody := map[string]any{
		"model": b.providerModel(req.Model), "prompt": req.Prompt,
		"n": n, "size": size, "response_format": "b64_json",
	}
	body, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/v1/images/generations", bytes.NewReader(body))
	if err != nil {
		return ImageResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if b.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+b.apiKey)
	}

	resp, err := b.http.Do(httpReq)
	if err != nil {
		return ImageResult{}, fmt.Errorf("image provider call: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		excerpt, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return ImageResult{}, fmt.Errorf("image provider returned %d for model %q: %s", resp.StatusCode, b.providerModel(req.Model), bytes.TrimSpace(excerpt))
	}
	var out imageResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ImageResult{}, fmt.Errorf("decode image response: %w", err)
	}
	imgs := make([]string, 0, len(out.Data))
	for _, d := range out.Data {
		if d.B64JSON != "" {
			imgs = append(imgs, d.B64JSON)
		}
	}
	if len(imgs) == 0 {
		return ImageResult{}, fmt.Errorf("image provider returned no images for model %q (check the model slug / availability)", b.providerModel(req.Model))
	}
	return ImageResult{B64: imgs}, nil
}
