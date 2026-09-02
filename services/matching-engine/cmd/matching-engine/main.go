// Command matching-engine is the HTTP entrypoint for the exchange (KW03).
//
// Phase 1 runs the **mock adapter only**: it serves simulated market data for the trading surface
// (KW02) and the Phase 1 index reads, and it refuses every write with 503 EXCHANGE_PAUSED because the
// exchange is paused pending licensing (F22). It holds no order book, opens no database, emits no
// events, and moves no credit — there is deliberately nothing here that could accept a customer order.
//
// The real engine (Redis order book, price-time priority matching, event sourcing, atomic settlement
// against credit-ledger) is specified in SPEC.md and lands at Phase 2 switch-on behind the same
// contract.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/trade1/matching-engine/internal/api"
	"github.com/trade1/matching-engine/internal/auth"
	"github.com/trade1/matching-engine/internal/config"
	"github.com/trade1/matching-engine/internal/obs"
)

// Version is set at build time (-ldflags -X main.Version=...).
var Version = "dev"

// main wires config → credential resolver → HTTP server and serves until killed.
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	cfg := config.Load()

	if cfg.JWTSecret == "" {
		slog.Warn("PLATFORM_JWT_SECRET unset; tenant reads (orders, positions, fills) will be rejected (401)")
	}
	resolver := auth.NewResolver(cfg.JWTSecret)

	// Expose Prometheus /metrics next to the app (same port, cluster-internal) + instrument app routes.
	mux := http.NewServeMux()
	mux.Handle("/metrics", obs.Handler())
	mux.Handle("/", obs.Instrument(api.New(cfg, resolver)))
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("matching-engine listening (mock adapter; order entry paused)",
		"addr", cfg.Addr, "env", cfg.Env, "version", Version)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
