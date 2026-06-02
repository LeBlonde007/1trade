// Command compute-control is the HTTP entrypoint for the GPU compute control plane (F12): it serves
// the GPU-type catalog + per-tenant quota and the internal scheduling surface (submit / get / cancel
// jobs) that the inference gateway+runtime use to place gang-scheduled pods. M2 ships the MockScheduler
// (in-memory, GPU-free) behind the same interface the real Kueue+Volcano backend will implement.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/exascale/compute-control/internal/api"
	"github.com/exascale/compute-control/internal/auth"
	"github.com/exascale/compute-control/internal/config"
	"github.com/exascale/compute-control/internal/events"
	"github.com/exascale/compute-control/internal/scheduler"
)

// Version is set at build time (-ldflags -X main.Version=...).
var Version = "dev"

// main wires config → usage publisher → scheduler backend → HTTP server and serves until killed.
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	cfg := config.Load()

	// Metered GPU usage drives ledger debits. Prefer NATS; fall back to a log publisher so local dev
	// still runs (no debits flow, but scheduling works end-to-end).
	var usage events.Publisher = events.NoopPublisher{}
	if np, err := events.Open(cfg.NATSURL); err != nil {
		slog.Warn("NATS unavailable; using noop usage publisher (no debits will flow)", "err", err)
	} else {
		defer np.Close()
		usage = np
	}

	// Scheduler backend selection. M2 ships "mock" (in-memory, GPU-free); "k8s" (Kueue+Volcano) lands
	// behind the same interface in M3.
	var sched scheduler.Scheduler
	switch cfg.Scheduler {
	case "mock":
		sched = scheduler.NewMock(cfg.H100Count, cfg.H200Count, cfg.SupplySource, usage)
		slog.Info("scheduler: mock-GPU", "h100", cfg.H100Count, "h200", cfg.H200Count, "supply", cfg.SupplySource)
	default:
		slog.Error("unsupported scheduler backend (only 'mock' in M2)", "scheduler", cfg.Scheduler)
		os.Exit(1)
	}

	if cfg.JWTSecret == "" {
		slog.Warn("PLATFORM_JWT_SECRET unset; tenant reads will be rejected (401)")
	}
	resolver := auth.NewResolver(cfg.JWTSecret, cfg.ServiceToken)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.New(cfg, resolver, sched),
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("compute-control listening", "addr", cfg.Addr, "env", cfg.Env, "version", Version)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
