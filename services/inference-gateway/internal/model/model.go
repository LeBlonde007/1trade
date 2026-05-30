// Package model is the inference backend abstraction. The gateway talks to a Backend; the mock
// backend here runs without a GPU for local dev and tests, and real vLLM (F09) implements the same
// interface so it swaps in behind the gateway with no API change.
package model

import (
	"context"
	"fmt"
	"strings"
)

// Message is one chat message (OpenAI-shaped).
type Message struct {
	Role    string
	Content string
}

// ChatRequest is a normalized chat-completion request handed to a backend.
type ChatRequest struct {
	Model     string
	Messages  []Message
	MaxTokens int
}

// ChatResult is a backend's completion plus the token accounting that drives billing.
type ChatResult struct {
	Content          string
	PromptTokens     int
	CompletionTokens int
	FinishReason     string
}

// Backend serves inference. Implementations: MockBackend (here) and the vLLM client (F09).
type Backend interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResult, error)
}

// CountTokens is a rough, deterministic token estimate (~4 characters per token). It stands in for
// a real tokenizer so billing and tests are stable; the real backend reports exact counts.
func CountTokens(s string) int {
	n := len([]rune(s)) / 4
	if n < 1 {
		n = 1
	}
	return n
}

// MockBackend is a deterministic, GPU-free Backend for local dev and tests.
type MockBackend struct{}

// Chat returns a deterministic completion that echoes the last user turn, with real token counts so
// the metered `units` are meaningful. MaxTokens (when set) caps the completion's token estimate.
func (MockBackend) Chat(_ context.Context, req ChatRequest) (ChatResult, error) {
	var prompt strings.Builder
	last := ""
	for _, m := range req.Messages {
		prompt.WriteString(m.Role)
		prompt.WriteString(": ")
		prompt.WriteString(m.Content)
		prompt.WriteString("\n")
		if m.Role == "user" {
			last = m.Content
		}
	}
	out := fmt.Sprintf("This is a mock completion from %s. You said: %q", req.Model, truncate(last, 200))
	ct := CountTokens(out)
	if req.MaxTokens > 0 && ct > req.MaxTokens {
		ct = req.MaxTokens
	}
	return ChatResult{
		Content:          out,
		PromptTokens:     CountTokens(prompt.String()),
		CompletionTokens: ct,
		FinishReason:     "stop",
	}, nil
}

// truncate shortens s to at most n runes (keeps the mock echo bounded).
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
