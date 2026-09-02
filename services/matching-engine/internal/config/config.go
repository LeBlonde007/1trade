// Package config loads matching-engine configuration from environment variables only.
package config

import (
	"os"
	"time"
)

// Config holds runtime configuration for the matching engine.
//
// Note what is absent: there is no setting that opens order entry. The exchange is paused in code
// (domain.Status), not by configuration, so no environment variable — and no operator mistake — can
// start accepting orders before the licence lands. Switch-on is a reviewed code change (KW03 cutover).
type Config struct {
	Env            string        // dev | staging | prod
	Addr           string        // listen address, e.g. ":8087"
	JWTSecret      string        // shared HS256 secret to verify first-party tenant JWTs
	MethodologyURL string        // where clients link for the published index methodology
	HTTPTimeout    time.Duration // upstream HTTP client timeout
}

// Load reads configuration from the environment with sensible dev defaults.
func Load() Config {
	return Config{
		Env:            envOr("EXASCALE_ENV", "dev"),
		Addr:           envOr("TRADING_ADDR", ":8087"),
		JWTSecret:      os.Getenv("PLATFORM_JWT_SECRET"),
		MethodologyURL: envOr("INDEX_METHODOLOGY_URL", "/methodology"),
		HTTPTimeout:    5 * time.Second,
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
