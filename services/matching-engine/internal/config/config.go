// Package config loads matching-engine configuration from environment variables only.
package config

import (
	"os"
	"time"
)

// Config holds runtime configuration for the matching engine.
//
// Note what is absent: there is no setting that opens real-money order entry. Real money is refused in
// code (api.paperTenant), so no environment variable — and no operator mistake — can start accepting
// real orders before the licence lands. DatabaseURL decides only whether the *paper* venue runs:
// without a journal the service stays on the paused Phase 1 mock adapter.
type Config struct {
	Env            string        // dev | staging | prod
	Addr           string        // listen address, e.g. ":8087"
	JWTSecret      string        // shared HS256 secret to verify first-party tenant JWTs
	MethodologyURL string        // where clients link for the published index methodology
	HTTPTimeout    time.Duration // upstream HTTP client timeout

	// The paper venue (KW03 cutover, paper only).
	DatabaseURL     string        // the engine journal; empty keeps the service paused
	LedgerURL       string        // credit-ledger base URL (reserve, settle, release)
	SettleToken     string        // the engine's own ledger token (ledger-settle Secret, ADR-0004)
	ServiceToken    string        // shared service token: funds the paper liquidity account only
	NATSURL         string        // JetStream for trading events; empty skips publishing
	LiquidityTenant string        // the paper liquidity account's tenant id
	LiquidityEvery  time.Duration // how often the liquidity bot checks its quotes
}

// Load reads configuration from the environment with sensible dev defaults.
func Load() Config {
	return Config{
		Env:            envOr("TRADE1_ENV", "dev"),
		Addr:           envOr("TRADING_ADDR", ":8087"),
		JWTSecret:      os.Getenv("PLATFORM_JWT_SECRET"),
		MethodologyURL: envOr("INDEX_METHODOLOGY_URL", "/methodology"),
		HTTPTimeout:    5 * time.Second,

		DatabaseURL:     os.Getenv("DATABASE_URL"),
		LedgerURL:       envOr("CREDIT_LEDGER_URL", "http://credit-ledger:8002"),
		SettleToken:     os.Getenv("SETTLE_SERVICE_TOKEN"),
		ServiceToken:    os.Getenv("SERVICE_TOKEN"),
		NATSURL:         os.Getenv("NATS_URL"),
		LiquidityTenant: envOr("PAPER_LIQUIDITY_TENANT_ID", "00000000-0000-4000-8000-00000000b07e"),
		LiquidityEvery:  5 * time.Second,
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
