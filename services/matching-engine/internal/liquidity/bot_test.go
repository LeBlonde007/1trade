package liquidity

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/trade1/matching-engine/internal/domain"
	"github.com/trade1/matching-engine/internal/engine"
	"github.com/trade1/matching-engine/internal/settle"
	"github.com/trade1/matching-engine/internal/venue"
)

var t0 = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

// fakeFunder records top-ups.
type fakeFunder struct {
	mu   sync.Mutex
	asks []string
}

// TopUp records the asset asked for.
func (f *fakeFunder) TopUp(_ context.Context, asset, _ string, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asks = append(f.asks, asset)
	return nil
}

// newBot returns a bot on a fresh venue with a fixed reference price per product.
func newBot(mid map[string]float64, risk engine.RiskCheck) (*Bot, *venue.Venue, *time.Time) {
	v := venue.New(engine.New(engine.Config{Risk: risk}), nil, nil, "liq")
	now := t0
	b := &Bot{Venue: v, Tenant: "liq", Now: func() time.Time { return now },
		Mid: func(p domain.Product, _ time.Time) float64 {
			if m, ok := mid[p.ID]; ok {
				return m
			}
			return p.Reference
		}}
	return b, v, &now
}

// TestLadderIsTwoSidedOnTickAndAroundMid checks every tradeable paper book gets Levels bids and asks,
// on tick, bids below and asks above mid, with the first level 0.5% away (a 1% quoted spread).
func TestLadderIsTwoSidedOnTickAndAroundMid(t *testing.T) {
	b, v, _ := newBot(map[string]float64{"H100-SPOT": 3.00}, nil)
	placed := b.Step(context.Background())
	tradeable := 0
	for _, p := range domain.Catalog() {
		if p.Tradeable {
			tradeable++
		}
	}
	if placed != tradeable*6 {
		t.Fatalf("placed %d orders, want %d (3 levels × 2 sides × %d products)", placed, tradeable*6, tradeable)
	}
	bids, asks := v.Depth("H100-SPOT", 10)
	if len(bids) != 3 || len(asks) != 3 {
		t.Fatalf("H100 depth %d/%d, want 3/3", len(bids), len(asks))
	}
	if bids[0].Price != engine.MustFixed("2.98") || asks[0].Price != engine.MustFixed("3.02") {
		t.Fatalf("touch = %s / %s, want 2.98 / 3.02 (±0.5%% of 3.00 on a 0.01 tick)", bids[0].Price, asks[0].Price)
	}
	if bids[2].Price >= bids[1].Price || asks[2].Price <= asks[1].Price {
		t.Fatal("levels are not laddered away from mid")
	}
	if v, _ := v.Depth("H100-SPOT", 1); len(v) == 0 {
		t.Fatal("no bids")
	}
}

// TestRequotesOnlyWhenNeeded checks an unchanged market costs nothing, a small move is ignored, a real
// move re-quotes, and a taken level is replenished.
func TestRequotesOnlyWhenNeeded(t *testing.T) {
	mid := map[string]float64{"H100-SPOT": 3.00}
	b, v, now := newBot(mid, nil)
	b.Step(context.Background())
	if n := b.Step(context.Background()); n != 0 {
		t.Fatalf("an unchanged market re-quoted %d orders", n)
	}
	mid["H100-SPOT"] = 3.01 // 0.33%: below the 0.5% threshold
	if n := b.Step(context.Background()); n != 0 {
		t.Fatalf("a tiny move re-quoted %d orders", n)
	}
	mid["H100-SPOT"] = 3.10
	if n := b.Step(context.Background()); n != 6 {
		t.Fatalf("a 3%% move re-quoted %d orders, want 6", n)
	}
	if bids, _ := v.Depth("H100-SPOT", 10); len(bids) != 3 || bids[0].Price != engine.MustFixed("3.08") {
		t.Fatalf("old ladder not withdrawn: %+v", bids)
	}

	*now = now.Add(time.Second)
	if _, err := v.Submit(engine.SubmitCmd{OrderID: "c1", TenantID: "alice", ProductID: "H100-SPOT", Side: engine.Buy,
		Type: engine.Market, Quantity: engine.MustFixed("1"), IsPaper: true, TS: *now}); err != nil {
		t.Fatal(err)
	}
	if n := b.Step(context.Background()); n != 6 {
		t.Fatalf("a taken level was not replenished (re-quoted %d)", n)
	}
}

// TestRefillsOnlyWhatWasShort checks a ledger refusal triggers a top-up of exactly that asset.
func TestRefillsOnlyWhatWasShort(t *testing.T) {
	f := &fakeFunder{}
	b, _, _ := newBot(nil, func(o engine.Order, _ engine.Hold) string {
		if o.Side == engine.Sell && o.ProductID == "H100-SPOT" {
			return "insufficient_credit"
		}
		return ""
	})
	b.Funder = f
	b.Step(context.Background())
	if len(f.asks) != 3 {
		t.Fatalf("top-ups = %v, want 3 × gpu_h100 (one per refused ask)", f.asks)
	}
	for _, a := range f.asks {
		if a != "gpu_h100" {
			t.Fatalf("refilled %q, want gpu_h100", a)
		}
	}
}

// TestAdoptsRestingOrdersAfterRestart checks a restarted bot withdraws its old ladder instead of
// stacking a second one on top.
func TestAdoptsRestingOrdersAfterRestart(t *testing.T) {
	mid := map[string]float64{"H100-SPOT": 3.00}
	b, v, now := newBot(mid, nil)
	b.Step(context.Background())
	again := &Bot{Venue: v, Tenant: "liq", Now: func() time.Time { return *now }, Mid: b.Mid}
	mid["H100-SPOT"] = 3.20
	again.Step(context.Background())
	if bids, asks := v.Depth("H100-SPOT", 10); len(bids) != 3 || len(asks) != 3 {
		t.Fatalf("after restart depth %d/%d, want 3/3 (old ladder withdrawn)", len(bids), len(asks))
	}
}

// TestStopsQuotingWhileLedgerIsDown checks a ledger outage stops the step at the first refused quote
// (instead of journaling a refused ladder for every product) and flags Run to back off.
func TestStopsQuotingWhileLedgerIsDown(t *testing.T) {
	calls := 0
	b, v, _ := newBot(nil, func(engine.Order, engine.Hold) string { calls++; return settle.ReasonRiskUnavailable })
	if n := b.Step(context.Background()); n != 0 || !b.unavailable {
		t.Fatalf("placed %d, unavailable=%v; want 0 and true", n, b.unavailable)
	}
	if calls != 1 {
		t.Fatalf("asked the ledger %d times during an outage, want 1", calls)
	}
	if bids, asks := v.Depth("H100-SPOT", 10); len(bids)+len(asks) != 0 {
		t.Fatal("quotes rest although none could be reserved")
	}
}
