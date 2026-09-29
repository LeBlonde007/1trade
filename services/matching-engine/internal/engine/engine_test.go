package engine

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

// at returns t0 plus n milliseconds, so each command in a test has a distinct timestamp.
func at(n int) time.Time { return t0.Add(time.Duration(n) * time.Millisecond) }

// px and qty are short helpers for fixed-point literals.
func px(s string) Fixed  { return MustFixed(s) }
func qty(s string) Fixed { return MustFixed(s) }

// limit builds a paper GTC limit order on EAI-IDX (tick 0.000001).
func limit(id, tenant string, side Side, price, q string, n int) SubmitCmd {
	return SubmitCmd{OrderID: id, TenantID: tenant, ProductID: "EAI-IDX", Side: side, Type: Limit,
		Price: px(price), Quantity: qty(q), IsPaper: true, TS: at(n)}
}

// mustSubmit submits and fails the test on a validation error.
func mustSubmit(t *testing.T, e *Engine, c SubmitCmd) Result {
	t.Helper()
	r, err := e.Submit(c)
	if err != nil {
		t.Fatalf("submit %s: %v", c.OrderID, err)
	}
	return r
}

// trades extracts the trades from a result's events.
func trades(evs []Event) []Trade {
	var out []Trade
	for _, ev := range evs {
		if ev.Trade != nil {
			out = append(out, *ev.Trade)
		}
	}
	return out
}

// TestPriceTimePriority checks that a taker fills the best price first and, within a price, the
// earliest order first — at the resting price, not the taker's.
func TestPriceTimePriority(t *testing.T) {
	e := New(Config{})
	mustSubmit(t, e, limit("a1", "mm1", Sell, "0.001010", "100", 1))
	mustSubmit(t, e, limit("a2", "mm2", Sell, "0.001005", "100", 2)) // better price, later
	mustSubmit(t, e, limit("a3", "mm3", Sell, "0.001005", "100", 3)) // same price, later still

	r := mustSubmit(t, e, limit("b1", "cust", Buy, "0.001010", "250", 4))
	tr := trades(r.Events)
	if len(tr) != 3 {
		t.Fatalf("got %d trades, want 3", len(tr))
	}
	want := []struct{ maker, price, qty string }{
		{"a2", "0.001005", "100.000000"}, {"a3", "0.001005", "100.000000"}, {"a1", "0.001010", "50.000000"},
	}
	for i, w := range want {
		if tr[i].Seller.OrderID != w.maker || tr[i].Price.String() != w.price || tr[i].Quantity.String() != w.qty {
			t.Errorf("trade %d = %s @%s x%s, want %s @%s x%s", i, tr[i].Seller.OrderID, tr[i].Price, tr[i].Quantity, w.maker, w.price, w.qty)
		}
		if tr[i].AggressorSide != Buy || tr[i].Buyer.Liquidity != Taker || tr[i].Seller.Liquidity != Maker {
			t.Errorf("trade %d roles wrong: %+v", i, tr[i])
		}
	}
	if r.Order.State != Filled {
		t.Errorf("taker state = %s, want filled", r.Order.State)
	}
	_, asks := e.Depth("EAI-IDX", true, 0)
	if len(asks) != 1 || asks[0].Price != px("0.001010") || asks[0].Quantity != qty("50") {
		t.Errorf("remaining asks = %+v, want 50 @ 0.001010", asks)
	}
}

// TestLimitRestsRemainder checks a partially filled GTC limit order rests its remainder at its price.
func TestLimitRestsRemainder(t *testing.T) {
	e := New(Config{})
	mustSubmit(t, e, limit("a1", "mm", Sell, "0.001000", "40", 1))
	r := mustSubmit(t, e, limit("b1", "cust", Buy, "0.001002", "100", 2))
	if r.Order.State != PartiallyFilled || r.Order.Filled != qty("40") {
		t.Fatalf("taker = %s filled %s, want partially_filled 40", r.Order.State, r.Order.Filled)
	}
	bids, asks := e.Depth("EAI-IDX", true, 0)
	if len(asks) != 0 || len(bids) != 1 || bids[0].Price != px("0.001002") || bids[0].Quantity != qty("60") {
		t.Errorf("book = bids %+v asks %+v, want one bid 60 @ 0.001002", bids, asks)
	}
	open := e.OpenOrders("cust", true)
	if len(open) != 1 || open[0].OrderID != "b1" {
		t.Errorf("open orders = %+v", open)
	}
}

// TestMarketAndIOCNeverRest checks market and IOC remainders are cancelled, not rested.
func TestMarketAndIOCNeverRest(t *testing.T) {
	e := New(Config{})
	mustSubmit(t, e, limit("a1", "mm", Sell, "0.001000", "30", 1))

	m := mustSubmit(t, e, SubmitCmd{OrderID: "m1", TenantID: "cust", ProductID: "EAI-IDX", Side: Buy, Type: Market, Quantity: qty("50"), IsPaper: true, TS: at(2)})
	if m.Order.State != Cancelled || m.Order.Reason != ReasonUnfilled || m.Order.Filled != qty("30") {
		t.Errorf("market = %s/%s filled %s, want cancelled/unfilled_remainder 30", m.Order.State, m.Order.Reason, m.Order.Filled)
	}
	i := mustSubmit(t, e, SubmitCmd{OrderID: "i1", TenantID: "cust", ProductID: "EAI-IDX", Side: Buy, Type: IOC, Price: px("0.001000"), Quantity: qty("5"), IsPaper: true, TS: at(3)})
	if i.Order.State != Cancelled || i.Order.Filled != 0 {
		t.Errorf("ioc on empty side = %s filled %s, want cancelled 0", i.Order.State, i.Order.Filled)
	}
	if bids, asks := e.Depth("EAI-IDX", true, 0); len(bids)+len(asks) != 0 {
		t.Errorf("book not empty: bids %+v asks %+v", bids, asks)
	}
}

// TestFOKAllOrNothing checks FOK fills fully or leaves the book untouched.
func TestFOKAllOrNothing(t *testing.T) {
	e := New(Config{})
	mustSubmit(t, e, limit("a1", "mm1", Sell, "0.001000", "30", 1))
	mustSubmit(t, e, limit("a2", "mm2", Sell, "0.001001", "30", 2))

	fok := func(id, q, p string, n int) Result {
		return mustSubmit(t, e, SubmitCmd{OrderID: id, TenantID: "cust", ProductID: "EAI-IDX", Side: Buy, Type: FOK, Price: px(p), Quantity: qty(q), IsPaper: true, TS: at(n)})
	}
	r := fok("f1", "61", "0.001001", 3) // 60 available: must not fill at all
	if r.Order.State != Cancelled || r.Order.Reason != ReasonFOKUnfilled || len(trades(r.Events)) != 0 {
		t.Fatalf("oversized FOK = %s/%s with %d trades", r.Order.State, r.Order.Reason, len(trades(r.Events)))
	}
	r = fok("f2", "40", "0.001000", 4) // 40 needs the 0.001001 level, which the limit excludes
	if r.Order.State != Cancelled || len(trades(r.Events)) != 0 {
		t.Fatalf("price-limited FOK = %s with %d trades", r.Order.State, len(trades(r.Events)))
	}
	r = fok("f3", "60", "0.001001", 5)
	if r.Order.State != Filled || len(trades(r.Events)) != 2 {
		t.Fatalf("exact FOK = %s with %d trades, want filled with 2", r.Order.State, len(trades(r.Events)))
	}
}

// TestSelfTradePrevention checks a tenant never trades with itself: the incoming remainder is
// cancelled and the resting order is untouched.
func TestSelfTradePrevention(t *testing.T) {
	e := New(Config{})
	mustSubmit(t, e, limit("a1", "other", Sell, "0.001000", "10", 1))
	mustSubmit(t, e, limit("a2", "cust", Sell, "0.001001", "10", 2))
	r := mustSubmit(t, e, limit("b1", "cust", Buy, "0.001005", "50", 3))
	if tr := trades(r.Events); len(tr) != 1 || tr[0].Seller.TenantID != "other" {
		t.Fatalf("trades = %+v, want one against 'other'", tr)
	}
	if r.Order.State != Cancelled || r.Order.Reason != ReasonSelfTrade || r.Order.Filled != qty("10") {
		t.Errorf("taker = %s/%s filled %s, want cancelled/self_trade 10", r.Order.State, r.Order.Reason, r.Order.Filled)
	}
	if o, _ := e.Order("a2", "cust"); o.State != Accepted || o.Filled != 0 {
		t.Errorf("resting own order changed: %+v", o)
	}
}

// TestPaperRealIsolation checks paper and real-money orders never match, even at crossing prices.
func TestPaperRealIsolation(t *testing.T) {
	e := New(Config{})
	real := limit("a1", "mm", Sell, "0.001000", "10", 1)
	real.IsPaper = false
	mustSubmit(t, e, real)
	r := mustSubmit(t, e, limit("b1", "cust", Buy, "0.001005", "10", 2))
	if len(trades(r.Events)) != 0 || r.Order.State != Accepted {
		t.Fatalf("paper order traded against real book: %+v", r)
	}
	for _, ev := range r.Events {
		if ev.Order != nil && !ev.Order.IsPaper {
			t.Errorf("paper order emitted a real-money event: %+v", ev.Order)
		}
	}
}

// TestInternalPaperForbidden checks the insider-risk rule: internal accounts cannot enter paper books.
func TestInternalPaperForbidden(t *testing.T) {
	e := New(Config{})
	c := limit("mm1", "mm", Sell, "0.001000", "10", 1)
	c.IsInternal = true
	if _, err := e.Submit(c); !errors.Is(err, ErrInternalPaper) {
		t.Fatalf("internal paper order err = %v, want ErrInternalPaper", err)
	}
	c.IsPaper = false
	if _, err := e.Submit(c); err != nil {
		t.Fatalf("internal real-book order rejected: %v", err)
	}
}

// TestPaperLiquidityRule checks the paper liquidity account: it trades with customers on paper books
// (that is its purpose), but never on a real book and never as an internal account — so the
// insider-risk rule and paper/real isolation both still hold.
func TestPaperLiquidityRule(t *testing.T) {
	e := New(Config{})
	ask := limit("lp1", "liq", Sell, "0.001005", "10", 1)
	ask.IsLiquidity = true
	mustSubmit(t, e, ask)
	r := mustSubmit(t, e, limit("c1", "cust", Buy, "0.001005", "4", 2))
	if tr := trades(r.Events); len(tr) != 1 || tr[0].Seller.TenantID != "liq" || tr[0].Seller.IsInternal {
		t.Fatalf("customer did not trade with the paper liquidity account: %+v", tr)
	}
	if o, _ := e.Order("lp1", "liq"); !o.IsLiquidity {
		t.Fatal("liquidity flag lost on the order")
	}

	real := limit("lp2", "liq", Sell, "0.001005", "10", 3)
	real.IsLiquidity, real.IsPaper = true, false
	if _, err := e.Submit(real); !errors.Is(err, ErrLiquidityReal) {
		t.Fatalf("liquidity order on a real book: err = %v, want ErrLiquidityReal", err)
	}
	internal := limit("lp3", "liq", Sell, "0.001005", "10", 4)
	internal.IsLiquidity, internal.IsInternal = true, true
	if _, err := e.Submit(internal); err == nil {
		t.Fatal("an internal account was accepted as paper liquidity")
	}
	if len(e.Journal()) != 2 {
		t.Fatalf("refused liquidity orders reached the journal: %d commands", len(e.Journal()))
	}
}

// TestLiquidityFlagKeepsOldJournalEncoding checks omitempty: a customer submit encodes exactly as it
// did before the flag existed, so earlier journals and their hash chains are untouched.
func TestLiquidityFlagKeepsOldJournalEncoding(t *testing.T) {
	b, err := json.Marshal(Command{Submit: &SubmitCmd{OrderID: "o", TenantID: "t", IsPaper: true}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "IsLiquidity") {
		t.Fatalf("customer submit encodes the liquidity flag: %s", b)
	}
}

// TestValidation checks every contract-level validation error and that none of them changes state.
func TestValidation(t *testing.T) {
	base := limit("x", "t", Buy, "0.001000", "1", 1)
	cases := []struct {
		name string
		mod  func(c *SubmitCmd)
		want error
	}{
		{"unknown product", func(c *SubmitCmd) { c.ProductID = "NOPE" }, ErrUnknownProduct},
		{"forward not tradeable", func(c *SubmitCmd) { c.ProductID = "H100-FWD-30D"; c.Price = px("3.04") }, ErrNotTradeable},
		{"bad side", func(c *SubmitCmd) { c.Side = "hold" }, ErrBadSide},
		{"bad type", func(c *SubmitCmd) { c.Type = "stop" }, ErrBadType},
		{"ioc type with day", func(c *SubmitCmd) { c.Type = IOC; c.TIF = Day }, ErrBadTIF},
		{"market with fok tif", func(c *SubmitCmd) { c.Type = Market; c.Price = 0; c.TIF = TIFFOK }, ErrBadTIF},
		{"limit with fok tif", func(c *SubmitCmd) { c.TIF = TIFFOK }, ErrBadTIF},
		{"zero quantity", func(c *SubmitCmd) { c.Quantity = 0 }, ErrBadQuantity},
		{"zero price", func(c *SubmitCmd) { c.Price = 0 }, ErrBadPrice},
		{"off tick", func(c *SubmitCmd) { c.ProductID = "H100-SPOT"; c.Price = px("2.995") }, ErrBadPrice},
		{"market with price", func(c *SubmitCmd) { c.Type = Market }, ErrPriceOnMarket},
		{"no order id", func(c *SubmitCmd) { c.OrderID = "" }, ErrMissingID},
		{"no timestamp", func(c *SubmitCmd) { c.TS = time.Time{} }, ErrMissingTimestamp},
		{"notional overflow", func(c *SubmitCmd) { c.Price = px("9000000000000"); c.Quantity = px("9000000000000") }, ErrNotional},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := New(Config{})
			c := base
			tc.mod(&c)
			if _, err := e.Submit(c); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if len(e.Journal()) != 0 {
				t.Errorf("invalid command was journaled")
			}
		})
	}
}

// TestContractDefaultTIF checks the contract's default time_in_force (gtc), sent by a generated client
// on a market/ioc/fok order, is read as unspecified rather than rejected.
func TestContractDefaultTIF(t *testing.T) {
	e := New(Config{})
	r := mustSubmit(t, e, SubmitCmd{OrderID: "m1", TenantID: "c", ProductID: "EAI-IDX", Side: Buy, Type: Market, TIF: GTC, Quantity: qty("1"), IsPaper: true, TS: at(1)})
	if r.Order.TIF != TIFIOC {
		t.Errorf("market with gtc resolved to %q, want ioc", r.Order.TIF)
	}
	r = mustSubmit(t, e, SubmitCmd{OrderID: "f1", TenantID: "c", ProductID: "EAI-IDX", Side: Buy, Type: FOK, TIF: GTC, Price: px("0.001"), Quantity: qty("1"), IsPaper: true, TS: at(2)})
	if r.Order.TIF != TIFFOK {
		t.Errorf("fok with gtc resolved to %q, want fok", r.Order.TIF)
	}
}

// TestIdempotentSubmit checks a re-sent order id is a no-op for its tenant and refused for another.
func TestIdempotentSubmit(t *testing.T) {
	e := New(Config{})
	mustSubmit(t, e, limit("b1", "cust", Buy, "0.001000", "10", 1))
	r := mustSubmit(t, e, limit("b1", "cust", Buy, "0.001000", "10", 2))
	if !r.Duplicate || len(r.Events) != 0 {
		t.Fatalf("duplicate submit emitted events: %+v", r)
	}
	if bids, _ := e.Depth("EAI-IDX", true, 0); bids[0].Quantity != qty("10") {
		t.Errorf("duplicate submit doubled the book: %+v", bids)
	}
	if _, err := e.Submit(limit("b1", "intruder", Buy, "0.001000", "10", 3)); !errors.Is(err, ErrOrderIDTaken) {
		t.Errorf("cross-tenant id reuse err = %v, want ErrOrderIDTaken", err)
	}
}

// TestCancel checks cancel removes a resting order, is tenant-scoped, and refuses closed orders.
func TestCancel(t *testing.T) {
	e := New(Config{})
	mustSubmit(t, e, limit("b1", "cust", Buy, "0.001000", "10", 1))
	if _, err := e.Cancel(CancelCmd{OrderID: "b1", TenantID: "intruder", TS: at(2)}); !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("cross-tenant cancel err = %v, want not found", err)
	}
	r, err := e.Cancel(CancelCmd{OrderID: "b1", TenantID: "cust", TS: at(3)})
	if err != nil || r.Order.State != Cancelled || r.Order.Reason != ReasonUserCancel {
		t.Fatalf("cancel = %+v, %v", r.Order, err)
	}
	if bids, _ := e.Depth("EAI-IDX", true, 0); len(bids) != 0 {
		t.Errorf("cancelled order still on book: %+v", bids)
	}
	if _, err := e.Cancel(CancelCmd{OrderID: "b1", TenantID: "cust", TS: at(4)}); !errors.Is(err, ErrOrderNotOpen) {
		t.Errorf("second cancel err = %v, want ErrOrderNotOpen", err)
	}
}

// TestExpireDay checks only TIF=day orders are expired.
func TestExpireDay(t *testing.T) {
	e := New(Config{})
	d := limit("d1", "cust", Buy, "0.001000", "10", 1)
	d.TIF = Day
	mustSubmit(t, e, d)
	mustSubmit(t, e, limit("g1", "cust", Buy, "0.000999", "10", 2))
	evs, err := e.ExpireDay(ExpireDayCmd{TS: at(3)})
	if err != nil || len(evs) != 1 || evs[0].Order.OrderID != "d1" || evs[0].Order.Reason != ReasonDayExpired {
		t.Fatalf("expire events = %+v, %v", evs, err)
	}
	if open := e.OpenOrders("cust", true); len(open) != 1 || open[0].OrderID != "g1" {
		t.Errorf("open after expiry = %+v, want only g1", open)
	}
}

// TestRiskRejectIsJournaled checks a risk rejection emits a rejected event and that replay reuses the
// recorded decision instead of asking the risk hook again.
func TestRiskRejectIsJournaled(t *testing.T) {
	e := New(Config{Risk: func(o Order, _ Hold) string {
		if o.Quantity > qty("100") {
			return "position_limit"
		}
		return ""
	}})
	r := mustSubmit(t, e, limit("b1", "cust", Buy, "0.001000", "500", 1))
	if r.Order.State != Rejected || r.Order.Reason != "position_limit" || len(r.Events) != 1 {
		t.Fatalf("risk reject = %+v", r)
	}
	called := false
	_, evs, err := Replay(Config{Risk: func(Order, Hold) string { called = true; return "" }}, e.Journal())
	if err != nil || called {
		t.Fatalf("replay consulted the risk hook (called=%v, err=%v)", called, err)
	}
	if evs[0].Order.State != Rejected {
		t.Errorf("replayed state = %s, want rejected", evs[0].Order.State)
	}
}

// TestFees checks fees are fixed-point, charged on notional, and split by liquidity role.
func TestFees(t *testing.T) {
	e := New(Config{})
	mustSubmit(t, e, SubmitCmd{OrderID: "a1", TenantID: "mm", ProductID: "H100-SPOT", Side: Sell, Type: Limit, Price: px("2.99"), Quantity: qty("10"), IsPaper: true, TS: at(1)})
	r := mustSubmit(t, e, SubmitCmd{OrderID: "b1", TenantID: "cust", ProductID: "H100-SPOT", Side: Buy, Type: Market, Quantity: qty("10"), IsPaper: true, TS: at(2)})
	tr := trades(r.Events)[0]
	// notional 29.90; taker 1% = 0.299, maker 0.5% = 0.1495
	if tr.Buyer.Fee.String() != "0.299000" || tr.Seller.Fee.String() != "0.149500" {
		t.Errorf("fees buyer %s seller %s, want 0.299000 / 0.149500", tr.Buyer.Fee, tr.Seller.Fee)
	}
	if tr.CreditType != "gpu_h100" {
		t.Errorf("credit type %s, want gpu_h100", tr.CreditType)
	}
}

// TestTradeIDsAreUUIDs checks derived trade ids have the UUID shape the contract requires.
func TestTradeIDsAreUUIDs(t *testing.T) {
	id := uuidFrom("trade", "epoch-a", bookKey{product: "EAI-IDX", paper: true}, 1)
	if len(id) != 36 || id[14] != '8' || (id[19] != '8' && id[19] != '9' && id[19] != 'a' && id[19] != 'b') {
		t.Errorf("id %q is not a v8 RFC 9562 UUID", id)
	}
	if uuidFrom("trade", "epoch-b", bookKey{product: "EAI-IDX", paper: true}, 1) == id {
		t.Error("two journals (epochs) derived the same trade id for the same book and sequence")
	}
}

// TestHolds pins the hold for each order shape: a sell holds its credits; a limit buy holds
// floor(limit × qty) + taker fee; a market buy holds the exact sweep cost + taker fee, and nothing when
// the book is empty.
func TestHolds(t *testing.T) {
	var got []Hold
	e := New(Config{Risk: func(_ Order, h Hold) string { got = append(got, h); return "" }})
	h100 := func(id, tenant string, side Side, typ OrderType, price, q string, n int) SubmitCmd {
		return SubmitCmd{OrderID: id, TenantID: tenant, ProductID: "H100-SPOT", Side: side, Type: typ, Price: px(price), Quantity: qty(q), IsPaper: true, TS: at(n)}
	}
	mustSubmit(t, e, h100("m0", "c", Buy, Market, "0", "5", 1))       // empty book
	mustSubmit(t, e, h100("s1", "mm1", Sell, Limit, "2.99", "4", 2))  // sell
	mustSubmit(t, e, h100("s2", "mm2", Sell, Limit, "3.00", "10", 3)) // sell
	mustSubmit(t, e, h100("b1", "c", Buy, Limit, "3.00", "10", 4))    // limit buy
	mustSubmit(t, e, h100("s3", "mm3", Sell, Limit, "3.10", "10", 5)) // refill asks
	mustSubmit(t, e, h100("m1", "c", Buy, Market, "0", "12", 6))      // sweeps 4 @ 3.00 then 8 @ 3.10
	want := []struct{ kind, amt string }{
		{"cash", "0.000000"},
		{"credit", "4.000000"},
		{"credit", "10.000000"},
		{"cash", "30.300000"}, // 30.00 + 1% 0.30
		{"credit", "10.000000"},
		{"cash", "37.168000"}, // b1 left 4 @ 3.00: (4 × 3.00 = 12.00) + (8 × 3.10 = 24.80) = 36.80, + 1% 0.368
	}
	if len(got) != len(want) {
		t.Fatalf("got %d holds, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Kind != w.kind || got[i].Amount.String() != w.amt {
			t.Errorf("hold %d = %s %s, want %s %s", i, got[i].Kind, got[i].Amount, w.kind, w.amt)
		}
	}
}
