package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/trade1/matching-engine/internal/config"
	"github.com/trade1/matching-engine/internal/domain"
	"github.com/trade1/matching-engine/internal/engine"
	"github.com/trade1/matching-engine/internal/journal"
	"github.com/trade1/matching-engine/internal/liquidity"
	"github.com/trade1/matching-engine/internal/marketdata"
	"github.com/trade1/matching-engine/internal/outbox"
	"github.com/trade1/matching-engine/internal/settle"
	"github.com/trade1/matching-engine/internal/venue"
)

// startPaper brings up the paper venue (SPEC.md §7.3 cutover, paper only) and its background work:
//
//  1. journal.Recover with ReserveRisk (every order's hold is reserved in the ledger; fails closed),
//     plus the view rebuilt from the replayed events;
//  2. the reservation reconciler, resumed from the journal's tombstones and sweeping the ledger;
//  3. the settlement relay — the journal's events, in order, through settle.Worker, with its own
//     durable cursor so a restart neither skips nor double-applies;
//  4. the NATS relay (trades.executed.v1, orders.state.v1) when NATS is configured;
//  5. the paper liquidity bot, funded once with paper value;
//  6. day-order expiry at 00:00 UTC.
//
// It returns an error rather than a half-started venue: the caller then stays paused.
func startPaper(ctx context.Context, cfg config.Config) (*venue.Venue, error) {
	if cfg.SettleToken == "" {
		return nil, fmt.Errorf("SETTLE_SERVICE_TOKEN is required to settle paper trades")
	}
	store, err := journal.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	ledger := settle.New(cfg.LedgerURL, cfg.SettleToken, cfg.HTTPTimeout)
	rec := &settle.Reconciler{Ledger: ledger, Lister: ledger}
	// The engine is pure (no context): its risk hook runs under the engine lock and bounds each ledger
	// call with its own timeout, so it deliberately does not inherit ctx.
	risk := settle.ReserveRisk(ledger, 3*time.Second) //nolint:contextcheck // see above
	v, ecfg, err := venue.Recover(ctx, store, engine.Config{
		Fees: engine.DefaultFees, Risk: risk, OnUnjournaled: rec.Suspect,
	}, cfg.LiquidityTenant)
	if err != nil {
		store.Close()
		return nil, err
	}
	rec.Engine = v
	rec.Resume(v.Voided())
	go rec.Run(ctx, 30*time.Second)

	worker := &settle.Worker{Ledger: ledger, OnUnsettleable: func(t engine.Trade, err error) {
		slog.Error("paper trade refused by the ledger", "trade_id", t.TradeID, "err", err)
	}}
	settleRelay := outbox.NewHandler(ecfg, store, outbox.NewPGCursor(store.Pool(), "settle"),
		func(ctx context.Context, ev engine.Event) error {
			_, err := worker.Process(ctx, []engine.Event{ev})
			return err
		})
	go follow(ctx, "settlement", settleRelay)

	if cfg.NATSURL != "" {
		go publish(ctx, cfg.NATSURL, ecfg, store)
	}

	bot := &liquidity.Bot{
		Venue: v, Tenant: cfg.LiquidityTenant,
		Mid: func(p domain.Product, t time.Time) float64 { return marketdata.QuoteAt(p, t).Mid },
	}
	go func() {
		if cfg.ServiceToken == "" {
			slog.Warn("SERVICE_TOKEN unset: the paper liquidity account cannot be funded")
		} else {
			funder := &liquidity.Funder{BaseURL: cfg.LedgerURL, Token: cfg.ServiceToken, Tenant: cfg.LiquidityTenant}
			// Quote only once funded: a quote the ledger cannot reserve is refused and journaled for nothing.
			for err := funder.Seed(ctx, "250000.000000", seedCredits()); err != nil; err = funder.Seed(ctx, "250000.000000", seedCredits()) {
				slog.Warn("paper liquidity: seeding failed; retrying", "err", err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
				}
			}
			bot.Funder = funder
		}
		bot.Run(ctx, cfg.LiquidityEvery)
	}()
	go expireDays(ctx, v)

	slog.Info("paper venue open", "epoch", ecfg.Epoch, "liquidity_tenant", cfg.LiquidityTenant, "nats", cfg.NATSURL != "")
	return v, nil
}

// seedCredits is the liquidity account's starting inventory: enough of each credit for many ladders
// (3 levels of $250 per product, 40 times over), at the reference price.
func seedCredits() map[string]string {
	out := map[string]string{}
	for _, p := range domain.Catalog() {
		if p.Tradeable && p.Reference > 0 {
			out[p.CreditType] = engine.Fixed(3 * 40 * 250 / p.Reference * 1e6).String()
		}
	}
	return out
}

// follow steps a relay until ctx ends: quickly while there is work, backing off when idle or failing.
func follow(ctx context.Context, name string, r *outbox.Relay) {
	delay := 200 * time.Millisecond
	for {
		n, err := r.Step(ctx)
		switch {
		case err != nil:
			slog.Warn("relay stalled; retrying", "relay", name, "err", err)
			delay = min(delay*2, 10*time.Second)
		case n > 0:
			delay = 50 * time.Millisecond
		default:
			delay = 200 * time.Millisecond
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

// publish connects to NATS (retrying) and relays the journal's events to JetStream.
func publish(ctx context.Context, url string, ecfg engine.Config, store *journal.Store) {
	for {
		nc, err := nats.Connect(url, nats.MaxReconnects(-1))
		if err == nil {
			sink, serr := outbox.NewJetStreamSink(nc)
			if serr == nil {
				slog.Info("trading events publishing to NATS")
				follow(ctx, "nats", outbox.New(ecfg, store, outbox.NewPGCursor(store.Pool(), "nats"), sink))
				nc.Close()
				return
			}
			nc.Close()
			err = serr
		}
		slog.Warn("NATS unavailable for trading events; retrying", "err", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
	}
}

// expireDays cancels resting day orders at each 00:00 UTC session boundary.
func expireDays(ctx context.Context, v *venue.Venue) {
	for {
		now := time.Now().UTC()
		next := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
		select {
		case <-ctx.Done():
			return
		case <-time.After(next.Sub(now)):
		}
		if n, err := v.ExpireDay(next); err != nil {
			slog.Warn("day-order expiry failed", "err", err)
		} else if n > 0 {
			slog.Info("day orders expired", "count", n)
		}
	}
}
