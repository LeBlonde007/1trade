// Command surveillance runs trade surveillance (KW05): it replays and follows the engine's order and
// trade events, raises surveillance.alert.v1 alerts, stores them, and serves them for review.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/trade1/surveillance/internal/api"
	"github.com/trade1/surveillance/internal/config"
	"github.com/trade1/surveillance/internal/detect"
	"github.com/trade1/surveillance/internal/obs"
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

	// The event pipeline starts in the background and keeps retrying: pods start in any order, and
	// surveillance must not give up on the market because NATS came up a second later. Until it is
	// running, /readyz reports not-ready.
	// status wraps the error: atomic.Value refuses to store a nil interface, which "watching" is.
	type status struct{ err error }
	var watching atomic.Value
	watching.Store(status{errors.New("event pipeline not started")})
	if cfg.NATSURL == "" {
		slog.Warn("NATS_URL unset: surveillance is NOT watching the market (review API only)")
		watching.Store(status{errors.New("NATS_URL unset")})
	} else {
		go func() {
			for delay := time.Second; ; delay = min(delay*2, 30*time.Second) {
				r, err := pipeline.Start(cfg.NATSURL, func(pub pipeline.Publisher) *pipeline.Pipeline {
					return pipeline.New(detect.DefaultConfig(), st, pub)
				})
				if err == nil {
					watching.Store(status{})
					_ = r // runs for the life of the process
					return
				}
				watching.Store(status{fmt.Errorf("event pipeline not connected: %w", err)})
				slog.Error("surveillance pipeline not started; retrying — the market is unwatched until it does", "err", err, "retry_in", delay)
				time.Sleep(delay)
			}
		}()
	}
	isWatching := func() error { return watching.Load().(status).err }

	// /metrics beside the API on the same port (cluster-internal), API behind RED instrumentation.
	mux := http.NewServeMux()
	mux.Handle("/metrics", obs.Handler())
	mux.Handle("/", obs.Instrument(api.New(st, cfg.ServiceToken, isWatching)))
	srv := &http.Server{Addr: cfg.Addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	slog.Info("surveillance listening", "addr", cfg.Addr, "env", cfg.Env)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server", "err", err)
		os.Exit(1)
	}
}
