// Command matching-engine is the HTTP entrypoint for the exchange (KW03).
//
// With a journal (DATABASE_URL) it runs the **paper venue**: the real engine, reservations and
// settlement against credit-ledger, event publishing, and the paper liquidity account (startPaper).
// Paper orders are accepted; real-money orders are refused with 503 EXCHANGE_PAUSED in code, because
// real trading waits on the licence (F22).
//
// Without a journal it runs the Phase 1 **mock adapter**: simulated market data, and every write
// refused with 503 EXCHANGE_PAUSED.
package main

import (
	"context"
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
	if _, err := obs.InitTracing(context.Background(), "matching-engine"); err != nil {
		slog.Warn("tracing disabled", "err", err)
	}

	if cfg.JWTSecret == "" {
		slog.Warn("PLATFORM_JWT_SECRET unset; tenant reads (orders, positions, fills) will be rejected (401)")
	}
	resolver := auth.NewResolver(cfg.JWTSecret)

	handler, mode := http.Handler(api.New(cfg, resolver)), "mock adapter; order entry paused"
	if cfg.DatabaseURL != "" {
		v, err := startPaper(context.Background(), cfg)
		if err != nil {
			slog.Error("paper venue failed to start; staying paused", "err", err)
		} else {
			handler, mode = api.NewPaper(cfg, resolver, v, cfg.LiquidityTenant, nil), "paper venue; real money paused"
		}
	}

	// Expose Prometheus /metrics next to the app (same port, cluster-internal) + instrument app routes.
	mux := http.NewServeMux()
	mux.Handle("/metrics", obs.Handler())
	mux.Handle("/", obs.Instrument(handler))
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("matching-engine listening", "mode", mode, "addr", cfg.Addr, "env", cfg.Env, "version", Version)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
