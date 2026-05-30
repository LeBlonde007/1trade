// Command inference-gateway is the HTTP entrypoint for the OpenAI-compatible inference API (F08).
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/exascale/inference-gateway/internal/api"
	"github.com/exascale/inference-gateway/internal/config"
)

// Version is set at build time (-ldflags -X main.Version=...).
var Version = "dev"

// main wires config → HTTP server and serves until killed.
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	cfg := config.Load()

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.New(cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("inference-gateway listening", "addr", cfg.Addr, "env", cfg.Env, "version", Version)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
