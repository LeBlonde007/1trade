package model

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestVLLMBackendChat verifies the backend forwards to the runtime's /v1/chat/completions and maps
// the OpenAI response back, including using the runtime's exact token usage for billing.
func TestVLLMBackendChat(t *testing.T) {
	var gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			w.WriteHeader(404)
			return
		}
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotModel = body.Model
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message":       map[string]string{"role": "assistant", "content": "hello from llama"},
				"finish_reason": "stop",
			}},
			"usage": map[string]int{"prompt_tokens": 11, "completion_tokens": 7},
		})
	}))
	defer srv.Close()

	b := NewVLLMBackend(srv.URL, "", nil, 5*time.Second)
	res, err := b.Chat(context.Background(), ChatRequest{
		Model: "llama-3.1-8b", Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if gotModel != "llama-3.1-8b" {
		t.Fatalf("runtime got model %q", gotModel)
	}
	if res.Content != "hello from llama" || res.PromptTokens != 11 || res.CompletionTokens != 7 || res.FinishReason != "stop" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

// TestVLLMBackendFallbackTokens checks that when the runtime omits usage, the backend estimates
// non-zero token counts (so billing is never zero).
func TestVLLMBackendFallbackTokens(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "some generated words here"}}},
		})
	}))
	defer srv.Close()

	res, err := NewVLLMBackend(srv.URL, "", nil, 5*time.Second).Chat(context.Background(), ChatRequest{
		Model: "llama-3.1-8b", Messages: []Message{{Role: "user", Content: "a longer prompt to estimate"}},
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if res.PromptTokens <= 0 || res.CompletionTokens <= 0 {
		t.Fatalf("expected estimated non-zero tokens, got %+v", res)
	}
}

// TestVLLMBackendError surfaces a runtime non-200 as an error (the gateway 500s).
func TestVLLMBackendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	if _, err := NewVLLMBackend(srv.URL, "", nil, 2*time.Second).Chat(context.Background(), ChatRequest{
		Model: "llama-3.1-8b", Messages: []Message{{Role: "user", Content: "hi"}},
	}); err == nil {
		t.Fatal("expected error on runtime 503")
	}
}

// TestVLLMBackendHostedProvider verifies that with an API key + model map (the hosted-provider path,
// e.g. OpenRouter), the backend sends a bearer token and translates the catalog id to the provider
// slug — while the customer still asked for our catalog id.
func TestVLLMBackendHostedProvider(t *testing.T) {
	var gotAuth, gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotModel = body.Model
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": "hi"}, "finish_reason": "stop"}},
			"usage":   map[string]int{"prompt_tokens": 3, "completion_tokens": 2},
		})
	}))
	defer srv.Close()

	b := NewVLLMBackend(srv.URL, "sk-test-key", map[string]string{"llama-3.1-8b": "meta-llama/llama-3.1-8b-instruct"}, 5*time.Second)
	if _, err := b.Chat(context.Background(), ChatRequest{
		Model: "llama-3.1-8b", Messages: []Message{{Role: "user", Content: "hi"}},
	}); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if gotAuth != "Bearer sk-test-key" {
		t.Fatalf("expected bearer auth header, got %q", gotAuth)
	}
	if gotModel != "meta-llama/llama-3.1-8b-instruct" {
		t.Fatalf("expected provider slug, got %q", gotModel)
	}
}
