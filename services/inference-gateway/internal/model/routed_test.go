package model

import (
	"context"
	"testing"
)

// stubChat is a minimal Backend that reports which backend handled the call.
type stubChat struct{ tag string }

func (s stubChat) Chat(_ context.Context, _ ChatRequest) (ChatResult, error) {
	return ChatResult{Content: s.tag}, nil
}

// TestRoutedBackend_routesByModel checks model ids in OpenAIModels hit the OpenAI backend while the rest
// fall through to Default, and that an unconfigured OpenAI backend always uses Default.
func TestRoutedBackend_routesByModel(t *testing.T) {
	r := &RoutedBackend{
		Default:      stubChat{tag: "default"},
		OpenAI:       stubChat{tag: "openai"},
		OpenAIModels: map[string]bool{"gpt-image-1.5": true, "nemotron-vision": true},
	}
	cases := map[string]string{
		"gpt-image-1.5":   "openai",  // mapped → OpenAI
		"nemotron-vision": "openai",  // mapped → OpenAI
		"deepseek-v3.2":   "default", // not mapped → DO
		"stable-diffusion-3.5": "default",
	}
	for model, want := range cases {
		got, err := r.Chat(context.Background(), ChatRequest{Model: model})
		if err != nil || got.Content != want {
			t.Fatalf("model %q routed to %q (err %v), want %q", model, got.Content, err, want)
		}
	}

	// No OpenAI backend configured → every model uses Default, even mapped ones.
	r2 := &RoutedBackend{Default: stubChat{tag: "default"}, OpenAI: nil, OpenAIModels: map[string]bool{"gpt-image-1.5": true}}
	if got, _ := r2.Chat(context.Background(), ChatRequest{Model: "gpt-image-1.5"}); got.Content != "default" {
		t.Fatalf("with nil OpenAI, routed to %q, want default", got.Content)
	}
}
