// Package config loads inference-gateway configuration from environment variables only.
package config

import (
	"os"
	"time"
)

// Config holds runtime configuration for the inference gateway.
type Config struct {
	Env             string        // dev | staging | prod
	Addr            string        // listen address, e.g. ":8085"
	PlatformCoreURL string        // base URL for API-key introspection (F08 auth, task #21)
	CreditLedgerURL string        // base URL for the pre-flight balance check + (events drive debit)
	NATSURL         string        // JetStream URL for emitting inference.usage.v1
	JWTSecret       string        // shared HS256 secret to verify first-party tenant JWTs
	ServiceToken    string        // service-to-service token for internal calls (ledger/platform-core)
	HTTPTimeout     time.Duration // upstream HTTP client timeout
}

// Load reads configuration from the environment with sensible dev defaults.
func Load() Config {
	return Config{
		Env:             envOr("EXASCALE_ENV", "dev"),
		Addr:            envOr("INFERENCE_ADDR", ":8085"),
		PlatformCoreURL: envOr("PLATFORM_CORE_URL", "http://platform-core:8001"),
		CreditLedgerURL: envOr("CREDIT_LEDGER_URL", "http://credit-ledger:8002"),
		NATSURL:         envOr("NATS_URL", "nats://nats.data.svc.cluster.local:4222"),
		JWTSecret:       os.Getenv("PLATFORM_JWT_SECRET"),
		ServiceToken:    os.Getenv("SERVICE_TOKEN"),
		HTTPTimeout:     5 * time.Second,
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
