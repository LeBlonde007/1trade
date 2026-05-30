// Package config loads platform-core configuration from environment variables only.
package config

import (
	"os"
	"time"
)

// Config holds runtime configuration.
type Config struct {
	Env                 string        // dev | staging | prod
	Addr                string        // listen address, e.g. ":8001"
	DatabaseURL         string        // Postgres DSN
	JWTSecret           string        // HS256 signing secret — SHARED with every service that verifies tokens
	ServiceToken        string        // service-to-service token (guards internal endpoints; auths calls to the ledger)
	CreditLedgerURL     string        // base URL for booking settled purchases (F06)
	StripeWebhookSecret string        // Stripe webhook signing secret (verifies inbound webhooks)
	StripeSecretKey     string        // Stripe API key (real checkout sessions; empty → mock Stripe)
	TokenTTL            time.Duration // issued-token lifetime
}

// Load reads configuration from the environment with sensible dev defaults.
func Load() Config {
	ttl := 24 * time.Hour
	if v := os.Getenv("TOKEN_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			ttl = d
		}
	}
	return Config{
		Env:                 envOr("EXASCALE_ENV", "dev"),
		Addr:                envOr("PLATFORM_ADDR", ":8001"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		JWTSecret:           os.Getenv("PLATFORM_JWT_SECRET"),
		ServiceToken:        os.Getenv("SERVICE_TOKEN"),
		CreditLedgerURL:     envOr("CREDIT_LEDGER_URL", "http://credit-ledger:8002"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		TokenTTL:            ttl,
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
