// Package config loads the credit-ledger configuration from environment variables only
// (ENGINEERING_STANDARDS §4 — no config files in source).
package config

import "os"

// Config holds the service's runtime configuration.
type Config struct {
	Env          string // dev | staging | prod
	Addr         string // listen address, e.g. ":8002"
	DatabaseURL  string // Postgres DSN
	NATSURL      string // NATS URL for credit.tx.v1 (empty → log-only publisher)
	JWTSecret    string // HS256 secret shared with platform-core (verifies tenant JWTs)
	ServiceToken string // bearer token gating internal service-to-service endpoints
}

// Load reads configuration from the environment, applying sensible dev defaults.
func Load() Config {
	return Config{
		Env:          envOr("EXASCALE_ENV", "dev"),
		Addr:         envOr("LEDGER_ADDR", ":8002"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		NATSURL:      os.Getenv("NATS_URL"),
		JWTSecret:    os.Getenv("PLATFORM_JWT_SECRET"),
		ServiceToken: os.Getenv("SERVICE_TOKEN"),
	}
}

// IsDev reports whether the service is running in the dev environment (enables auth shortcuts).
func (c Config) IsDev() bool { return c.Env == "dev" }

// envOr returns the env var or a fallback when unset/empty.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
