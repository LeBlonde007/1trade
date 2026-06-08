// Package config loads inference-gateway configuration from environment variables only.
package config

import (
	"encoding/json"
	"os"
	"time"
)

// Config holds runtime configuration for the inference gateway.
type Config struct {
	Env               string            // dev | staging | prod
	Addr              string            // listen address, e.g. ":8085"
	PlatformCoreURL   string            // base URL for API-key introspection (F08 auth, task #21)
	CreditLedgerURL   string            // base URL for the pre-flight balance check + (events drive debit)
	NATSURL           string            // JetStream URL for emitting inference.usage.v1
	JWTSecret         string            // shared HS256 secret to verify first-party tenant JWTs
	ServiceToken      string            // service-to-service token for internal calls (ledger/platform-core)
	InferenceBackend  string            // "mock" (default) | "vllm" — which model.Backend to serve with
	VLLMBaseURL       string            // runtime base URL when InferenceBackend=vllm (local stub OR a hosted OpenAI-compatible provider, e.g. OpenRouter https://openrouter.ai/api/v1)
	InferenceAPIKey   string            // bearer key for a hosted provider (OpenRouter/Groq/…); empty for the keyless local stub. NEVER hardcoded — from a Secret.
	InferenceModelMap map[string]string // optional catalog-id → provider-id map (e.g. {"llama-3.1-8b":"meta-llama/llama-3.1-8b-instruct"})
	InferenceTimeout  time.Duration     // model-call timeout (generation can take longer than control calls)
	HTTPTimeout       time.Duration     // upstream HTTP client timeout
}

// Load reads configuration from the environment with sensible dev defaults.
func Load() Config {
	return Config{
		Env:               envOr("EXASCALE_ENV", "dev"),
		Addr:              envOr("INFERENCE_ADDR", ":8085"),
		PlatformCoreURL:   envOr("PLATFORM_CORE_URL", "http://platform-core:8001"),
		CreditLedgerURL:   envOr("CREDIT_LEDGER_URL", "http://credit-ledger:8002"),
		NATSURL:           envOr("NATS_URL", "nats://nats.data.svc.cluster.local:4222"),
		JWTSecret:         os.Getenv("PLATFORM_JWT_SECRET"),
		ServiceToken:      os.Getenv("SERVICE_TOKEN"),
		InferenceBackend:  envOr("INFERENCE_BACKEND", "mock"),
		VLLMBaseURL:       envOr("VLLM_BASE_URL", "http://inference-runtime:8000"),
		InferenceAPIKey:   os.Getenv("INFERENCE_API_KEY"),
		InferenceModelMap: mediaModelMap(os.Getenv("INFERENCE_MODEL_MAP")),
		InferenceTimeout:  120 * time.Second,
		HTTPTimeout:       5 * time.Second,
	}
}

// IsDev reports whether the dev environment is active.
func (c Config) IsDev() bool { return c.Env == "dev" }

// envOr returns the env var or a fallback when unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// jsonMap parses a JSON object string into a map (catalog-id → provider-id). It returns nil on
// empty/invalid input rather than failing: a bad model map degrades to pass-through ids, never a
// crash at startup. Used only for the optional INFERENCE_MODEL_MAP convenience translation.
func jsonMap(s string) map[string]string {
	if s == "" {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}

// defaultMediaMap maps the catalog's media model ids to the upstream provider's slugs so image / video
// / speech generation works out of the box against DigitalOcean's multimodal inference. Flux and
// ElevenLabs aren't carried by DO, so they're substituted with the nearest available model — set
// INFERENCE_MODEL_MAP (which overrides these) or another provider for exact parity.
var defaultMediaMap = map[string]string{
	"gpt-image-1.5":        "stable-diffusion-3.5-large", // openai-gpt-image is tier-gated on DO base
	"stable-diffusion-3.5": "stable-diffusion-3.5-large",
	"flux-schnell":         "stable-diffusion-3.5-large", // DO has no Flux
	"wan-t2v":              "wan2-2-t2v-a14b",
	"qwen3-tts":            "qwen3-tts-voicedesign",
	"elevenlabs-tts":       "qwen3-tts-voicedesign",
}

// mediaModelMap merges the built-in media defaults with the operator's INFERENCE_MODEL_MAP, with the
// env map taking precedence so any mapping can be overridden per deployment.
func mediaModelMap(env string) map[string]string {
	merged := make(map[string]string, len(defaultMediaMap)+8)
	for k, v := range defaultMediaMap {
		merged[k] = v
	}
	for k, v := range jsonMap(env) {
		merged[k] = v
	}
	return merged
}
