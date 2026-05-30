// Command inference-gateway is the HTTP entrypoint for the OpenAI-compatible inference API (F08).
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/exascale/inference-gateway/internal/api"
	"github.com/exascale/inference-gateway/internal/config"
	"github.com/exascale/inference-gateway/internal/events"
	"github.com/exascale/inference-gateway/internal/ledger"
	"github.com/exascale/inference-gateway/internal/model"
)

// Version is set at build time (-ldflags -X main.Version=...).
var Version = "dev"

// main wires config → model backend + usage publisher → HTTP server and serves until killed.
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	cfg := config.Load()

	// Usage events drive ledger debits. Prefer NATS; fall back to logging so local dev still runs.
	var usage events.Publisher
	if np, err := events.NewNatsPublisher(cfg.NATSURL); err != nil {
		slog.Warn("NATS unavailable; using log publisher (no debits will flow)", "err", err)
		usage = events.LogPublisher{}
	} else {
		defer np.Close()
		usage = np
	}

	// Backend selection (F09): the real vLLM runtime or the GPU-free mock, behind one interface.
	var backend model.Backend = model.MockBackend{}
	if cfg.InferenceBackend == "vllm" {
		backend = model.NewVLLMBackend(cfg.VLLMBaseURL, cfg.InferenceTimeout)
		slog.Info("inference backend: vllm", "runtime", cfg.VLLMBaseURL)
	} else {
		slog.Info("inference backend: mock")
	}

	// Pre-flight credit guard — only when a JWT secret is configured (it mints the tenant token the
	// ledger verifies). A nil checker disables the pre-flight.
	var credit api.CreditChecker
	if cfg.JWTSecret != "" {
		credit = ledger.NewClient(cfg)
	} else {
		slog.Warn("PLATFORM_JWT_SECRET unset; credit pre-flight disabled")
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.New(cfg, backend, usage, credit),
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("inference-gateway listening", "addr", cfg.Addr, "env", cfg.Env, "version", Version)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
