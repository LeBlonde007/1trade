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
		InferenceModelMap: jsonMap(os.Getenv("INFERENCE_MODEL_MAP")),
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
