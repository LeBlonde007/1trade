// Package api is the HTTP surface of the matching engine (docs/contracts/openapi/trading.yaml, plus the
// Phase 1 index reads from openapi/index.yaml).
//
// Two rules shape every handler here:
//
//   - **Writes are refused, always.** POST /v1/trading/orders and DELETE /v1/trading/orders/{id} return
//     503 EXCHANGE_PAUSED unconditionally. They do not validate, queue, log an intent, or partially
//     apply anything — an order that cannot be filled must not look accepted.
//   - **Everything served is paper.** `is_paper` is hardcoded true on every object, not copied from a
//     request or a claim. There is no code path that emits `is_paper: false`.
//
// Market data reads are public (they expose no tenant data). Orders, positions, and fills require a
// verified tenant JWT and are scoped to the tenant id from that token, never from a parameter.
// Error bodies use the contract's ApiError shape ({code,message}); internals are logged, never leaked.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/exascale/matching-engine/internal/auth"
	"github.com/exascale/matching-engine/internal/config"
	"github.com/exascale/matching-engine/internal/domain"
	"github.com/exascale/matching-engine/internal/marketdata"
	"github.com/exascale/matching-engine/internal/metrics"
	"github.com/exascale/matching-engine/internal/portfolio"
	"github.com/exascale/matching-engine/internal/refindex"
)

// Server wires config + the credential resolver behind one routed handler. The engine holds no state:
// every response is derived from the product catalog and the current time.
type Server struct {
	cfg  config.Config
	auth *auth.Resolver
	mux  *http.ServeMux
	// now is the clock, injectable so tests can pin time and assert that history is stable.
	now func() time.Time
}

// New builds the routed handler.
func New(cfg config.Config, resolver *auth.Resolver) *Server {
	s := &Server{cfg: cfg, auth: resolver, mux: http.NewServeMux(), now: func() time.Time { return time.Now().UTC() }}
	s.routes()
	return s
}

// NewWithClock builds the routed handler with a fixed clock, for deterministic tests.
func NewWithClock(cfg config.Config, resolver *auth.Resolver, now func() time.Time) *Server {
	s := &Server{cfg: cfg, auth: resolver, mux: http.NewServeMux(), now: now}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// routes registers every endpoint from the contracts.
func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	s.mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	s.mux.HandleFunc("GET /v1/trading/products", s.listProducts)
	s.mux.HandleFunc("GET /v1/trading/products/{productId}", s.getProduct)
	s.mux.HandleFunc("GET /v1/trading/products/{productId}/quote", s.getQuote)
	s.mux.HandleFunc("GET /v1/trading/products/{productId}/orderbook", s.getOrderBook)
	s.mux.HandleFunc("GET /v1/trading/products/{productId}/trades", s.listTrades)
	s.mux.HandleFunc("GET /v1/trading/products/{productId}/candles", s.listCandles)

	s.mux.HandleFunc("GET /v1/trading/orders", s.listOrders)
	s.mux.HandleFunc("POST /v1/trading/orders", s.placeOrder)
	s.mux.HandleFunc("DELETE /v1/trading/orders/{orderId}", s.cancelOrder)
	s.mux.HandleFunc("GET /v1/trading/positions", s.listPositions)
	s.mux.HandleFunc("GET /v1/trading/fills", s.listFills)

	s.mux.HandleFunc("GET /v1/index/latest", s.indexLatest)
	s.mux.HandleFunc("GET /v1/index/history", s.indexHistory)
	s.mux.HandleFunc("GET /v1/index/methodology", s.indexMethodology)
}

// listProducts serves the whole catalog plus the exchange status. Public read.
func (s *Server) listProducts(w http.ResponseWriter, _ *http.Request) {
	metrics.RecordMarketData("products")
	catalog := domain.Catalog()
	products := make([]map[string]any, 0, len(catalog))
	for _, p := range catalog {
		products = append(products, productJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"products":        products,
		"exchange_status": statusJSON(domain.Status(s.cfg.MethodologyURL)),
	})
}

// getProduct serves one product with its 24h market summary. Public read.
func (s *Server) getProduct(w http.ResponseWriter, r *http.Request) {
	p, ok := s.product(w, r)
	if !ok {
		return
	}
	metrics.RecordMarketData("product")
	sum := marketdata.SummaryAt(p, s.now())
	writeJSON(w, http.StatusOK, map[string]any{
		"product": productJSON(p),
		"summary": summaryJSON(p, sum),
	})
}

// getQuote serves the two-sided quote for a product. Public read.
func (s *Server) getQuote(w http.ResponseWriter, r *http.Request) {
	p, ok := s.product(w, r)
	if !ok {
		return
	}
	metrics.RecordMarketData("quote")
	q := marketdata.QuoteAt(p, s.now())
	writeJSON(w, http.StatusOK, map[string]any{
		"product_id": p.ID,
		"bid":        dec(q.Bid),
		"ask":        dec(q.Ask),
		"bid_size":   dec(q.BidSize),
		"ask_size":   dec(q.AskSize),
		"mid":        dec(q.Mid),
		"spread":     dec(q.Spread),
		"spread_bps": oneDP(q.SpreadBps),
		"as_of":      ts(q.AsOf),
		"is_paper":   true,
	})
}

// getOrderBook serves aggregated depth for a product. Public read.
func (s *Server) getOrderBook(w http.ResponseWriter, r *http.Request) {
	p, ok := s.product(w, r)
	if !ok {
		return
	}
	depth := intParam(r, "depth", 10, 1, 50)
	metrics.RecordMarketData("orderbook")
	book := marketdata.BookAt(p, s.now(), depth)
	writeJSON(w, http.StatusOK, map[string]any{
		"product_id": p.ID,
		"bids":       levelsJSON(book.Bids),
		"asks":       levelsJSON(book.Asks),
		"as_of":      ts(book.AsOf),
		"is_paper":   true,
	})
}

// listTrades serves recent prints for a product, newest first. Public read.
func (s *Server) listTrades(w http.ResponseWriter, r *http.Request) {
	p, ok := s.product(w, r)
	if !ok {
		return
	}
	limit := intParam(r, "limit", 40, 1, 200)
	metrics.RecordMarketData("trades")
	prints := marketdata.PrintsBefore(p, s.now(), limit)
	out := make([]map[string]any, 0, len(prints))
	for _, pr := range prints {
		out = append(out, map[string]any{
			"trade_id":       pr.TradeID,
			"product_id":     p.ID,
			"price":          dec(pr.Price),
			"quantity":       dec(pr.Quantity),
			"aggressor_side": pr.AggressorSide,
			"block":          pr.Block,
			"executed_at":    ts(pr.ExecutedAt),
			"is_paper":       true,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"trades": out})
}

// listCandles serves OHLCV bars for one interval, oldest first. Public read. An unknown interval is a
// 422 rather than a silent fallback — a chart mislabelled with the wrong interval is worse than an error.
func (s *Server) listCandles(w http.ResponseWriter, r *http.Request) {
	p, ok := s.product(w, r)
	if !ok {
		return
	}
	interval := r.URL.Query().Get("interval")
	if interval == "" {
		interval = "1m"
	}
	secs, valid := marketdata.IntervalSeconds(interval)
	if !valid {
		writeErr(w, http.StatusUnprocessableEntity, "invalid_request", "interval must be one of 1m, 5m, 15m, 1h, 4h, 1d")
		return
	}
	limit := intParam(r, "limit", 200, 1, 500)
	metrics.RecordMarketData("candles")
	candles := marketdata.CandlesBefore(p, s.now(), secs, limit)
	out := make([]map[string]any, 0, len(candles))
	for _, c := range candles {
		out = append(out, map[string]any{
			"time":   c.Time,
			"open":   dec(c.Open),
			"high":   dec(c.High),
			"low":    dec(c.Low),
			"close":  dec(c.Close),
			"volume": dec(c.Volume),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"interval": interval, "candles": out})
}

// listOrders serves the tenant's orders — always empty in Phase 1, because no order has ever been
// accepted. It returns 200 with an empty list rather than 503: the question "what are my orders?" has a
// truthful answer, and it is "none".
func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireTenant(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": []any{}})
}

// orderRequest mirrors the contract's OrderRequest. It is decoded only to label the blocked-attempt
// metric — no field is validated or acted on, since the request is refused either way.
type orderRequest struct {
	ProductID string `json:"product_id"`
	Side      string `json:"side"`
}

// placeOrder refuses order entry with 503 EXCHANGE_PAUSED. This is the single most important handler in
// the service: it is the boundary that keeps an unlicensed exchange from accepting customer orders. It
// authenticates first so an anonymous caller gets 401 rather than learning about the pause, then records
// the attempt (real demand evidence for the F22 licensing decision) and refuses.
func (s *Server) placeOrder(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireTenant(w, r); !ok {
		return
	}
	var req orderRequest
	// A malformed body changes nothing: the response is 503 regardless, so the error is ignored
	// deliberately and the metric falls back to "unknown" labels.
	_ = json.NewDecoder(r.Body).Decode(&req)
	metrics.RecordBlockedOrder(req.ProductID, req.Side)
	slog.Info("order entry refused: exchange paused", "product_id", req.ProductID, "side", req.Side)
	s.writePaused(w)
}

// cancelOrder refuses cancellation with 503 EXCHANGE_PAUSED — there is nothing to cancel.
func (s *Server) cancelOrder(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireTenant(w, r); !ok {
		return
	}
	metrics.CancelAttemptsBlockedTotal.Inc()
	s.writePaused(w)
}

// listPositions serves the tenant's simulated positions, marked to the current price.
func (s *Server) listPositions(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireTenant(w, r)
	if !ok {
		return
	}
	positions := portfolio.Positions(p.TenantID, s.now())
	out := make([]map[string]any, 0, len(positions))
	for _, pos := range positions {
		out = append(out, map[string]any{
			"product_id":         pos.ProductID,
			"side":               pos.Side,
			"net_quantity":       dec(pos.NetQuantity),
			"avg_entry_price":    dec(pos.AvgEntryPrice),
			"mark_price":         dec(pos.MarkPrice),
			"notional":           dec(pos.Notional),
			"unrealized_pnl":     dec(pos.UnrealizedPnL),
			"unrealized_pnl_pct": twoDP(pos.UnrealizedPnLPct),
			"realized_pnl_total": dec(pos.RealizedPnLTotal),
			"is_paper":           true,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"positions": out, "is_paper": true})
}

// listFills serves the tenant's simulated fill history, newest first.
func (s *Server) listFills(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireTenant(w, r)
	if !ok {
		return
	}
	productID := strings.TrimSpace(r.URL.Query().Get("productId"))
	if productID != "" {
		if _, exists := domain.Find(productID); !exists {
			writeErr(w, http.StatusUnprocessableEntity, "invalid_request", "unknown product_id")
			return
		}
	}
	limit := intParam(r, "limit", 50, 1, 200)
	fills := portfolio.Fills(p.TenantID, s.now(), productID, limit)
	out := make([]map[string]any, 0, len(fills))
	for _, f := range fills {
		out = append(out, map[string]any{
			"fill_id":     f.FillID,
			"order_id":    f.OrderID,
			"product_id":  f.ProductID,
			"side":        f.Side,
			"price":       dec(f.Price),
			"quantity":    dec(f.Quantity),
			"notional":    dec(f.Notional),
			"fee":         dec(f.Fee),
			"liquidity":   f.Liquidity,
			"executed_at": ts(f.ExecutedAt),
			"is_paper":    true,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"fills": out})
}

// indexLatest serves the most recent index print. Public read, always flagged as mock.
func (s *Server) indexLatest(w http.ResponseWriter, _ *http.Request) {
	now := s.now()
	print, ok := refindex.Latest(now)
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no index print available")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"print":               printJSON(print),
		"next_publication_at": ts(refindex.NextPublication(now)),
		"is_mock":             true,
	})
}

// indexHistory serves historical index prints, oldest first. Public read, always flagged as mock.
func (s *Server) indexHistory(w http.ResponseWriter, r *http.Request) {
	days := intParam(r, "days", 90, 1, 730)
	prints := refindex.History(s.now(), days)
	out := make([]map[string]any, 0, len(prints))
	for _, p := range prints {
		out = append(out, printJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"prints": out, "is_mock": true})
}

// indexMethodology serves the current methodology version and the disclosed constituent weights.
func (s *Server) indexMethodology(w http.ResponseWriter, _ *http.Request) {
	cons := refindex.Constituents(s.now())
	out := make([]map[string]any, 0, len(cons))
	for _, c := range cons {
		out = append(out, map[string]any{
			"name":        c.Name,
			"credit_type": c.CreditType,
			"weight":      c.Weight,
			"source":      c.Source,
			"observed_at": ts(c.ObservedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version":        refindex.MethodologyVersion,
		"effective_date": refindex.MethodologyEffective,
		"window_minutes": refindex.WindowMinutes,
		"trim_pct":       refindex.TrimPct,
		"volume_floor":   refindex.VolumeFloor,
		"constituents":   out,
		"is_mock":        true,
	})
}

// product resolves the {productId} path value to a catalog product, writing a 404 when it is unknown.
func (s *Server) product(w http.ResponseWriter, r *http.Request) (domain.Product, bool) {
	id := r.PathValue("productId")
	p, ok := domain.Find(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "not_found", "no such product")
		return domain.Product{}, false
	}
	return p, true
}

// requireTenant resolves a first-party tenant JWT; on failure it writes a 401 and returns ok=false.
func (s *Server) requireTenant(w http.ResponseWriter, r *http.Request) (auth.Principal, bool) {
	p, err := s.auth.ResolveJWT(bearer(r))
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "a valid tenant token is required")
		return auth.Principal{}, false
	}
	return p, true
}

// writePaused writes the contract's ExchangePausedError with a 503.
func (s *Server) writePaused(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, map[string]any{
		"code":             domain.PausedCode,
		"message":          domain.PausedMessage,
		"methodology_url":  s.cfg.MethodologyURL,
	})
}

// productJSON renders a Product in the contract shape. is_real_money_enabled is a literal false: no
// product is real-money enabled without the licence, and no input can change that.
func productJSON(p domain.Product) map[string]any {
	return map[string]any{
		"product_id":            p.ID,
		"product_type":          p.ProductType,
		"credit_type":           p.CreditType,
		"name":                  p.Name,
		"description":           p.Description,
		"family":                p.Family,
		"underlying_asset":      orNil(p.UnderlyingAsset),
		"delivery_date":         orNil(deliveryDate(p)),
		"contract_size":         orNil(p.ContractSize),
		"tick_size":             p.TickSize,
		"quote_precision":       p.QuotePrecision,
		"tradeable":             p.Tradeable,
		"is_real_money_enabled": false,
		"is_paper":              true,
	}
}

// deliveryDate returns a forward's settlement date, or empty for spot products. Forwards settle 30 days
// out from the current month's start, which keeps the demo's dated contract plausible over time.
func deliveryDate(p domain.Product) string {
	if p.ProductType != domain.TypeForward {
		return ""
	}
	if p.DeliveryDate != "" {
		return p.DeliveryDate
	}
	return time.Now().UTC().AddDate(0, 0, 30).Format("2006-01-02")
}

// summaryJSON renders a MarketSummary in the contract shape.
func summaryJSON(p domain.Product, s marketdata.Summary) map[string]any {
	return map[string]any{
		"product_id":     p.ID,
		"last":           dec(s.Last),
		"open_24h":       dec(s.Open24h),
		"high_24h":       dec(s.High24h),
		"low_24h":        dec(s.Low24h),
		"change_pct_24h": twoDP(s.ChangePct24h),
		"volume_24h":     dec(s.Volume24h),
		"spread_bps":     oneDP(s.SpreadBps),
		"is_paper":       true,
	}
}

// statusJSON renders an ExchangeStatus in the contract shape.
func statusJSON(st domain.ExchangeStatus) map[string]any {
	return map[string]any{
		"state":            st.State,
		"reason":           st.Reason,
		"methodology_url":  st.MethodologyURL,
	}
}

// levelsJSON renders order-book levels in the contract shape.
func levelsJSON(levels []marketdata.Level) []map[string]any {
	out := make([]map[string]any, 0, len(levels))
	for _, l := range levels {
		out = append(out, map[string]any{
			"price":      dec(l.Price),
			"size":       dec(l.Size),
			"cumulative": dec(l.Cumulative),
		})
	}
	return out
}

// printJSON renders an index print in the contract shape.
func printJSON(p refindex.Print) map[string]any {
	return map[string]any{
		"print_id":            p.PrintID,
		"value":               dec(p.Value),
		"published_at":        ts(p.PublishedAt),
		"methodology_version": p.MethodologyVersion,
		"observation_count":   p.ObservationCount,
		"excluded_count":      p.ExcludedCount,
		"chain_hash":          p.ChainHash,
		"prev_chain_hash":     orNil(p.PrevChainHash),
		"provisional":         p.Provisional,
		"source":              p.Source,
	}
}

// dec formats a value as the contract's fixed-point 6dp Decimal string. Every price, size, and amount
// crosses the wire through here — never as a JSON number, per credit-types.md §4.
func dec(v float64) string { return strconv.FormatFloat(v, 'f', 6, 64) }

// twoDP formats a percentage with 2 decimal places.
func twoDP(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }

// oneDP formats a basis-point value with 1 decimal place.
func oneDP(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }

// ts formats a time as RFC 3339 in UTC.
func ts(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05Z07:00") }

// orNil returns the string or nil when empty (so JSON renders null, matching the nullable contract).
func orNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// intParam reads a bounded integer query parameter, falling back to def when absent, unparseable, or
// out of range. Clamping rather than erroring keeps a demo surface resilient to a stale client.
func intParam(r *http.Request, name string, def, minimum, maximum int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	if n < minimum {
		return minimum
	}
	if n > maximum {
		return maximum
	}
	return n
}

// bearer extracts the token from an Authorization: Bearer <token> header.
func bearer(r *http.Request) string {
	if after, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr writes the contract's ApiError shape ({code, message}).
func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"code": code, "message": msg})
}
