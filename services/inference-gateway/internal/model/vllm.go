package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// VLLMBackend serves inference by calling a runtime pod's OpenAI-compatible API (vLLM's native
// shape) over HTTP. It implements model.Backend, so swapping it in for MockBackend gives the gateway
// real model output with no change to the customer API. See services/inference-runtime/README.md.
type VLLMBackend struct {
	baseURL string
	http    *http.Client
}

// NewVLLMBackend builds a backend pointed at a runtime base URL (e.g. http://inference-runtime:8000).
func NewVLLMBackend(baseURL string, timeout time.Duration) *VLLMBackend {
	return &VLLMBackend{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
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
	reqBody := map[string]any{"model": req.Model, "messages": msgs, "stream": false}
	if req.MaxTokens > 0 {
		reqBody["max_tokens"] = req.MaxTokens
	}
	body, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return ChatResult{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := b.http.Do(httpReq)
	if err != nil {
		return ChatResult{}, fmt.Errorf("runtime call: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ChatResult{}, fmt.Errorf("runtime returned %d", resp.StatusCode)
	}
	var out vllmChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ChatResult{}, fmt.Errorf("decode runtime response: %w", err)
	}
	if len(out.Choices) == 0 {
		return ChatResult{}, fmt.Errorf("runtime returned no choices")
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
