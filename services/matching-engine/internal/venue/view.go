package venue

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"sort"
	"time"

	"github.com/trade1/matching-engine/internal/engine"
)

// maxPrints bounds each product's tape in memory; older prints still live in the minute bars.
const maxPrints = 1000

// OrderView is one customer order as the API shows it.
type OrderView struct {
	OrderID    string
	TenantID   string
	ProductID  string
	Side       engine.Side
	Type       engine.OrderType
	TIF        engine.TIF
	State      engine.State
	Reason     string
	Quantity   engine.Fixed
	Filled     engine.Fixed
	LimitPrice engine.Fixed // zero for market orders
	notional   engine.Fixed // Σ price × qty over its fills, for the average fill price
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// AvgFillPrice is the volume-weighted fill price, and false when nothing has filled.
func (o OrderView) AvgFillPrice() (engine.Fixed, bool) {
	if o.Filled == 0 {
		return 0, false
	}
	return mulDiv(o.notional, unit, o.Filled), true
}

// Fill is one side of one trade, as the tenant who traded it sees it.
type Fill struct {
	FillID     string
	OrderID    string
	ProductID  string
	Side       engine.Side
	Price      engine.Fixed
	Quantity   engine.Fixed
	Notional   engine.Fixed
	Fee        engine.Fixed
	Liquidity  string
	ExecutedAt time.Time
}

// Print is one trade on the public tape. Counterparties are never exposed.
type Print struct {
	TradeID       string
	ProductID     string
	Price         engine.Fixed
	Quantity      engine.Fixed
	AggressorSide engine.Side
	ExecutedAt    time.Time
}

// Bar is one minute of trading in a product.
type Bar struct {
	Time                   int64 // bar open, Unix seconds
	Open, High, Low, Close engine.Fixed
	Volume                 engine.Fixed
}

// Position is a tenant's net exposure in one product, from its fills. Amounts are signed micro-units.
type Position struct {
	ProductID   string
	Net         int64 // + long, − short (micro-units of the credit)
	Cost        int64 // what the open quantity cost, ≥ 0
	RealizedPnL int64 // closed profit net of fees
}

// AvgEntry is the average price of the open quantity (zero when flat).
func (p Position) AvgEntry() int64 {
	if p.Net == 0 {
		return 0
	}
	return divSigned(p.Cost, int64(unit), abs(p.Net))
}

// Unrealized is the open quantity's profit at mark: value minus cost for a long, the reverse for a short.
func (p Position) Unrealized(mark engine.Fixed) int64 {
	value := divSigned(abs(p.Net), int64(mark), int64(unit))
	if p.Net > 0 {
		return value - p.Cost
	}
	return p.Cost - value
}

// View is the venue's read model. It is only touched under the Venue lock.
type View struct {
	liquidity string
	tif       map[string]engine.TIF           // order id → time in force (from the submit commands)
	orders    map[string]*OrderView           // customer orders by id
	byTenant  map[string][]string             // tenant → its order ids, oldest first
	fills     map[string][]Fill               // tenant → fills, oldest first
	positions map[string]map[string]*Position // tenant → product → position
	prints    map[string][]Print              // product → recent prints, oldest first
	bars      map[string]map[int64]*Bar       // product → minute → bar
	last      map[string]engine.Fixed         // product → last trade price
}

// newView returns an empty view.
func newView(liquidityTenant string) *View {
	return &View{
		liquidity: liquidityTenant, tif: map[string]engine.TIF{}, orders: map[string]*OrderView{},
		byTenant: map[string][]string{}, fills: map[string][]Fill{}, positions: map[string]map[string]*Position{},
		prints: map[string][]Print{}, bars: map[string]map[int64]*Bar{}, last: map[string]engine.Fixed{},
	}
}

// apply folds engine events into the view, in emission order. Only paper books are viewed.
func (v *View) apply(evs []engine.Event) {
	for _, ev := range evs {
		switch {
		case ev.Order != nil && ev.Order.IsPaper:
			v.applyOrder(ev.Order)
		case ev.Trade != nil && ev.Trade.IsPaper:
			v.applyTrade(ev.Trade)
		}
	}
}

// applyOrder records an order transition. The event carries the full order snapshot.
func (v *View) applyOrder(e *engine.OrderEvent) {
	if e.TenantID == v.liquidity {
		return
	}
	o, ok := v.orders[e.OrderID]
	if !ok {
		o = &OrderView{OrderID: e.OrderID, TenantID: e.TenantID, ProductID: e.ProductID, Side: e.Side,
			Type: e.Type, TIF: v.tif[e.OrderID], CreatedAt: e.TS}
		v.orders[e.OrderID] = o
		v.byTenant[e.TenantID] = append(v.byTenant[e.TenantID], e.OrderID)
	}
	o.State, o.Reason, o.Quantity, o.Filled, o.LimitPrice, o.UpdatedAt =
		e.State, e.Reason, e.Quantity, e.FilledQuantity, e.LimitPrice, e.TS
}

// applyTrade records a print, the minute bar, and each customer side's fill and position.
func (v *View) applyTrade(t *engine.Trade) {
	v.prints[t.ProductID] = append(v.prints[t.ProductID], Print{TradeID: t.TradeID, ProductID: t.ProductID,
		Price: t.Price, Quantity: t.Quantity, AggressorSide: t.AggressorSide, ExecutedAt: t.ExecutedAt})
	if n := len(v.prints[t.ProductID]); n > maxPrints {
		v.prints[t.ProductID] = append([]Print(nil), v.prints[t.ProductID][n-maxPrints:]...)
	}
	v.last[t.ProductID] = t.Price

	minute := t.ExecutedAt.Unix() / 60 * 60
	if v.bars[t.ProductID] == nil {
		v.bars[t.ProductID] = map[int64]*Bar{}
	}
	if b := v.bars[t.ProductID][minute]; b == nil {
		v.bars[t.ProductID][minute] = &Bar{Time: minute, Open: t.Price, High: t.Price, Low: t.Price, Close: t.Price, Volume: t.Quantity}
	} else {
		b.High, b.Low, b.Close, b.Volume = max(b.High, t.Price), min(b.Low, t.Price), t.Price, b.Volume+t.Quantity
	}

	notional, _ := engine.Notional(t.Price, t.Quantity)
	for _, side := range []struct {
		c    engine.Counterparty
		side engine.Side
	}{{t.Buyer, engine.Buy}, {t.Seller, engine.Sell}} {
		if side.c.TenantID == v.liquidity {
			continue
		}
		if o := v.orders[side.c.OrderID]; o != nil {
			o.notional += notional
		}
		v.fills[side.c.TenantID] = append(v.fills[side.c.TenantID], Fill{
			FillID: fillID(t.TradeID, side.side), OrderID: side.c.OrderID, ProductID: t.ProductID, Side: side.side,
			Price: t.Price, Quantity: t.Quantity, Notional: notional, Fee: side.c.Fee, Liquidity: side.c.Liquidity,
			ExecutedAt: t.ExecutedAt,
		})
		v.position(side.c.TenantID, t.ProductID).add(side.side, t.Price, t.Quantity, notional, side.c.Fee)
	}
}

// position returns (creating) a tenant's position in a product.
func (v *View) position(tenant, product string) *Position {
	if v.positions[tenant] == nil {
		v.positions[tenant] = map[string]*Position{}
	}
	p := v.positions[tenant][product]
	if p == nil {
		p = &Position{ProductID: product}
		v.positions[tenant][product] = p
	}
	return p
}

// add applies one fill: it grows the position at cost, or closes part of it and realizes the
// difference against the average cost (flipping through zero if the fill is larger). Fees always
// reduce realized profit.
func (p *Position) add(side engine.Side, price, qty, notional, fee engine.Fixed) {
	signed := int64(qty)
	if side == engine.Sell {
		signed = -signed
	}
	p.RealizedPnL -= int64(fee)
	if p.Net == 0 || (p.Net > 0) == (signed > 0) {
		p.Net += signed
		p.Cost += int64(notional)
		return
	}
	closeQty := min(abs(signed), abs(p.Net))
	closedCost := divSigned(p.Cost, closeQty, abs(p.Net))
	proceeds := divSigned(closeQty, int64(price), int64(unit))
	if p.Net > 0 {
		p.RealizedPnL += proceeds - closedCost
		p.Net -= closeQty
	} else {
		p.RealizedPnL += closedCost - proceeds
		p.Net += closeQty
	}
	p.Cost -= closedCost
	if rest := abs(signed) - closeQty; rest > 0 { // the fill flipped the position
		p.Net = rest
		if signed < 0 {
			p.Net = -rest
		}
		p.Cost = divSigned(rest, int64(price), int64(unit))
	}
	if p.Net == 0 {
		p.Cost = 0
	}
}

// Orders returns a tenant's orders, newest first, optionally only one contract state
// ("open" covers accepted and partially filled), at most limit.
func (v *View) Orders(tenant, state string, limit int) []OrderView {
	ids := v.byTenant[tenant]
	out := make([]OrderView, 0, min(limit, len(ids)))
	for i := len(ids) - 1; i >= 0 && len(out) < limit; i-- {
		o := v.orders[ids[i]]
		if state == "" || ContractState(o.State) == state {
			out = append(out, *o)
		}
	}
	return out
}

// Get returns one of a tenant's orders.
func (v *View) Get(tenant, orderID string) (OrderView, bool) {
	o, ok := v.orders[orderID]
	if !ok || o.TenantID != tenant {
		return OrderView{}, false
	}
	return *o, true
}

// Fills returns a tenant's fills, newest first, optionally for one product, at most limit.
func (v *View) Fills(tenant, product string, limit int) []Fill {
	all := v.fills[tenant]
	out := make([]Fill, 0, min(limit, len(all)))
	for i := len(all) - 1; i >= 0 && len(out) < limit; i-- {
		if product == "" || all[i].ProductID == product {
			out = append(out, all[i])
		}
	}
	return out
}

// Positions returns a tenant's non-flat positions and positions with realized profit, by product id.
func (v *View) Positions(tenant string) []Position {
	var out []Position
	for _, p := range v.positions[tenant] {
		if p.Net != 0 || p.RealizedPnL != 0 {
			out = append(out, *p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProductID < out[j].ProductID })
	return out
}

// Prints returns a product's most recent prints, newest first, at most limit.
func (v *View) Prints(product string, limit int) []Print {
	all := v.prints[product]
	out := make([]Print, 0, min(limit, len(all)))
	for i := len(all) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, all[i])
	}
	return out
}

// Last returns a product's last trade price.
func (v *View) Last(product string) (engine.Fixed, bool) {
	p, ok := v.last[product]
	return p, ok
}

// Bars aggregates a product's minute bars into bars of secs seconds whose open is in [from, to).
// Buckets with no trades are absent from the result.
func (v *View) Bars(product string, secs, from, to int64) map[int64]Bar {
	out := map[int64]Bar{}
	var minutes []int64
	for m := range v.bars[product] {
		if m >= from && m < to {
			minutes = append(minutes, m)
		}
	}
	sort.Slice(minutes, func(i, j int) bool { return minutes[i] < minutes[j] })
	for _, m := range minutes {
		mb := v.bars[product][m]
		k := m / secs * secs
		b, ok := out[k]
		if !ok {
			out[k] = Bar{Time: k, Open: mb.Open, High: mb.High, Low: mb.Low, Close: mb.Close, Volume: mb.Volume}
			continue
		}
		b.High, b.Low, b.Close, b.Volume = max(b.High, mb.High), min(b.Low, mb.Low), mb.Close, b.Volume+mb.Volume
		out[k] = b
	}
	return out
}

// ContractState maps an engine state to the trading contract's order state.
func ContractState(s engine.State) string {
	if s == engine.Accepted {
		return "open"
	}
	return string(s)
}

// fillID derives a stable UUID-shaped id for one side of a trade.
func fillID(tradeID string, side engine.Side) string {
	sum := sha256.Sum256([]byte("fill:" + tradeID + ":" + string(side)))
	h := hex.EncodeToString(sum[:16])
	return h[0:8] + "-" + h[8:12] + "-4" + h[13:16] + "-8" + h[17:20] + "-" + h[20:32]
}

// unit is one whole unit in micro-units.
const unit = engine.Fixed(1_000_000)

// mulDiv returns floor(a×b/d) for non-negative fixed-point values.
func mulDiv(a, b, d engine.Fixed) engine.Fixed {
	if d == 0 {
		return 0
	}
	return engine.Fixed(divSigned(int64(a), int64(b), int64(d)))
}

// divSigned returns a×b/d truncated toward zero, without intermediate overflow.
func divSigned(a, b, d int64) int64 {
	if d == 0 {
		return 0
	}
	p := new(big.Int).Mul(big.NewInt(a), big.NewInt(b))
	return p.Quo(p, big.NewInt(d)).Int64()
}

// abs is |x|.
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// FormatSigned renders signed micro-units as a 6-dp decimal string ("-1.250000").
func FormatSigned(x int64) string {
	if x < 0 {
		return "-" + engine.Fixed(-x).String()
	}
	return engine.Fixed(x).String()
}
