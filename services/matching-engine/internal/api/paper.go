package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/trade1/matching-engine/internal/auth"
	"github.com/trade1/matching-engine/internal/domain"
	"github.com/trade1/matching-engine/internal/engine"
	"github.com/trade1/matching-engine/internal/marketdata"
	"github.com/trade1/matching-engine/internal/metrics"
	"github.com/trade1/matching-engine/internal/venue"
)

// The paper venue (trading.yaml v1.1). With a venue attached, the trading API serves the real engine:
//
//   - **Paper order entry is open.** is_paper comes from the verified token, never the body. A
//     real-money principal still gets 503 EXCHANGE_PAUSED: nothing here can accept a real order.
//   - Orders, fills and positions are the tenant's real paper activity.
//   - The book, quote and tape are the real paper book. A bar with no trades is filled from the
//     reference walk and labelled `source: reference`, so a chart is never silently simulated.

// paperOrderRequest is the contract's OrderRequest. Unknown fields are refused.
type paperOrderRequest struct {
	ProductID   string  `json:"product_id"`
	Side        string  `json:"side"`
	OrderType   string  `json:"order_type"`
	Quantity    string  `json:"quantity"`
	LimitPrice  *string `json:"limit_price"`
	TimeInForce string  `json:"time_in_force"`
}

// paperTenant authenticates an order-entry caller and admits only paper principals. A real-money
// principal is refused with the paused body: real trading waits on the licence (F22).
func (s *Server) paperTenant(w http.ResponseWriter, r *http.Request) (auth.Principal, bool) {
	p, ok := s.requireTenant(w, r)
	if !ok {
		return p, false
	}
	if !p.IsPaper {
		metrics.RecordBlockedOrder("", "")
		s.writePaused(w)
		return p, false
	}
	if p.TenantID == s.liquidityTenant {
		writeErr(w, http.StatusForbidden, "forbidden", "this account cannot trade through the API")
		return p, false
	}
	return p, true
}

// placePaperOrder accepts a paper order. The order id is derived from the tenant and the
// Idempotency-Key, so a retry returns the same order; the same key with a different order is a 409.
func (s *Server) placePaperOrder(w http.ResponseWriter, r *http.Request) {
	p, ok := s.paperTenant(w, r)
	if !ok {
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 128 {
		writeErr(w, http.StatusBadRequest, "invalid_request", "an Idempotency-Key header (at most 128 characters) is required")
		return
	}
	var req paperOrderRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "invalid_request", "invalid order body")
		return
	}
	cmd, msg := s.submitCmd(p, req, key)
	if msg != "" {
		writeErr(w, http.StatusUnprocessableEntity, "invalid_request", msg)
		return
	}
	res, err := s.venue.Submit(cmd)
	switch {
	case errors.Is(err, engine.ErrOrderIDTaken):
		writeErr(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "that Idempotency-Key is already in use")
		return
	case errors.Is(err, engine.ErrJournal):
		writeErr(w, http.StatusServiceUnavailable, "unavailable", "order entry is briefly unavailable; retry with the same Idempotency-Key")
		return
	case err != nil:
		writeErr(w, http.StatusUnprocessableEntity, "invalid_request", strings.TrimPrefix(err.Error(), "engine: "))
		return
	}
	if res.Duplicate && !sameOrder(res.Order, cmd) {
		writeErr(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "that Idempotency-Key was used for a different order")
		return
	}
	status := http.StatusCreated
	if res.Duplicate {
		status = http.StatusOK
	} else {
		slog.Info("paper order", "tenant_id", p.TenantID, "order_id", res.Order.OrderID, "product_id", cmd.ProductID,
			"side", cmd.Side, "state", res.Order.State, "reason", res.Order.Reason)
	}
	s.writeOrder(w, status, p.TenantID, res.Order.OrderID)
}

// submitCmd validates the body into an engine command, or returns why it is unusable.
func (s *Server) submitCmd(p auth.Principal, req paperOrderRequest, key string) (engine.SubmitCmd, string) {
	qty, err := engine.ParseFixed(strings.TrimSpace(req.Quantity))
	if err != nil || qty <= 0 {
		return engine.SubmitCmd{}, "quantity must be a positive decimal with at most 6 places"
	}
	var price engine.Fixed
	if req.LimitPrice != nil && strings.TrimSpace(*req.LimitPrice) != "" && req.OrderType != string(engine.Market) {
		if price, err = engine.ParseFixed(strings.TrimSpace(*req.LimitPrice)); err != nil {
			return engine.SubmitCmd{}, "limit_price must be a decimal with at most 6 places"
		}
	}
	return engine.SubmitCmd{
		OrderID: orderID(p.TenantID, key), TenantID: p.TenantID, SubAccountID: p.SubAccountID,
		ProductID: req.ProductID, Side: engine.Side(req.Side), Type: engine.OrderType(req.OrderType),
		TIF: engine.TIF(req.TimeInForce), Price: price, Quantity: qty, IsPaper: true, TS: s.now(),
	}, ""
}

// sameOrder reports whether a replayed Idempotency-Key describes the order it created. A limit order's
// time in force counts too (the engine defaults an omitted one to gtc).
func sameOrder(o engine.Order, c engine.SubmitCmd) bool {
	tif := c.TIF
	if o.Type == engine.Limit && tif == "" {
		tif = engine.GTC
	}
	return o.ProductID == c.ProductID && o.Side == c.Side && o.Type == c.Type && o.Quantity == c.Quantity &&
		o.Price == c.Price && (o.Type != engine.Limit || o.TIF == tif)
}

// orderID derives the order's UUID from the tenant and its Idempotency-Key.
func orderID(tenant, key string) string {
	sum := sha256.Sum256([]byte("order:" + tenant + ":" + key))
	h := hex.EncodeToString(sum[:16])
	return h[0:8] + "-" + h[8:12] + "-4" + h[13:16] + "-a" + h[17:20] + "-" + h[20:32]
}

// cancelPaperOrder cancels one of the tenant's open paper orders.
func (s *Server) cancelPaperOrder(w http.ResponseWriter, r *http.Request) {
	p, ok := s.paperTenant(w, r)
	if !ok {
		return
	}
	id := r.PathValue("orderId")
	_, err := s.venue.Cancel(engine.CancelCmd{OrderID: id, TenantID: p.TenantID, TS: s.now()})
	switch {
	case errors.Is(err, engine.ErrOrderNotFound):
		writeErr(w, http.StatusNotFound, "not_found", "no such order")
		return
	case errors.Is(err, engine.ErrOrderNotOpen):
		writeErr(w, http.StatusConflict, "order_not_open", "the order is no longer open")
		return
	case errors.Is(err, engine.ErrJournal):
		writeErr(w, http.StatusServiceUnavailable, "unavailable", "cancel is briefly unavailable; retry")
		return
	case err != nil:
		writeErr(w, http.StatusUnprocessableEntity, "invalid_request", strings.TrimPrefix(err.Error(), "engine: "))
		return
	}
	s.writeOrder(w, http.StatusOK, p.TenantID, id)
}

// writeOrder renders one of the tenant's orders from the view.
func (s *Server) writeOrder(w http.ResponseWriter, status int, tenant, id string) {
	var o venue.OrderView
	var ok bool
	s.venue.Read(func(v *venue.View) { o, ok = v.Get(tenant, id) })
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no such order")
		return
	}
	writeJSON(w, status, orderJSON(o))
}

// listPaperOrders serves the tenant's paper orders, newest first.
func (s *Server) listPaperOrders(w http.ResponseWriter, r *http.Request, p auth.Principal) {
	state := r.URL.Query().Get("state")
	switch state {
	case "", "open", "filled", "cancelled", "rejected":
	default:
		writeErr(w, http.StatusUnprocessableEntity, "invalid_request", "state must be open, filled, cancelled or rejected")
		return
	}
	limit := intParam(r, "limit", 50, 1, 200)
	var orders []venue.OrderView
	s.venue.Read(func(v *venue.View) {
		if state != "open" {
			orders = v.Orders(p.TenantID, state, limit)
			return
		}
		for _, o := range v.Orders(p.TenantID, "", 1<<20) { // open includes partially filled
			if o.State == engine.Accepted || o.State == engine.PartiallyFilled {
				if orders = append(orders, o); len(orders) == limit {
					return
				}
			}
		}
	})
	out := make([]map[string]any, 0, len(orders))
	for _, o := range orders {
		out = append(out, orderJSON(o))
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": out})
}

// orderJSON renders an order in the contract shape.
func orderJSON(o venue.OrderView) map[string]any {
	out := map[string]any{
		"order_id": o.OrderID, "product_id": o.ProductID, "side": o.Side, "order_type": o.Type,
		"quantity": o.Quantity.String(), "filled_quantity": o.Filled.String(), "state": venue.ContractState(o.State),
		"reason": orNil(o.Reason), "limit_price": nil, "avg_fill_price": nil, "time_in_force": orNil(string(o.TIF)),
		"is_paper": true, "created_at": ts(o.CreatedAt), "updated_at": ts(o.UpdatedAt),
	}
	if o.LimitPrice > 0 {
		out["limit_price"] = o.LimitPrice.String()
	}
	if avg, ok := o.AvgFillPrice(); ok {
		out["avg_fill_price"] = avg.String()
	}
	return out
}

// listPaperFills serves the tenant's fills, newest first.
func (s *Server) listPaperFills(w http.ResponseWriter, r *http.Request, p auth.Principal, product string) {
	limit := intParam(r, "limit", 50, 1, 200)
	var fills []venue.Fill
	s.venue.Read(func(v *venue.View) { fills = v.Fills(p.TenantID, product, limit) })
	out := make([]map[string]any, 0, len(fills))
	for _, f := range fills {
		out = append(out, map[string]any{
			"fill_id": f.FillID, "order_id": f.OrderID, "product_id": f.ProductID, "side": f.Side,
			"price": f.Price.String(), "quantity": f.Quantity.String(), "notional": f.Notional.String(),
			"fee": f.Fee.String(), "liquidity": f.Liquidity, "executed_at": ts(f.ExecutedAt), "is_paper": true,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"fills": out})
}

// listPaperPositions serves the tenant's positions, marked to the book's mid.
func (s *Server) listPaperPositions(w http.ResponseWriter, p auth.Principal) {
	var positions []venue.Position
	s.venue.Read(func(v *venue.View) { positions = v.Positions(p.TenantID) })
	out := make([]map[string]any, 0, len(positions))
	for _, pos := range positions {
		prod, ok := domain.Find(pos.ProductID)
		if !ok {
			continue
		}
		mark := s.mark(prod)
		side := "long"
		if pos.Net < 0 {
			side = "short"
		}
		unreal := pos.Unrealized(mark)
		pct := "0.00"
		if pos.Cost > 0 {
			pct = pctString(unreal, pos.Cost)
		}
		net := pos.Net
		if net < 0 {
			net = -net
		}
		notional, _ := engine.Notional(engine.Fixed(net), mark)
		out = append(out, map[string]any{
			"product_id": pos.ProductID, "side": side, "net_quantity": venue.FormatSigned(pos.Net),
			"avg_entry_price": venue.FormatSigned(pos.AvgEntry()), "mark_price": mark.String(),
			"notional": notional.String(), "unrealized_pnl": venue.FormatSigned(unreal), "unrealized_pnl_pct": pct,
			"realized_pnl_total": venue.FormatSigned(pos.RealizedPnL), "is_paper": true,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"positions": out, "is_paper": true})
}

// pctString renders num/den as a signed percentage with 2 decimals, in integer arithmetic.
func pctString(num, den int64) string {
	bp := num * 10_000 / den // hundredths of a percent
	sign := ""
	if bp < 0 {
		sign, bp = "-", -bp
	}
	return fmt.Sprintf("%s%d.%02d", sign, bp/100, bp%100)
}

// mark is the price a position is valued at: the book's mid, else the last trade, else the reference.
func (s *Server) mark(p domain.Product) engine.Fixed {
	bids, asks := s.venue.Depth(p.ID, 1)
	if len(bids) > 0 && len(asks) > 0 {
		return (bids[0].Price + asks[0].Price) / 2
	}
	var last engine.Fixed
	var ok bool
	s.venue.Read(func(v *venue.View) { last, ok = v.Last(p.ID) })
	if ok {
		return last
	}
	return engine.Fixed(marketdata.QuoteAt(p, s.now()).Mid * 1e6)
}

// paperQuote serves the best bid and offer of the paper book (the reference quote when a side is empty).
func (s *Server) paperQuote(w http.ResponseWriter, p domain.Product) {
	bids, asks := s.venue.Depth(p.ID, 1)
	now := s.now()
	if len(bids) == 0 || len(asks) == 0 {
		q := marketdata.QuoteAt(p, now)
		writeJSON(w, http.StatusOK, map[string]any{
			"product_id": p.ID, "bid": dec(q.Bid), "ask": dec(q.Ask), "bid_size": dec(0), "ask_size": dec(0),
			"mid": dec(q.Mid), "spread": dec(q.Spread), "spread_bps": oneDP(q.SpreadBps), "as_of": ts(now),
			"source": "reference", "is_paper": true,
		})
		return
	}
	bid, ask := bids[0].Price, asks[0].Price
	mid := (bid + ask) / 2
	writeJSON(w, http.StatusOK, map[string]any{
		"product_id": p.ID, "bid": bid.String(), "ask": ask.String(), "bid_size": bids[0].Quantity.String(),
		"ask_size": asks[0].Quantity.String(), "mid": mid.String(), "spread": (ask - bid).String(),
		"spread_bps": bps(ask-bid, mid), "as_of": ts(now), "source": "book", "is_paper": true,
	})
}

// bps renders spread/mid in basis points with one decimal, in integer arithmetic.
func bps(spread, mid engine.Fixed) string {
	if mid <= 0 {
		return "0.0"
	}
	tenths := int64(spread) * 100_000 / int64(mid)
	return fmt.Sprintf("%d.%d", tenths/10, tenths%10)
}

// paperBook serves the paper book's aggregated depth with running totals.
func (s *Server) paperBook(w http.ResponseWriter, p domain.Product, depth int) {
	bids, asks := s.venue.Depth(p.ID, depth)
	writeJSON(w, http.StatusOK, map[string]any{
		"product_id": p.ID, "bids": depthJSON(bids), "asks": depthJSON(asks), "as_of": ts(s.now()), "is_paper": true,
	})
}

// depthJSON renders levels with the cumulative size from the touch.
func depthJSON(levels []engine.DepthLevel) []map[string]any {
	out := make([]map[string]any, 0, len(levels))
	var cum engine.Fixed
	for _, l := range levels {
		cum += l.Quantity
		out = append(out, map[string]any{"price": l.Price.String(), "size": l.Quantity.String(), "cumulative": cum.String()})
	}
	return out
}

// paperTrades serves the paper tape, newest first. Counterparties are never exposed.
func (s *Server) paperTrades(w http.ResponseWriter, p domain.Product, limit int) {
	var prints []venue.Print
	s.venue.Read(func(v *venue.View) { prints = v.Prints(p.ID, limit) })
	out := make([]map[string]any, 0, len(prints))
	for _, pr := range prints {
		out = append(out, map[string]any{
			"trade_id": pr.TradeID, "product_id": p.ID, "price": pr.Price.String(), "quantity": pr.Quantity.String(),
			"aggressor_side": pr.AggressorSide, "block": false, "executed_at": ts(pr.ExecutedAt), "is_paper": true,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"trades": out})
}

// paperCandles serves OHLCV bars from paper trades. A bar with no trades comes from the reference walk
// and says so (`source: reference`), so the chart is continuous but never passes simulation off as trading.
func (s *Server) paperCandles(w http.ResponseWriter, p domain.Product, interval string, secs int64, limit int) {
	now := s.now()
	ref := marketdata.CandlesBefore(p, now, secs, limit)
	var bars map[int64]venue.Bar
	if len(ref) > 0 {
		s.venue.Read(func(v *venue.View) { bars = v.Bars(p.ID, secs, ref[0].Time, now.Unix()+secs) })
	}
	out := make([]map[string]any, 0, len(ref))
	for _, c := range ref {
		if b, ok := bars[c.Time]; ok {
			out = append(out, map[string]any{"time": b.Time, "open": b.Open.String(), "high": b.High.String(),
				"low": b.Low.String(), "close": b.Close.String(), "volume": b.Volume.String(), "source": "trades"})
			continue
		}
		out = append(out, map[string]any{"time": c.Time, "open": dec(c.Open), "high": dec(c.High), "low": dec(c.Low),
			"close": dec(c.Close), "volume": dec(0), "source": "reference"})
	}
	writeJSON(w, http.StatusOK, map[string]any{"interval": interval, "candles": out})
}

// paperSummary is the 24h rollup from paper trades; with no trades it is flat at the mark.
func (s *Server) paperSummary(p domain.Product) map[string]any {
	now := s.now()
	mark := s.mark(p)
	var bars map[int64]venue.Bar
	var last engine.Fixed
	var traded bool
	s.venue.Read(func(v *venue.View) {
		bars = v.Bars(p.ID, 60, now.Add(-24*time.Hour).Unix(), now.Unix()+60)
		last, traded = v.Last(p.ID)
	})
	if !traded {
		last = mark
	}
	open, high, low, vol := last, last, last, engine.Fixed(0)
	first := int64(1 << 62)
	for t, b := range bars {
		if t < first {
			first, open = t, b.Open
		}
		high, low, vol = max(high, b.High), min(low, b.Low), vol+b.Volume
	}
	change := "0.00"
	if open > 0 {
		change = pctString(int64(last-open), int64(open))
	}
	bids, asks := s.venue.Depth(p.ID, 1)
	spread := "0.0"
	if len(bids) > 0 && len(asks) > 0 {
		spread = bps(asks[0].Price-bids[0].Price, (asks[0].Price+bids[0].Price)/2)
	}
	return map[string]any{
		"product_id": p.ID, "last": last.String(), "open_24h": open.String(), "high_24h": high.String(),
		"low_24h": low.String(), "change_pct_24h": change, "volume_24h": vol.String(), "spread_bps": spread, "is_paper": true,
	}
}
