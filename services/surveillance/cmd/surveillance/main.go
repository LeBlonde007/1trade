// Command surveillance runs trade surveillance (KW05): it replays and follows the engine's order and
// trade events, raises surveillance.alert.v1 alerts, stores them, and serves them for review.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/trade1/surveillance/internal/api"
	"github.com/trade1/surveillance/internal/config"
	"github.com/trade1/surveillance/internal/detect"
	"github.com/trade1/surveillance/internal/pipeline"
	"github.com/trade1/surveillance/internal/store"
)

// main wires config → store → event pipeline → review API.
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	st, err := store.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect store", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	if cfg.NATSURL == "" {
		slog.Warn("NATS_URL unset: surveillance is NOT watching the market (review API only)")
	} else {
		r, err := pipeline.Start(cfg.NATSURL, func(pub pipeline.Publisher) *pipeline.Pipeline {
			return pipeline.New(detect.DefaultConfig(), st, pub)
		})
		if err != nil {
			slog.Error("surveillance pipeline not started — the market is unwatched", "err", err)
		} else {
			defer r.Close()
		}
	}

	srv := &http.Server{Addr: cfg.Addr, Handler: api.New(st, cfg.ServiceToken), ReadHeaderTimeout: 5 * time.Second}
	slog.Info("surveillance listening", "addr", cfg.Addr, "env", cfg.Env)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
