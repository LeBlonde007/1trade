// Command platform-core is the HTTP entrypoint for the auth/accounts service (F02/F03).
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/exascale/platform-core/internal/api"
	"github.com/exascale/platform-core/internal/config"
	"github.com/exascale/platform-core/internal/store"
)

// Version is set at build time (-ldflags -X main.Version=...).
var Version = "dev"

// main wires config → Postgres store → HTTP server and serves until killed.
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	if cfg.JWTSecret == "" {
		slog.Error("PLATFORM_JWT_SECRET is required (shared with services that verify tokens)")
		os.Exit(1)
	}

	st, err := store.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect store", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.New(cfg, st),
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("platform-core listening", "addr", cfg.Addr, "env", cfg.Env, "version", Version)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
