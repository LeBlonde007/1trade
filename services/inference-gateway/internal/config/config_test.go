package config_test

import (
	"testing"

	"github.com/trade1/inference-gateway/internal/config"
)

// TestMediaDefaultsOnlyOnDigitalOcean — the built-in media map is DigitalOcean slugs. Merging it
// into every deployment made image/speech/video models look routable on backends that cannot serve
// them: a self-hosted text-only runtime advertised wan-t2v and stable-diffusion, then failed the
// call at the upstream. They belong only where they resolve.
func TestMediaDefaultsOnlyOnDigitalOcean(t *testing.T) {
	t.Setenv("INFERENCE_MODEL_MAP", `{"llama-3.2-1b":"llama3.2:1b"}`)

	t.Setenv("VLLM_BASE_URL", "http://host.k3d.internal:11434") // self-hosted Ollama
	local := config.Load()
	if _, ok := local.InferenceModelMap["wan-t2v"]; ok {
		t.Error("self-hosted backend must NOT inherit DigitalOcean media defaults")
	}
	if got := local.InferenceModelMap["llama-3.2-1b"]; got != "llama3.2:1b" {
		t.Errorf("operator mapping lost: got %q", got)
	}
	if len(local.InferenceModelMap) != 1 {
		t.Errorf("self-hosted map = %d entries, want only the operator's 1", len(local.InferenceModelMap))
	}

	t.Setenv("VLLM_BASE_URL", "https://inference.do-ai.run") // DigitalOcean multimodal
	do := config.Load()
	if _, ok := do.InferenceModelMap["wan-t2v"]; !ok {
		t.Error("DigitalOcean backend should still get the media defaults")
	}
	if got := do.InferenceModelMap["llama-3.2-1b"]; got != "llama3.2:1b" {
		t.Errorf("operator map must still win over defaults: got %q", got)
	}
}
