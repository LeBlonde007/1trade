// Package portfolio derives a tenant's simulated positions and fill history for the paused exchange.
//
// Like internal/marketdata, everything here is a pure function of (tenant, timestamp): a tenant always
// sees the same book of positions with the same entry prices, marked to the current simulated price.
// No position is stored, no credit moves, and nothing is settled — the exchange is paused and accepts
// no orders, so a real position cannot exist. Every object is paper.
//
// Positions are scoped by tenant id, which the API layer takes from the verified JWT and never from a
// query parameter — one tenant must not be able to read another's book, even a simulated one.
package portfolio

import (
	"math"
	"time"

	"github.com/exascale/matching-engine/internal/domain"
	"github.com/exascale/matching-engine/internal/marketdata"
)

// Position is one open position, marked to the current simulated price.
type Position struct {
	ProductID        string
	Side             string // long | short
	NetQuantity      float64
	AvgEntryPrice    float64
	MarkPrice        float64
	Notional         float64
	UnrealizedPnL    float64
	UnrealizedPnLPct float64
	RealizedPnLTotal float64
}

// Fill is one historical fill.
type Fill struct {
	FillID     string
	OrderID    string
	ProductID  string
	Side       string // buy | sell
	Price      float64
	Quantity   float64
	Notional   float64
	Fee        float64
	Liquidity  string // maker | taker
	ExecutedAt time.Time
}

// feeRate is the simulated taker fee (1%), matching the conversion spread the platform already
// charges on credit conversion (credit-types.md §3). Makers pay half.
const feeRate = 0.01

// maxPositions caps how many products a simulated book holds, so the portfolio screen shows a
// realistic handful rather than every listed product.
const maxPositions = 4

// Positions returns the tenant's simulated open positions at t, marked to the current price. The set of
// products, sides, and sizes is derived from the tenant id, so it is stable for that tenant and
// different across tenants.
func Positions(tenantID string, t time.Time) []Position {
	products := selected(tenantID)
	out := make([]Position, 0, len(products))
	for i, p := range products {
		idx := int64(i)
		mark := marketdata.Price(p.ID, p.Reference, p.Vol, t.Unix())

		// Entry is the price at a point 1–30 days back, so P&L is a real consequence of the price curve
		// rather than an invented number.
		ageDays := 1 + int64(marketdata.Hash01(tenantID+"age"+p.ID, idx)*29)
		entry := marketdata.Price(p.ID, p.Reference, p.Vol, t.Unix()-ageDays*86400)

		side := "long"
		dir := 1.0
		if marketdata.Hash01(tenantID+"side"+p.ID, idx) < 0.3 {
			side = "short"
			dir = -1.0
		}
		qty := marketdata.LogNormalSize(tenantID+"qty"+p.ID, idx, baseQty(p), 0.7)

		pnl := (mark - entry) * qty * dir
		var pnlPct float64
		if entry > 0 {
			pnlPct = (mark - entry) / entry * 100 * dir
		}
		out = append(out, Position{
			ProductID:     p.ID,
			Side:          side,
			NetQuantity:   qty,
			AvgEntryPrice: entry,
			MarkPrice:     mark,
			Notional:      mark * qty,
			UnrealizedPnL: pnl,
			// Realized P&L is a fraction of unrealized, representing earlier partial closes.
			RealizedPnLTotal: pnl * 0.35 * marketdata.Hash01(tenantID+"real"+p.ID, idx),
			UnrealizedPnLPct: pnlPct,
		})
	}
	return out
}

// Fills returns the tenant's simulated fill history at t, newest first, optionally filtered to one
// product. Fills are keyed to fixed time buckets, so paging back through history is stable.
func Fills(tenantID string, t time.Time, productID string, limit int) []Fill {
	if limit < 1 {
		return nil
	}
	products := selected(tenantID)
	if productID != "" {
		filtered := make([]domain.Product, 0, 1)
		for _, p := range products {
			if p.ID == productID {
				filtered = append(filtered, p)
			}
		}
		products = filtered
	}
	if len(products) == 0 {
		return nil
	}

	// One fill every few hours, spread across the tenant's products.
	const fillInterval int64 = 3 * 3600
	nowBucket := t.Unix() / fillInterval
	out := make([]Fill, 0, limit)
	for j := 0; j < limit; j++ {
		bucket := nowBucket - int64(j)
		p := products[int(bucket%int64(len(products))+int64(len(products)))%len(products)]
		offset := int64(marketdata.Hash01(tenantID+"foff", bucket) * float64(fillInterval))
		ts := time.Unix(bucket*fillInterval+offset, 0).UTC()
		if ts.After(t) {
			ts = t
		}
		px := marketdata.Price(p.ID, p.Reference, p.Vol, ts.Unix())
		qty := marketdata.LogNormalSize(tenantID+"fqty", bucket, baseQty(p), 0.8)
		side := "buy"
		if marketdata.Hash01(tenantID+"fside", bucket) < 0.45 {
			side = "sell"
		}
		liquidity := "taker"
		rate := feeRate
		if marketdata.Hash01(tenantID+"fliq", bucket) < 0.5 {
			liquidity = "maker"
			rate = feeRate / 2
		}
		notional := px * qty
		out = append(out, Fill{
			FillID:     marketdata.DeterministicID(tenantID, "fill", bucket),
			OrderID:    marketdata.DeterministicID(tenantID, "order", bucket),
			ProductID:  p.ID,
			Side:       side,
			Price:      px,
			Quantity:   qty,
			Notional:   notional,
			Fee:        notional * rate,
			Liquidity:  liquidity,
			ExecutedAt: ts,
		})
	}
	return out
}

// baseQty is a plausible position size for a product, sized so notionals land in a similar range
// whether the product trades at $0.0004 or $3.
func baseQty(p domain.Product) float64 {
	if p.Reference <= 0 {
		return 1000
	}
	return math.Max(4, 10000/math.Sqrt(p.Reference*1000))
}

// selected picks the products a tenant holds, deterministically from the tenant id. Tradeable products
// only: a simulated book should not hold a forward that is not listed for trading yet.
func selected(tenantID string) []domain.Product {
	all := domain.Catalog()
	tradeable := make([]domain.Product, 0, len(all))
	for _, p := range all {
		if p.Tradeable {
			tradeable = append(tradeable, p)
		}
	}
	if len(tradeable) == 0 {
		return nil
	}
	count := 2 + int(marketdata.Hash01(tenantID+"n", 0)*float64(maxPositions-1))
	if count > len(tradeable) {
		count = len(tradeable)
	}
	// Walk the list from a tenant-specific offset with a co-prime stride, so different tenants hold
	// different (and non-adjacent) products without repeats.
	start := int(marketdata.Hash01(tenantID+"start", 0) * float64(len(tradeable)))
	stride := 1 + int(marketdata.Hash01(tenantID+"stride", 0)*3)
	out := make([]domain.Product, 0, count)
	seen := make(map[string]bool, count)
	for i := 0; len(out) < count && i < len(tradeable)*4; i++ {
		p := tradeable[(start+i*stride)%len(tradeable)]
		if seen[p.ID] {
			continue
		}
		seen[p.ID] = true
		out = append(out, p)
	}
	return out
}
