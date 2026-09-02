// Command compute-control is the HTTP entrypoint for the GPU compute control plane (F12/F13): it
// serves the GPU-type catalog + per-tenant quota, the internal scheduling surface (submit / get /
// cancel jobs) the inference gateway+runtime use to place gang-scheduled pods, and the customer-facing
// on-demand GPU instance lifecycle (create / stop / start / delete). M2/M3 ship the in-memory mock-GPU
// backend behind the same interfaces the real Kueue+Volcano + provisioner backend will implement.
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/trade1/compute-control/internal/api"
	"github.com/trade1/compute-control/internal/auth"
	"github.com/trade1/compute-control/internal/config"
	"github.com/trade1/compute-control/internal/domain"
	"github.com/trade1/compute-control/internal/events"
	"github.com/trade1/compute-control/internal/instance"
	"github.com/trade1/compute-control/internal/obs"
	"github.com/trade1/compute-control/internal/pool"
	"github.com/trade1/compute-control/internal/scheduler"
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

	// One shared GPU pool: internal jobs (scheduler) and customer instances (manager) contend for the
	// same GPUs, so capacity is never double-counted.
	gpuPool := pool.New(map[string]int{domain.CreditH100: cfg.H100Count, domain.CreditH200: cfg.H200Count})

	// Scheduler backend selection. M2/M3 ship "mock" (in-memory, GPU-free); "k8s" (Kueue+Volcano) lands
	// behind the same interface later.
	var sched scheduler.Scheduler
	switch cfg.Scheduler {
	case "mock":
		sched = scheduler.NewMockWithPool(gpuPool, cfg.SupplySource, usage)
		slog.Info("scheduler: mock-GPU", "h100", cfg.H100Count, "h200", cfg.H200Count, "supply", cfg.SupplySource)
	default:
		slog.Error("unsupported scheduler backend (only 'mock' in M2/M3)", "scheduler", cfg.Scheduler)
		os.Exit(1)
	}

	// Customer instance manager (F13) over the same pool. A background ticker meters running instances
	// per interval → compute.usage.v1 → ledger debits the gpu_* tier.
	mgr := instance.NewManager(gpuPool, cfg.SupplySource, usage)
	meterStop := startMetering(mgr, cfg.MeterInterval)
	defer close(meterStop)

	if cfg.JWTSecret == "" {
		slog.Warn("PLATFORM_JWT_SECRET unset; tenant reads will be rejected (401)")
	}
	resolver := auth.NewResolver(cfg.JWTSecret, cfg.ServiceToken)

	// Expose Prometheus /metrics next to the app (same port, cluster-internal) + instrument app routes.
	mux := http.NewServeMux()
	mux.Handle("/metrics", obs.Handler())
	mux.Handle("/", obs.Instrument(api.New(cfg, resolver, sched, mgr)))
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("compute-control listening", "addr", cfg.Addr, "env", cfg.Env, "version", Version)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}

// startMetering runs a ticker that meters running instances every interval (per-interval GPU debits),
// returning a channel that stops it when closed.
func startMetering(mgr *instance.Manager, interval time.Duration) chan struct{} {
	stop := make(chan struct{})
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				mgr.MeterTick()
			}
		}
	}()
	return stop
}
