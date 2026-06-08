package client

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestStreamChatTokens verifies the SSE parser: it concatenates the per-chunk content deltas in order,
// ignores keep-alives, stops at [DONE], and returns the usage carried on the final chunk.
func TestStreamChatTokens(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, ": keep-alive\n\n")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"Hello"}}]}`+"\n\n")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":", world"}}]}`+"\n\n")
		fmt.Fprint(w, `data: {"choices":[{"delta":{"content":"!"}}],"usage":{"prompt_tokens":3,"completion_tokens":4,"total_tokens":7}}`+"\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	var got strings.Builder
	usage, err := StreamChat(srv.URL, "/v1/chat/completions", "tok", map[string]any{"stream": true}, func(s string) {
		got.WriteString(s)
	})
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}
	if got.String() != "Hello, world!" {
		t.Fatalf("streamed content = %q, want %q", got.String(), "Hello, world!")
	}
	if usage.TotalTokens != 7 || usage.PromptTokens != 3 || usage.CompletionTokens != 4 {
		t.Fatalf("usage = %+v, want 3/4/7", usage)
	}
}

// TestStreamChatError surfaces a non-2xx stream as an *APIError with the upstream code/message intact
// (so the CLI can attach a hint).
func TestStreamChatError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
		fmt.Fprint(w, `{"code":"insufficient_credits","message":"not enough text credits"}`)
	}))
	defer srv.Close()

	_, err := StreamChat(srv.URL, "/v1/chat/completions", "tok", map[string]any{"stream": true}, func(string) {})
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *APIError, got %v", err)
	}
	if ae.Status != 402 || ae.Code != "insufficient_credits" {
		t.Fatalf("APIError = %+v, want 402/insufficient_credits", ae)
	}
}
