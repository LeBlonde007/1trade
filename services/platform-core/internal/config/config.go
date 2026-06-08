// Package config loads platform-core configuration from environment variables only.
package config

import (
	"os"
	"strings"
	"time"
)

// Config holds runtime configuration.
type Config struct {
	Env                      string        // dev | staging | prod
	Addr                     string        // listen address, e.g. ":8001"
	DatabaseURL              string        // Postgres DSN
	JWTSecret                string        // HS256 signing secret — SHARED with every service that verifies tokens
	ServiceToken             string        // service-to-service token (guards internal endpoints; auths calls to the ledger)
	CreditLedgerURL          string        // base URL for booking settled purchases (F06)
	StripeWebhookSecret      string        // Stripe webhook signing secret (verifies inbound webhooks)
	StripeSecretKey          string        // Stripe API key (real checkout sessions; empty → mock Stripe)
	BillingAutoSettle        bool          // dev/sandbox: settle MockStripe checkouts inline (no webhook); never true in prod
	KYCAutoApprove           bool          // dev/sandbox: a KYC submission is verified instantly (no compliance back-office); false in prod
	SMTPAddr                 string        // SMTP server host:port for transactional email (empty → email disabled)
	SMTPUser                 string        // SMTP AUTH username (empty → no auth, e.g. Mailpit)
	SMTPPass                 string        // SMTP AUTH password (from a Secret — never hardcoded)
	SMTPTLS                  string        // transport: "implicit" (465) | "starttls" (587) | "" (plain/Mailpit; inferred from port)
	EmailFrom                string        // From address for platform email
	AppBaseURL               string        // public base URL of the web app, for links in emails
	RequireEmailVerification bool          // gate login on a verified email; opt-in (default off), enabled by the deploy only when real SMTP is wired
	TokenTTL                 time.Duration // issued-token lifetime
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
		// Outside prod there is no compliance back-office, so a KYC submission is auto-verified to keep
		// the real-money gate testable end-to-end. Prod requires a real review (manual or IDV vendor).
		KYCAutoApprove: os.Getenv("EXASCALE_ENV") != "prod",
		SMTPAddr:       os.Getenv("SMTP_ADDR"),
		SMTPUser:       os.Getenv("SMTP_USER"),
		SMTPPass:       os.Getenv("SMTP_PASS"),
		SMTPTLS:        os.Getenv("SMTP_TLS"),
		EmailFrom:      envOr("EMAIL_FROM", "noreply@exascale.local"),
		AppBaseURL:     envOr("APP_BASE_URL", "http://localhost:3000"),
		// Gate login on a verified email. OPT-IN (default off) and coupled to a working mailer: the deploy
		// turns it on only when it wires real SMTP, so a sandbox without email creds isn't bricked (no one
		// could verify → no one could log in) and local/CI/e2e keep immediate login.
		RequireEmailVerification: envBool("REQUIRE_EMAIL_VERIFICATION", false),
		TokenTTL:                 ttl,
	}
}

// envBool reads a boolean env var ("true"/"1" → true, "false"/"0" → false), falling back when unset
// or unparseable.
func envBool(key string, fallback bool) bool {
	switch strings.ToLower(os.Getenv(key)) {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	default:
		return fallback
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
