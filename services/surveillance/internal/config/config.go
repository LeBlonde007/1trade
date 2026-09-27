// Package config loads surveillance configuration from environment variables only.
package config

import "os"

// Config is the service's runtime configuration.
type Config struct {
	Env          string // dev | staging | prod
	Addr         string // listen address
	DatabaseURL  string // Postgres DSN for the alerts table
	NATSURL      string // NATS; empty disables the event pipeline (the review API still serves)
	ServiceToken string // bearer for the internal review API
}

// Load reads the environment with dev defaults.
func Load() Config {
	return Config{
		Env:          envOr("TRADE1_ENV", "dev"),
		Addr:         envOr("SURVEILLANCE_ADDR", ":8088"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		NATSURL:      os.Getenv("NATS_URL"),
		ServiceToken: os.Getenv("SERVICE_TOKEN"),
	}
}

// envOr returns the variable or a fallback when unset.
func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
