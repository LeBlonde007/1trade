package detect

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"time"
)

// orderRec is what the detector remembers about a live order.
type orderRec struct {
	id, tenant, product, side string
	qty, price                Fixed
	accepted                  time.Time
}

// tradeRec is one remembered trade.
type tradeRec struct {
	ts                     time.Time
	id, product            string
	buyer, seller, aggress string // aggress = the aggressor's tenant
	qty, price             Fixed
}

// cancelRec is one remembered user cancel.
type cancelRec struct {
	ts                     time.Time
	orderID, product, side string
	price                  Fixed
	unfilled               bool
}

// acceptRec is one remembered order acceptance.
type acceptRec struct {
	ts                       time.Time
	tenant, product, orderID string
	qty                      Fixed
}

// partition is all state for one of paper / real money. They never share anything.
type partition struct {
	now       time.Time
	orders    map[string]*orderRec
	trades    []tradeRec
	accepts   []acceptRec
	cancels   []cancelRec
	cancelBy  map[string]string    // order id → tenant, for cancel records
	seen      map[string]time.Time // trade ids / order event ids already processed (redelivery)
	nearWash  map[string][]string  // unordered pair|product → trade ids of near round trips in window
	positions map[string]map[string]Fixed
	last      map[string]Fixed
}

// Detector evaluates every order and trade event against the rules. Not safe for concurrent use: feed
// it one event stream in order (surveillance consumes each subject in sequence).
type Detector struct {
	cfg     Config
	parts   map[bool]*partition
	emitted map[string]bool
}

// New returns a detector with the given thresholds.
func New(cfg Config) *Detector {
	return &Detector{cfg: cfg, parts: map[bool]*partition{}, emitted: map[string]bool{}}
}

// part returns the partition for is_paper, creating it on first use.
func (d *Detector) part(paper bool) *partition {
	p := d.parts[paper]
	if p == nil {
		p = &partition{orders: map[string]*orderRec{}, cancelBy: map[string]string{},
			positions: map[string]map[string]Fixed{}, last: map[string]Fixed{}, seen: map[string]time.Time{}, nearWash: map[string][]string{}}
		d.parts[paper] = p
	}
	return p
}

// advance moves the partition's clock to ts (never backwards) and drops history no rule can use.
func (d *Detector) advance(p *partition, ts time.Time) {
	if ts.After(p.now) {
		p.now = ts
	}
	horizon := p.now.Add(-d.cfg.maxWindow())
	i := sort.Search(len(p.trades), func(i int) bool { return !p.trades[i].ts.Before(horizon) })
	p.trades = p.trades[i:]
	j := sort.Search(len(p.accepts), func(i int) bool { return !p.accepts[i].ts.Before(horizon) })
	p.accepts = p.accepts[j:]
	k := sort.Search(len(p.cancels), func(i int) bool { return !p.cancels[i].ts.Before(horizon) })
	for _, c := range p.cancels[:k] {
		delete(p.cancelBy, c.orderID)
	}
	p.cancels = p.cancels[k:]
	for id, ts := range p.seen {
		if ts.Before(horizon) {
			delete(p.seen, id)
		}
	}
}

// firstSight records an event id and reports whether it is new. JetStream delivers at least once, so a
// redelivered event must change nothing — otherwise a replayed trade could pair with its own later
// neighbours and raise a fresh alert. An empty id cannot be deduplicated and is always processed.
func (p *partition) firstSight(id string, ts time.Time) bool {
	if id == "" {
		return true
	}
	if _, dup := p.seen[id]; dup {
		return false
	}
	p.seen[id] = ts
	return true
}

// OnOrder feeds one orders.state.v1 event and returns any alerts it completes.
func (d *Detector) OnOrder(o Order) []Alert {
	p := d.part(o.IsPaper)
	d.advance(p, o.TS)
	if o.EventID != "" && !p.firstSight("order-event:"+o.EventID, o.TS) {
		return nil
	}
	var out []Alert
	switch o.State {
	case "accepted":
		price := Fixed(0)
		if o.LimitPrice != nil {
			price = fx(*o.LimitPrice)
		}
		p.orders[o.OrderID] = &orderRec{id: o.OrderID, tenant: o.TenantID, product: o.ProductID, side: o.Side,
			qty: fx(o.Quantity), price: price, accepted: o.TS}
		p.accepts = append(p.accepts, acceptRec{ts: o.TS, tenant: o.TenantID, product: o.ProductID, orderID: o.OrderID, qty: fx(o.Quantity)})
		out = append(out, d.excessiveCancellation(p, o)...)
	case "cancelled":
		rec := p.orders[o.OrderID]
		delete(p.orders, o.OrderID)
		// Only a customer's own cancel is a signal. The engine's automatic cancels (IOC / market
		// remainders, FOK misses, self-trade prevention) are not the customer withdrawing liquidity.
		if rec == nil || o.Reason == nil || *o.Reason != "user_cancel" {
			break
		}
		filled := fx(o.FilledQuantity)
		p.cancels = append(p.cancels, cancelRec{ts: o.TS, orderID: o.OrderID, product: rec.product, side: rec.side, price: rec.price, unfilled: filled == 0})
		p.cancelBy[o.OrderID] = rec.tenant
		out = append(out, d.spoofing(p, rec, filled, o.TS, o.IsPaper)...)
		out = append(out, d.layering(p, rec, o.TS, o.IsPaper)...)
	case "filled", "rejected":
		delete(p.orders, o.OrderID)
	}
	return out
}

// OnTrade feeds one trades.executed.v1 event and returns any alerts it completes.
func (d *Detector) OnTrade(t Trade) []Alert {
	p := d.part(t.IsPaper)
	d.advance(p, t.ExecutedAt)
	if !p.firstSight("trade:"+t.TradeID, t.ExecutedAt) {
		return nil
	}
	qty, price := fx(t.Quantity), fx(t.Price)
	aggressor := t.Buyer.TenantID
	if t.AggressorSide == "sell" {
		aggressor = t.Seller.TenantID
	}
	p.trades = append(p.trades, tradeRec{ts: t.ExecutedAt, id: t.TradeID, product: t.ProductID,
		buyer: t.Buyer.TenantID, seller: t.Seller.TenantID, aggress: aggressor, qty: qty, price: price})
	p.last[t.ProductID] = price
	d.addPosition(p, t.Buyer.TenantID, t.ProductID, qty)
	d.addPosition(p, t.Seller.TenantID, t.ProductID, -qty)

	var out []Alert
	out = append(out, d.washTrade(p, t, qty, price)...)
	out = append(out, d.markingTheClose(p, t)...)
	out = append(out, d.crossProduct(p, t, aggressor)...)
	out = append(out, d.positionLimit(p, t, t.Buyer.TenantID)...)
	out = append(out, d.positionLimit(p, t, t.Seller.TenantID)...)
	return out
}

// addPosition adjusts a tenant's net position in a product.
func (d *Detector) addPosition(p *partition, tenant, product string, delta Fixed) {
	m := p.positions[tenant]
	if m == nil {
		m = map[string]Fixed{}
		p.positions[tenant] = m
	}
	m[product] += delta
}

// owner resolves a tenant's beneficial owner.
func (d *Detector) owner(tenant string) string {
	if o, ok := d.cfg.BeneficialOwner[tenant]; ok && o != "" {
		return o
	}
	return tenant
}

// emit finalises an alert: deterministic id from (rule, tenant, product, paper, key), deduped so a
// replayed stream never raises it twice. Returns nil if already raised.
func (d *Detector) emit(a Alert, key string) []Alert {
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%s|%s|%t|%s", a.Rule, a.TenantID, a.ProductID, a.IsPaper, key))
	a.AlertID = "sva_" + hex.EncodeToString(sum[:])[:24]
	if d.emitted[a.AlertID] {
		return nil
	}
	d.emitted[a.AlertID] = true
	if a.TradeIDs == nil {
		a.TradeIDs = []string{}
	}
	if a.OrderIDs == nil {
		a.OrderIDs = []string{}
	}
	a.WindowStart, a.WindowEnd, a.DetectedAt = a.WindowStart.UTC(), a.WindowEnd.UTC(), a.DetectedAt.UTC()
	return []Alert{a}
}

// bucket names the fixed window of length w containing ts, for dedupe keys.
func bucket(ts time.Time, w time.Duration) (time.Time, string) {
	start := ts.Truncate(w)
	return start, start.UTC().Format(time.RFC3339)
}

// ppm returns a/b in parts per million (0 when b is 0), without overflow.
func ppm(a, b Fixed) int64 {
	if b == 0 {
		return 0
	}
	n := new(big.Int).Mul(big.NewInt(int64(a)), big.NewInt(1_000_000))
	return n.Quo(n, big.NewInt(int64(b))).Int64()
}

// ptr returns a pointer to s (for nullable payload fields).
func ptr(s string) *string { return &s }

// --- rules -------------------------------------------------------------------------------------

// washTrade flags a trade whose two sides share a beneficial owner, and a round trip: the same two
// parties trading the same size back the other way within the window, near the same price.
func (d *Detector) washTrade(p *partition, t Trade, qty, price Fixed) []Alert {
	b, s := t.Buyer.TenantID, t.Seller.TenantID
	if d.owner(b) == d.owner(s) {
		return d.emit(Alert{Rule: RuleWash, Severity: "high", Action: ActionReview, TenantID: b, CounterpartyTenantID: ptr(s),
			ProductID: t.ProductID, IsPaper: t.IsPaper, WindowStart: t.ExecutedAt, WindowEnd: t.ExecutedAt, DetectedAt: t.ExecutedAt,
			Evidence: map[string]any{"pattern": "same_beneficial_owner", "owner": d.owner(b), "quantity": qty.String(), "price": price.String()},
			TradeIDs: []string{t.TradeID}}, "trade:"+t.TradeID)
	}
	from := t.ExecutedAt.Add(-d.cfg.WashWindow)
	for i := len(p.trades) - 2; i >= 0; i-- { // the last element is this trade
		r := p.trades[i]
		if r.ts.Before(from) {
			break
		}
		if r.product != t.ProductID || r.buyer != s || r.seller != b {
			continue
		}
		qtyDiff := ppm((r.qty - qty).abs(), qty)
		priceBps := ppm((r.price-price).abs(), price) / 100
		if qtyDiff > d.cfg.WashQtyTolerancePPM || priceBps > d.cfg.WashPriceBps {
			continue
		}
		exact := qtyDiff == 0 && priceBps == 0 && r.price == price
		pair := b + "|" + s
		if s < b {
			pair = s + "|" + b
		}
		pairKey := pair + "|" + t.ProductID
		if !exact {
			// Keep only near round trips still inside the window, then add this one.
			kept := p.nearWash[pairKey][:0]
			for _, id := range p.nearWash[pairKey] {
				if tr := p.tradeByID(id); tr != nil && !tr.ts.Before(from) {
					kept = append(kept, id)
				}
			}
			p.nearWash[pairKey] = append(kept, t.TradeID)
			if len(p.nearWash[pairKey]) < d.cfg.WashMinNearPairs {
				return nil
			}
		}
		ids := []string{r.id, t.TradeID}
		key := "pair:" + r.id + ":" + t.TradeID
		pattern := "round_trip_exact"
		if !exact {
			ids = append([]string{r.id}, p.nearWash[pairKey]...)
			_, bk := bucket(t.ExecutedAt, d.cfg.WashWindow)
			key, pattern = "near:"+pairKey+":"+bk, "round_trip_repeated"
		}
		return d.emit(Alert{Rule: RuleWash, Severity: "medium", Action: ActionReview, TenantID: b, CounterpartyTenantID: ptr(s),
			ProductID: t.ProductID, IsPaper: t.IsPaper, WindowStart: r.ts, WindowEnd: t.ExecutedAt, DetectedAt: t.ExecutedAt,
			Evidence: map[string]any{"pattern": pattern, "out_quantity": r.qty.String(), "back_quantity": qty.String(),
				"out_price": r.price.String(), "back_price": price.String(), "quantity_diff_ppm": qtyDiff, "price_diff_bps": priceBps,
				"near_round_trips": len(p.nearWash[pairKey]), "window_seconds": int64(d.cfg.WashWindow / time.Second)},
			TradeIDs: ids}, key)
	}
	return nil
}

// tradeByID finds a remembered trade (nil once it has aged out of every window).
func (p *partition) tradeByID(id string) *tradeRec {
	for i := len(p.trades) - 1; i >= 0; i-- {
		if p.trades[i].id == id {
			return &p.trades[i]
		}
	}
	return nil
}

// spoofing flags a large order cancelled quickly and mostly unfilled while its owner traded the other
// side during its life — size shown to move the market, never meant to trade.
func (d *Detector) spoofing(p *partition, rec *orderRec, filled Fixed, now time.Time, paper bool) []Alert {
	life := now.Sub(rec.accepted)
	if life > d.cfg.SpoofMaxLifetime || ppm(filled, rec.qty) > d.cfg.SpoofMaxFillPPM {
		return nil
	}
	var total Fixed
	n := 0
	for _, a := range p.accepts {
		if a.product == rec.product && a.orderID != rec.id {
			total += a.qty
			n++
		}
	}
	if n < d.cfg.SpoofMinSamples {
		return nil
	}
	avg := total / Fixed(n)
	if avg <= 0 || rec.qty < avg*Fixed(d.cfg.SpoofSizeMultiple) {
		return nil
	}
	ids := make([]string, 0, 4)
	var opposite Fixed
	for _, t := range p.trades {
		if t.product != rec.product || t.ts.Before(rec.accepted) || t.ts.After(now) {
			continue
		}
		if (rec.side == "buy" && t.seller == rec.tenant) || (rec.side == "sell" && t.buyer == rec.tenant) {
			ids = append(ids, t.id)
			opposite += t.qty
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return d.emit(Alert{Rule: RuleSpoofing, Severity: "high", Action: ActionRateLimit, TenantID: rec.tenant, ProductID: rec.product,
		IsPaper: paper, WindowStart: rec.accepted, WindowEnd: now, DetectedAt: now,
		Evidence: map[string]any{"order_quantity": rec.qty.String(), "average_order_quantity": avg.String(),
			"size_multiple_threshold": d.cfg.SpoofSizeMultiple, "lifetime_ms": life.Milliseconds(),
			"filled_ppm": ppm(filled, rec.qty), "opposite_side_traded": opposite.String(), "side": rec.side},
		TradeIDs: ids, OrderIDs: []string{rec.id}}, "order:"+rec.id)
}

// layering flags several unfilled orders cancelled on one side at several price levels within the
// window while their owner traded the other side — false depth, stacked and pulled.
func (d *Detector) layering(p *partition, rec *orderRec, now time.Time, paper bool) []Alert {
	from := now.Add(-d.cfg.LayerWindow)
	levels := map[Fixed]bool{}
	ids := make([]string, 0, d.cfg.LayerMinOrders)
	for _, c := range p.cancels {
		if c.ts.Before(from) || !c.unfilled || c.product != rec.product || c.side != rec.side || p.cancelBy[c.orderID] != rec.tenant {
			continue
		}
		levels[c.price] = true
		ids = append(ids, c.orderID)
	}
	if len(ids) < d.cfg.LayerMinOrders || len(levels) < d.cfg.LayerMinLevels {
		return nil
	}
	trades := make([]string, 0, 4)
	for _, t := range p.trades {
		if t.product == rec.product && !t.ts.Before(from) &&
			((rec.side == "buy" && t.seller == rec.tenant) || (rec.side == "sell" && t.buyer == rec.tenant)) {
			trades = append(trades, t.id)
		}
	}
	if len(trades) == 0 {
		return nil
	}
	start, key := bucket(now, d.cfg.LayerWindow)
	return d.emit(Alert{Rule: RuleLayering, Severity: "high", Action: ActionReview, TenantID: rec.tenant, ProductID: rec.product,
		IsPaper: paper, WindowStart: start, WindowEnd: now, DetectedAt: now,
		Evidence: map[string]any{"cancelled_unfilled_orders": len(ids), "distinct_price_levels": len(levels), "side": rec.side,
			"min_orders": d.cfg.LayerMinOrders, "min_levels": d.cfg.LayerMinLevels, "window_seconds": int64(d.cfg.LayerWindow / time.Second)},
		TradeIDs: trades, OrderIDs: ids}, key)
}

// markingTheClose flags a tenant whose share of a product's volume in the window before the daily index
// print crosses the threshold, and asks the index to exclude those observations.
func (d *Detector) markingTheClose(p *partition, t Trade) []Alert {
	ts := t.ExecutedAt.UTC()
	closeAt := time.Date(ts.Year(), ts.Month(), ts.Day(), d.cfg.CloseHourUTC, 0, 0, 0, time.UTC)
	from := closeAt.Add(-d.cfg.CloseWindow)
	if ts.Before(from) || !ts.Before(closeAt) {
		return nil
	}
	var total Fixed
	byTenant := map[string]Fixed{}
	ids := map[string][]string{}
	for _, r := range p.trades {
		if r.product != t.ProductID || r.ts.Before(from) || r.ts.After(ts) {
			continue
		}
		total += r.qty
		for _, who := range []string{r.buyer, r.seller} {
			byTenant[who] += r.qty
			ids[who] = append(ids[who], r.id)
		}
	}
	if total < d.cfg.CloseMinVolume {
		return nil
	}
	var out []Alert
	for _, who := range []string{t.Buyer.TenantID, t.Seller.TenantID} {
		share := ppm(byTenant[who], total)
		if share < d.cfg.CloseSharePPM {
			continue
		}
		out = append(out, d.emit(Alert{Rule: RuleMarkClose, Severity: "high", Action: ActionExclude, TenantID: who, ProductID: t.ProductID,
			IsPaper: t.IsPaper, WindowStart: from, WindowEnd: closeAt, DetectedAt: t.ExecutedAt,
			Evidence: map[string]any{"tenant_volume": byTenant[who].String(), "window_volume": total.String(), "share_ppm": share,
				"threshold_ppm": d.cfg.CloseSharePPM, "index_print_at": closeAt.Format(time.RFC3339)},
			TradeIDs: ids[who]}, closeAt.Format(time.RFC3339))...)
	}
	return out
}

// crossProduct flags a tenant that dominates a product's aggressive volume, moves its price in its own
// direction, and holds a same-direction position in a product whose value depends on it.
func (d *Detector) crossProduct(p *partition, t Trade, aggressor string) []Alert {
	related := d.cfg.Related[t.ProductID]
	if len(related) == 0 {
		return nil
	}
	from := t.ExecutedAt.Add(-d.cfg.CrossWindow)
	var total, mine, net Fixed
	var first Fixed
	var ids []string
	for _, r := range p.trades {
		if r.product != t.ProductID || r.ts.Before(from) {
			continue
		}
		if first == 0 {
			first = r.price
		}
		total += r.qty
		if r.aggress == aggressor {
			mine += r.qty
			ids = append(ids, r.id)
			if r.buyer == aggressor {
				net += r.qty
			} else {
				net -= r.qty
			}
		}
	}
	share := ppm(mine, total)
	if share < d.cfg.CrossSharePPM || first == 0 || net == 0 {
		return nil
	}
	moveBps := ppm(fx(t.Price)-first, first) / 100
	if (net > 0 && moveBps < d.cfg.CrossMinMoveBps) || (net < 0 && -moveBps < d.cfg.CrossMinMoveBps) {
		return nil
	}
	var out []Alert
	for _, b := range related {
		pos := p.positions[aggressor][b]
		if pos == 0 || (pos > 0) != (net > 0) {
			continue
		}
		start, key := bucket(t.ExecutedAt, d.cfg.CrossWindow)
		out = append(out, d.emit(Alert{Rule: RuleCrossProduct, Severity: "medium", Action: ActionReview, TenantID: aggressor,
			ProductID: t.ProductID, RelatedProductID: ptr(b), IsPaper: t.IsPaper, WindowStart: start, WindowEnd: t.ExecutedAt,
			DetectedAt: t.ExecutedAt,
			Evidence: map[string]any{"aggressor_share_ppm": share, "threshold_ppm": d.cfg.CrossSharePPM, "price_move_bps": moveBps,
				"min_move_bps": d.cfg.CrossMinMoveBps, "net_aggressor_quantity": net.String(), "related_position": pos.String()},
			TradeIDs: ids}, key+"|"+b)...)
	}
	return out
}

// excessiveCancellation flags a tenant whose order-to-trade ratio on a product crosses the threshold
// within the window: churning orders that almost never trade.
func (d *Detector) excessiveCancellation(p *partition, o Order) []Alert {
	from := o.TS.Add(-d.cfg.OTRWindow)
	orders := 0
	for _, a := range p.accepts {
		if a.tenant == o.TenantID && a.product == o.ProductID && !a.ts.Before(from) {
			orders++
		}
	}
	if orders < d.cfg.OTRMinOrders {
		return nil
	}
	trades := 0
	ids := make([]string, 0, 8)
	for _, t := range p.trades {
		if t.product == o.ProductID && !t.ts.Before(from) && (t.buyer == o.TenantID || t.seller == o.TenantID) {
			trades++
			ids = append(ids, t.id)
		}
	}
	r := ratio(int64(orders), int64(max(trades, 1)))
	if r < d.cfg.OTRThreshold {
		return nil
	}
	start, key := bucket(o.TS, d.cfg.OTRWindow)
	return d.emit(Alert{Rule: RuleExcessCancel, Severity: "medium", Action: ActionRateLimit, TenantID: o.TenantID,
		ProductID: o.ProductID, IsPaper: o.IsPaper, WindowStart: start, WindowEnd: o.TS, DetectedAt: o.TS,
		Evidence: map[string]any{"orders": orders, "trades": trades, "order_to_trade_ratio": r.String(), "threshold": d.cfg.OTRThreshold.String(),
			"window_seconds": int64(d.cfg.OTRWindow / time.Second)},
		TradeIDs: ids}, key)
}

// positionLimit flags a tenant whose net position in a product, or gross exposure across products,
// exceeds its limit.
func (d *Detector) positionLimit(p *partition, t Trade, tenant string) []Alert {
	var out []Alert
	start, key := bucket(t.ExecutedAt, d.cfg.LimitBucket)
	pos := p.positions[tenant][t.ProductID]
	limit, ok := d.cfg.MaxPosition[t.ProductID]
	if !ok {
		limit = d.cfg.DefaultMaxPosition
	}
	if limit > 0 && pos.abs() > limit {
		out = append(out, d.emit(Alert{Rule: RulePosLimit, Severity: "high", Action: ActionSuspendRec, TenantID: tenant,
			ProductID: t.ProductID, IsPaper: t.IsPaper, WindowStart: start, WindowEnd: t.ExecutedAt, DetectedAt: t.ExecutedAt,
			Evidence: map[string]any{"limit": "net_position", "position": pos.String(), "max": limit.String()},
			TradeIDs: []string{t.TradeID}}, key)...)
	}
	if d.cfg.MaxGrossNotional > 0 {
		var gross Fixed
		for prod, q := range p.positions[tenant] {
			gross += mul(q.abs(), p.last[prod])
		}
		if gross > d.cfg.MaxGrossNotional {
			out = append(out, d.emit(Alert{Rule: RulePosLimit, Severity: "high", Action: ActionSuspendRec, TenantID: tenant,
				ProductID: "*", IsPaper: t.IsPaper, WindowStart: start, WindowEnd: t.ExecutedAt, DetectedAt: t.ExecutedAt,
				Evidence: map[string]any{"limit": "gross_exposure", "gross_notional": gross.String(), "max": d.cfg.MaxGrossNotional.String()},
				TradeIDs: []string{t.TradeID}}, key)...)
		}
	}
	return out
}
