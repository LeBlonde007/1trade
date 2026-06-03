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
	BillingAutoSettle   bool          // dev/sandbox: settle MockStripe checkouts inline (no webhook); never true in prod
	SMTPAddr            string        // SMTP server host:port for transactional email (empty → email disabled)
	EmailFrom           string        // From address for platform email
	AppBaseURL          string        // public base URL of the web app, for links in emails
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
		// No real Stripe key ⇒ MockStripe ⇒ no hosted page / webhook will ever fire, so settle
		// checkouts inline. Real deployments set STRIPE_SECRET_KEY → false → the webhook books.
		BillingAutoSettle: os.Getenv("STRIPE_SECRET_KEY") == "",
		SMTPAddr:            os.Getenv("SMTP_ADDR"),
		EmailFrom:           envOr("EMAIL_FROM", "noreply@exascale.local"),
		AppBaseURL:          envOr("APP_BASE_URL", "http://localhost:3000"),
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
