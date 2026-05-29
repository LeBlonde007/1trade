// Command credit-ledger is the HTTP entrypoint for the credit ledger service (F05).
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/exascale/credit-ledger/internal/api"
	"github.com/exascale/credit-ledger/internal/config"
	"github.com/exascale/credit-ledger/internal/events"
	"github.com/exascale/credit-ledger/internal/store"
)

// Version is set at build time (-ldflags -X main.Version=...).
var Version = "dev"

// main wires config → Postgres store → event publisher → HTTP server and serves until killed.
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	ctx := context.Background()
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect store", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	// Use NATS for credit.tx.v1 when configured; otherwise log-only (best-effort either way).
	var pub events.Publisher = events.LogPublisher{}
	if cfg.NATSURL != "" {
		np, err := events.NewNatsPublisher(cfg.NATSURL)
		if err != nil {
			slog.Error("connect NATS — falling back to log publisher", "err", err)
		} else {
			defer np.Close()
			pub = np
			slog.Info("publishing credit.tx.v1 to NATS", "url", cfg.NATSURL)
		}
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.New(cfg, st, pub),
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Info("credit-ledger listening", "addr", cfg.Addr, "env", cfg.Env, "version", Version)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
