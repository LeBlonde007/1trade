package liquidity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"math"
	"time"

	"github.com/trade1/matching-engine/internal/domain"
	"github.com/trade1/matching-engine/internal/engine"
	"github.com/trade1/matching-engine/internal/settle"
)

// Venue is what the bot needs from the exchange; *venue.Venue implements it.
type Venue interface {
	Submit(c engine.SubmitCmd) (engine.Result, error)
	Cancel(c engine.CancelCmd) (engine.Result, error)
	Order(orderID, tenantID string) (engine.Order, bool)
	OpenOrders(tenantID string) []engine.Order
}

// TopUpper refills the account's paper inventory (*Funder implements it).
type TopUpper interface {
	TopUp(ctx context.Context, asset, amount string, now time.Time) error
}

// Bot quotes a symmetric ladder around each product's reference price on its paper book, and
// re-quotes when the reference moves or a level has been taken.
type Bot struct {
	Venue  Venue
	Tenant string
	// Mid is the reference price the ladder centres on (the market-data reference walk).
	Mid func(p domain.Product, t time.Time) float64
	Now func() time.Time
	// Funder refills inventory when the ledger refuses an order for lack of it; nil disables refills.
	Funder TopUpper

	Levels        int     // levels per side (default 3)
	HalfSpreadPPM int64   // distance of the first level from mid (default 5_000 = 0.5%, so a 1% spread — KW04)
	StepPPM       int64   // distance between levels (default 2_500)
	RequotePPM    int64   // how far the reference must move before re-quoting (default 5_000 = the half-spread)
	LevelUSD      float64 // paper notional per level (default 250)

	live   map[string][]string     // product → the bot's resting order ids
	anchor map[string]engine.Fixed // product → mid the current ladder was built on
	// unavailable is set when the ledger could not answer a reservation: the step stops early and Run
	// backs off, so an outage does not fill the journal with refused quotes.
	unavailable bool
}

// defaults fills unset tuning with the KW04 base values.
func (b *Bot) defaults() {
	if b.Levels <= 0 {
		b.Levels = 3
	}
	if b.HalfSpreadPPM <= 0 {
		b.HalfSpreadPPM = 5_000
	}
	if b.StepPPM <= 0 {
		b.StepPPM = 2_500
	}
	// Re-quote when the reference has moved by the half-spread: quotes stay around it, and the journal
	// grows by ~6k commands a day across all products instead of ~17k at a quarter of it.
	if b.RequotePPM <= 0 {
		b.RequotePPM = 5_000
	}
	if b.LevelUSD <= 0 {
		b.LevelUSD = 250
	}
	if b.Now == nil {
		b.Now = func() time.Time { return time.Now().UTC() }
	}
}

// Step brings every tradeable product's ladder up to date and returns how many orders it placed.
func (b *Bot) Step(ctx context.Context) int {
	b.defaults()
	if b.live == nil {
		b.adopt()
	}
	now, placed := b.Now(), 0
	b.unavailable = false
	for _, p := range domain.Catalog() {
		if b.unavailable {
			break
		}
		if !p.Tradeable {
			continue
		}
		tick, err := engine.ParseFixed(p.TickSize)
		if err != nil || tick <= 0 {
			continue
		}
		mid := toTick(b.Mid(p, now), tick)
		if mid <= 0 {
			continue
		}
		if b.intact(p.ID) && !moved(b.anchor[p.ID], mid, b.RequotePPM) {
			continue
		}
		b.cancelAll(p.ID, now)
		placed += b.quote(ctx, p, tick, mid, now)
		b.anchor[p.ID] = mid
	}
	return placed
}

// Run steps every interval until ctx ends, backing off (up to a minute) while the ledger is unreachable.
func (b *Bot) Run(ctx context.Context, every time.Duration) {
	wait := every
	for {
		b.Step(ctx)
		if b.unavailable {
			wait = min(wait*2, time.Minute)
			slog.Warn("paper liquidity: ledger unavailable; backing off", "retry_in", wait)
		} else {
			wait = every
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// adopt picks up the bot's resting orders after a restart, so it cancels rather than duplicates them.
func (b *Bot) adopt() {
	b.live, b.anchor = map[string][]string{}, map[string]engine.Fixed{}
	for _, o := range b.Venue.OpenOrders(b.Tenant) {
		b.live[o.ProductID] = append(b.live[o.ProductID], o.OrderID)
	}
}

// intact reports whether every level of a product's ladder is still resting untouched.
func (b *Bot) intact(product string) bool {
	ids := b.live[product]
	if len(ids) != 2*b.Levels {
		return false
	}
	for _, id := range ids {
		o, ok := b.Venue.Order(id, b.Tenant)
		if !ok || o.State != engine.Accepted {
			return false
		}
	}
	return true
}

// cancelAll withdraws the product's resting ladder. An order that already filled is simply gone.
func (b *Bot) cancelAll(product string, now time.Time) {
	for _, id := range b.live[product] {
		if _, err := b.Venue.Cancel(engine.CancelCmd{OrderID: id, TenantID: b.Tenant, TS: now}); err != nil &&
			!errors.Is(err, engine.ErrOrderNotOpen) && !errors.Is(err, engine.ErrOrderNotFound) {
			slog.Warn("paper liquidity: cancel failed", "order_id", id, "err", err)
		}
	}
	b.live[product] = nil
}

// quote places the ladder: Levels bids below mid and Levels asks above, each worth LevelUSD.
func (b *Bot) quote(ctx context.Context, p domain.Product, tick, mid engine.Fixed, now time.Time) int {
	qty := engine.Fixed(math.Floor(b.LevelUSD / float64(mid) * 1e12)) // (USD / (mid/1e6)) credits × 1e6
	if qty <= 0 {
		return 0
	}
	placed := 0
	var prevBid, prevAsk engine.Fixed
	for i := 0; i < b.Levels; i++ {
		off := b.HalfSpreadPPM + int64(i)*b.StepPPM
		bid := floorTick(scale(mid, 1_000_000-off), tick)
		ask := ceilTick(scale(mid, 1_000_000+off), tick)
		// On a coarse tick two levels can round to one price; keep each level at least a tick further out.
		if i > 0 {
			bid, ask = min(bid, prevBid-tick), max(ask, prevAsk+tick)
		}
		prevBid, prevAsk = bid, ask
		for _, lv := range []struct {
			side  engine.Side
			price engine.Fixed
		}{{engine.Buy, bid}, {engine.Sell, ask}} {
			if lv.price <= 0 {
				continue
			}
			r, err := b.Venue.Submit(engine.SubmitCmd{
				OrderID: newID(), TenantID: b.Tenant, ProductID: p.ID, Side: lv.side, Type: engine.Limit,
				TIF: engine.GTC, Price: lv.price, Quantity: qty, IsPaper: true, IsLiquidity: true, TS: now,
			})
			if err != nil {
				slog.Warn("paper liquidity: submit failed", "product_id", p.ID, "err", err)
				continue
			}
			if r.Order.State == engine.Rejected {
				if r.Order.Reason == settle.ReasonRiskUnavailable {
					b.unavailable = true
					return placed
				}
				b.refill(ctx, p, r.Order.Reason, qty, now)
				continue
			}
			if r.Order.Open() {
				b.live[p.ID] = append(b.live[p.ID], r.Order.OrderID)
				placed++
			}
		}
	}
	return placed
}

// refill asks the funder for more of whatever the ledger said was short.
func (b *Bot) refill(ctx context.Context, p domain.Product, reason string, qty engine.Fixed, now time.Time) {
	if b.Funder == nil {
		return
	}
	var err error
	switch reason {
	case settle.ReasonInsufficientCash:
		err = b.Funder.TopUp(ctx, "USD", "50000.000000", now)
	case settle.ReasonInsufficientCredit:
		err = b.Funder.TopUp(ctx, p.CreditType, (qty * engine.Fixed(20*b.Levels)).String(), now)
	default:
		return
	}
	if err != nil {
		slog.Warn("paper liquidity: top-up failed", "product_id", p.ID, "reason", reason, "err", err)
	}
}

// toTick converts a reference price to micro-units on the product's tick. This is the one place a
// float (the reference walk) enters; every order price after it is integer arithmetic.
func toTick(mid float64, tick engine.Fixed) engine.Fixed {
	if mid <= 0 || math.IsNaN(mid) || math.IsInf(mid, 0) {
		return 0
	}
	return engine.Fixed(math.Round(mid*1e6/float64(tick))) * tick
}

// scale returns x × ppm / 1e6.
func scale(x engine.Fixed, ppm int64) engine.Fixed {
	return engine.Fixed(int64(x)/1_000_000*ppm + int64(x)%1_000_000*ppm/1_000_000)
}

// floorTick rounds down to a multiple of tick.
func floorTick(x, tick engine.Fixed) engine.Fixed { return x / tick * tick }

// ceilTick rounds up to a multiple of tick.
func ceilTick(x, tick engine.Fixed) engine.Fixed { return (x + tick - 1) / tick * tick }

// moved reports whether mid differs from the anchor by at least ppm (any change when there is none).
func moved(anchor, mid engine.Fixed, ppm int64) bool {
	if anchor <= 0 {
		return true
	}
	d := int64(mid - anchor)
	if d < 0 {
		d = -d
	}
	return d*1_000_000 >= int64(anchor)*ppm
}

// newID returns a random UUID v4 for a bot order.
func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}
