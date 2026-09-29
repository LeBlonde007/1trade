package detect

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

var t0 = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

// at returns t0 + d.
func at(d time.Duration) time.Time { return t0.Add(d) }

// event is one input: exactly one of order / trade.
type event struct {
	order *Order
	trade *Trade
}

// accept builds an accepted order event.
func accept(id, tenant, product, side, qty, price string, ts time.Time) event {
	p := price
	return event{order: &Order{OrderID: id, TenantID: tenant, ProductID: product, Side: side, OrderType: "limit",
		State: "accepted", Quantity: qty, FilledQuantity: "0", LimitPrice: &p, IsPaper: true, TS: ts}}
}

// cancel builds a cancelled order event with a reason.
func cancel(id, tenant, reason, filled string, ts time.Time) event {
	r := reason
	return event{order: &Order{OrderID: id, TenantID: tenant, State: "cancelled", Reason: &r, FilledQuantity: filled, IsPaper: true, TS: ts}}
}

// trade builds a trade event; aggressor is "buy" or "sell".
func trade(id, product, buyer, seller, aggressor, qty, price string, ts time.Time) event {
	return event{trade: &Trade{TradeID: id, ProductID: product, Price: price, Quantity: qty,
		Buyer: Party{TenantID: buyer, OrderID: "ob-" + id}, Seller: Party{TenantID: seller, OrderID: "os-" + id},
		AggressorSide: aggressor, IsPaper: true, ExecutedAt: ts}}
}

// run feeds events in order and returns every alert.
func run(d *Detector, evs []event) []Alert {
	var out []Alert
	for _, e := range evs {
		if e.order != nil {
			out = append(out, d.OnOrder(*e.order)...)
		} else {
			out = append(out, d.OnTrade(*e.trade)...)
		}
	}
	return out
}

// rules lists the rules of the alerts, for compact assertions.
func rules(as []Alert) []string {
	out := []string{}
	for _, a := range as {
		out = append(out, a.Rule+"/"+a.TenantID)
	}
	return out
}

// expect runs a scenario on a fresh default detector (optionally tuned) and checks the alert rules.
func expect(t *testing.T, name string, tune func(*Config), evs []event, want ...string) []Alert {
	t.Helper()
	cfg := DefaultConfig()
	if tune != nil {
		tune(&cfg)
	}
	got := run(New(cfg), evs)
	if w := append([]string{}, want...); !reflect.DeepEqual(rules(got), w) {
		t.Errorf("%s: alerts = %v, want %v", name, rules(got), w)
	}
	return got
}

// TestWashTrade covers the same-owner and round-trip patterns and their near misses.
func TestWashTrade(t *testing.T) {
	linked := func(c *Config) { c.BeneficialOwner = map[string]string{"A": "X", "B": "X"} }
	expect(t, "same owner", linked, []event{trade("t1", "EAI-IDX", "A", "B", "buy", "100", "1", at(0))}, "wash_trade/A")
	expect(t, "unlinked", nil, []event{trade("t1", "EAI-IDX", "A", "B", "buy", "100", "1", at(0))})

	rt := []event{
		trade("t1", "EAI-IDX", "A", "B", "buy", "100", "1.000", at(0)),
		trade("t2", "EAI-IDX", "B", "A", "buy", "100", "1.000", at(4*time.Minute)),
	}
	a := expect(t, "exact round trip", nil, rt, "wash_trade/B")
	if len(a) == 1 && (a[0].Evidence["pattern"] != "round_trip_exact" || len(a[0].TradeIDs) != 2) {
		t.Errorf("round-trip evidence = %v", a[0])
	}
	near := func(n int) []event {
		evs := []event{rt[0]}
		for i := range n {
			buyer, seller := "B", "A"
			if i%2 == 1 {
				buyer, seller = "A", "B"
			}
			qty, px := []string{"100.05", "100.02"}[i], []string{"1.0004", "1.0002"}[i] // near, never identical
			evs = append(evs, trade(fmt.Sprintf("n%d", i), "EAI-IDX", buyer, seller, "buy", qty, px, at(time.Duration(i+1)*time.Minute)))
		}
		return evs
	}
	expect(t, "one near round trip (chance)", nil, near(1))
	a = expect(t, "repeated near round trips", nil, near(2), "wash_trade/A")
	if len(a) == 1 && a[0].Evidence["pattern"] != "round_trip_repeated" {
		t.Errorf("repeated evidence = %v", a[0].Evidence)
	}
	expect(t, "round trip, size differs 1%", nil, []event{rt[0], trade("t2", "EAI-IDX", "B", "A", "buy", "101", "1.000", at(time.Minute))})
	expect(t, "round trip, price differs 0.1%", nil, []event{rt[0], trade("t2", "EAI-IDX", "B", "A", "buy", "100", "1.001", at(time.Minute))})
	expect(t, "round trip, outside window", nil, []event{rt[0], trade("t2", "EAI-IDX", "B", "A", "buy", "100", "1.000", at(6*time.Minute))})
}

// background returns n ordinary orders on a product from other tenants, to set the average size.
func background(n int, product string) []event {
	evs := make([]event, 0, n)
	for i := range n {
		evs = append(evs, accept(fmt.Sprintf("bg%d", i), fmt.Sprintf("bg-t%d", i), product, "buy", "10", "0.99", at(time.Duration(i)*time.Second)))
	}
	return evs
}

// TestSpoofing: a 50× order pulled in seconds while its owner trades the other side.
func TestSpoofing(t *testing.T) {
	base := background(5, "H100-SPOT")
	spoof := func(cancelAfter time.Duration, reason string, withOpposite bool) []event {
		evs := append([]event{}, base...)
		evs = append(evs, accept("big", "S", "H100-SPOT", "sell", "500", "3.10", at(10*time.Second)))
		if withOpposite {
			evs = append(evs, trade("x1", "H100-SPOT", "S", "M", "buy", "20", "3.00", at(12*time.Second)))
		}
		return append(evs, cancel("big", "S", reason, "0", at(10*time.Second+cancelAfter)))
	}
	a := expect(t, "spoof", nil, spoof(5*time.Second, "user_cancel", true), "spoofing/S")
	if len(a) == 1 && a[0].Action != ActionRateLimit {
		t.Errorf("spoof action = %s", a[0].Action)
	}
	expect(t, "held a minute", nil, spoof(time.Minute, "user_cancel", true))
	expect(t, "no opposite trade", nil, spoof(5*time.Second, "user_cancel", false))
	expect(t, "engine auto-cancel is not a signal", nil, spoof(5*time.Second, "unfilled_remainder", true))
}

// TestLayering: four bids at three+ levels pulled within a minute while the tenant sells.
func TestLayering(t *testing.T) {
	layer := func(prices ...string) []event {
		var evs []event
		for i, p := range prices {
			evs = append(evs, accept(fmt.Sprintf("L%d", i), "L", "TEXT-SPOT", "buy", "50", p, at(time.Duration(i)*time.Second)))
		}
		evs = append(evs, trade("s1", "TEXT-SPOT", "M", "L", "sell", "40", "0.00121", at(10*time.Second)))
		for i := range prices {
			evs = append(evs, cancel(fmt.Sprintf("L%d", i), "L", "user_cancel", "0", at(20*time.Second+time.Duration(i)*time.Second)))
		}
		return evs
	}
	expect(t, "layering", nil, layer("0.001200", "0.001199", "0.001198", "0.001197"), "layering/L")
	expect(t, "only two levels", nil, layer("0.001200", "0.001200", "0.001199", "0.001199"))
	expect(t, "three orders", nil, layer("0.001200", "0.001199", "0.001198"))
}

// TestMarkingTheClose: 60% of the volume in the half hour before the 16:00 print.
func TestMarkingTheClose(t *testing.T) {
	window := func(start time.Time) []event {
		return []event{
			trade("c1", "EAI-IDX", "M", "P", "buy", "700", "0.001", start),
			trade("c2", "EAI-IDX", "Q", "R", "buy", "500", "0.001", start.Add(time.Minute)),
			trade("c3", "EAI-IDX", "M", "U", "buy", "500", "0.001", start.Add(2*time.Minute)),
		}
	}
	close := time.Date(2026, 9, 27, 15, 40, 0, 0, time.UTC)
	a := expect(t, "into the close", nil, window(close), "marking_the_close/M")
	if len(a) == 1 && (a[0].Action != ActionExclude || a[0].WindowEnd.Hour() != 16) {
		t.Errorf("close alert = %+v", a[0])
	}
	expect(t, "same pattern at 14:00", nil, window(time.Date(2026, 9, 27, 14, 0, 0, 0, time.UTC)))
}

// TestCrossProduct: pushing TEXT-SPOT up while long the AI index that references it.
func TestCrossProduct(t *testing.T) {
	scenario := func(indexSide string) []event {
		evs := []event{}
		if indexSide == "long" {
			evs = append(evs, trade("i1", "EAI-IDX", "C", "Z", "buy", "5000", "0.001", at(0)))
		} else {
			evs = append(evs, trade("i1", "EAI-IDX", "Z", "C", "sell", "5000", "0.001", at(0)))
		}
		for i, px := range []string{"1.000", "1.005", "1.010", "1.020"} {
			evs = append(evs, trade(fmt.Sprintf("p%d", i), "TEXT-SPOT", "C", fmt.Sprintf("mm%d", i), "buy", "100", px, at(time.Minute+time.Duration(i)*time.Second)))
		}
		return evs
	}
	a := expect(t, "push + long related", nil, scenario("long"), "cross_product/C")
	if len(a) == 1 && (a[0].RelatedProductID == nil || *a[0].RelatedProductID != "EAI-IDX") {
		t.Errorf("related product = %v", a[0].RelatedProductID)
	}
	expect(t, "push + short related", nil, scenario("short"))
}

// TestExcessiveCancellation: 25 orders for one trade trips the ratio — once per window.
func TestExcessiveCancellation(t *testing.T) {
	churn := func(trades int) []event {
		var evs []event
		for i := range trades {
			evs = append(evs, trade(fmt.Sprintf("k%d", i), "EAI-IDX", "K", "Z", "buy", "1", "0.001", at(time.Duration(i)*time.Second)))
		}
		for i := range 25 {
			id := fmt.Sprintf("o%d", i)
			evs = append(evs, accept(id, "K", "EAI-IDX", "buy", "1", "0.0009", at(10*time.Second+time.Duration(i)*time.Second)),
				cancel(id, "K", "user_cancel", "0", at(10*time.Second+time.Duration(i)*time.Second+500*time.Millisecond)))
		}
		return evs
	}
	expect(t, "churn", func(c *Config) { c.LayerMinOrders = 1000 }, churn(1), "excessive_cancellation/K")
	expect(t, "ratio 5", func(c *Config) { c.LayerMinOrders = 1000 }, churn(5))
}

// TestPaperQuoterExemption checks the paper liquidity account's quoting (a ladder cancelled and
// re-placed while it trades the other side) raises no layering or order-to-trade alert on paper — yet
// the same pattern from a customer still does, the same account still alerts on a real book, and a
// trade-based rule (position limit) still applies to it.
func TestPaperQuoterExemption(t *testing.T) {
	ladder := func(tenant string, paper bool) []event {
		var evs []event
		for round := range 6 {
			base := time.Duration(round) * 5 * time.Second
			for i, px := range []string{"0.001200", "0.001199", "0.001198", "0.001197"} {
				ev := accept(fmt.Sprintf("%s-%d-%d", tenant, round, i), tenant, "TEXT-SPOT", "buy", "50", px, at(base+time.Duration(i)*time.Millisecond))
				ev.order.IsPaper = paper
				evs = append(evs, ev)
			}
			tr := trade(fmt.Sprintf("t-%s-%d", tenant, round), "TEXT-SPOT", "M", tenant, "sell", "1", "0.00121", at(base+time.Second))
			tr.trade.IsPaper = paper
			evs = append(evs, tr)
			for i := range 4 {
				ev := cancel(fmt.Sprintf("%s-%d-%d", tenant, round, i), tenant, "user_cancel", "0", at(base+2*time.Second+time.Duration(i)*time.Millisecond))
				ev.order.IsPaper = paper
				evs = append(evs, ev)
			}
		}
		return evs
	}
	quoters := func(c *Config) { c.PaperQuoters = map[string]bool{"LIQ": true}; c.OTRMinOrders = 20 }
	if got := expect(t, "quoter on paper", quoters, ladder("LIQ", true)); len(got) != 0 {
		return
	}
	if got := run(New(func() Config { c := DefaultConfig(); quoters(&c); return c }()), ladder("CUST", true)); len(got) == 0 {
		t.Error("a customer's identical layering pattern raised no alert")
	}
	if got := run(New(func() Config { c := DefaultConfig(); quoters(&c); return c }()), ladder("LIQ", false)); len(got) == 0 {
		t.Error("the quoter was exempt on a real book")
	}
	limits := func(c *Config) { quoters(c); c.DefaultMaxPosition = 3 * unit }
	expect(t, "position limit still applies", limits, ladder("LIQ", true), "position_limit/M", "position_limit/LIQ")
}

// TestPositionLimit: net and gross limits.
func TestPositionLimit(t *testing.T) {
	limits := func(c *Config) { c.DefaultMaxPosition = 100 * unit; c.MaxGrossNotional = 1000 * unit }
	expect(t, "within", limits, []event{trade("q1", "H100-SPOT", "P", "Z", "buy", "90", "3", at(0))})
	expect(t, "net breach", limits, []event{trade("q1", "H100-SPOT", "P", "Z", "buy", "150", "3", at(0))},
		"position_limit/P", "position_limit/Z")
	a := expect(t, "gross breach", func(c *Config) { c.MaxGrossNotional = 400 * unit }, []event{
		trade("q1", "H100-SPOT", "P", "Z", "buy", "90", "3", at(0)),
		trade("q2", "H200-SPOT", "P", "Y", "buy", "50", "3.8", at(time.Second)),
	}, "position_limit/P")
	if len(a) == 1 && a[0].ProductID != "*" {
		t.Errorf("gross alert product = %s", a[0].ProductID)
	}
}

// TestPaperRealNeverMix: the two halves of a round trip on different ledgers are not a round trip.
func TestPaperRealNeverMix(t *testing.T) {
	out := trade("t1", "EAI-IDX", "A", "B", "buy", "100", "1", at(0))
	back := trade("t2", "EAI-IDX", "B", "A", "buy", "100", "1", at(time.Minute))
	back.trade.IsPaper = false
	expect(t, "split across ledgers", nil, []event{out, back})
}

// TestRedeliveryIsIgnored: an at-least-once redelivery of a trade changes nothing.
func TestRedeliveryIsIgnored(t *testing.T) {
	d := New(DefaultConfig())
	t1 := trade("t1", "EAI-IDX", "A", "B", "buy", "100", "1", at(0))
	if got := run(d, []event{t1, t1, t1}); len(got) != 0 {
		t.Fatalf("duplicates raised %v", rules(got))
	}
	if pos := d.parts[true].positions["A"]["EAI-IDX"]; pos != 100*unit {
		t.Errorf("redelivery moved the position: %s", pos)
	}
}

// scenario is every rule firing once, for replay and contract tests.
func scenario() []event {
	var evs []event
	evs = append(evs, trade("t1", "EAI-IDX", "A", "B", "buy", "100", "1.000", at(0)),
		trade("t2", "EAI-IDX", "B", "A", "buy", "100", "1.000", at(time.Minute)))
	evs = append(evs, background(5, "H100-SPOT")...)
	evs = append(evs, accept("big", "S", "H100-SPOT", "sell", "500", "3.10", at(10*time.Second)),
		trade("x1", "H100-SPOT", "S", "M", "buy", "20", "3.00", at(12*time.Second)),
		cancel("big", "S", "user_cancel", "0", at(15*time.Second)))
	return evs
}

// TestReplayIsDeterministicAndIdempotent: same stream → same alerts; the same detector never raises
// an alert twice.
func TestReplayIsDeterministicAndIdempotent(t *testing.T) {
	a := run(New(DefaultConfig()), scenario())
	b := run(New(DefaultConfig()), scenario())
	if len(a) < 2 || !reflect.DeepEqual(a, b) {
		t.Fatalf("replay differs or too few alerts: %v vs %v", rules(a), rules(b))
	}
	d := New(DefaultConfig())
	run(d, scenario())
	if again := run(d, scenario()); len(again) != 0 {
		t.Errorf("re-fed stream re-raised %v", rules(again))
	}
}

// TestNoFalsePositiveStorm runs a busy, benign paper market — 30 tenants quoting and trading around a
// slowly drifting price, with occasional ordinary cancels — through the default thresholds. Honest
// flow must raise nothing (KW05 acceptance: no false-positive storms).
func TestNoFalsePositiveStorm(t *testing.T) {
	for _, seed := range []int64{7, 11, 23} {
		t.Run(fmt.Sprint("seed", seed), func(t *testing.T) { benignMarket(t, seed) })
	}
}

// benignMarket is one seeded run of TestNoFalsePositiveStorm.
func benignMarket(t *testing.T, seed int64) {
	r := rand.New(rand.NewSource(seed)) // #nosec G404 -- deterministic test input
	d := New(DefaultConfig())
	var alerts []Alert
	products := []string{"EAI-IDX", "TEXT-SPOT", "H100-SPOT"}
	mid := map[string]float64{"EAI-IDX": 0.001, "TEXT-SPOT": 0.00121, "H100-SPOT": 2.99}
	ts := time.Date(2026, 9, 27, 9, 0, 0, 0, time.UTC)
	for i := range 20000 {
		ts = ts.Add(time.Duration(200+r.Intn(1800)) * time.Millisecond) // ~5.5h, through the close window
		prod := products[r.Intn(len(products))]
		mid[prod] *= 1 + (r.Float64()-0.5)*0.0004
		px := fmt.Sprintf("%.6f", mid[prod])
		qty := fmt.Sprintf("%.6f", 5+r.ExpFloat64()*20)
		a, b := fmt.Sprintf("T%d", r.Intn(30)), fmt.Sprintf("T%d", r.Intn(30))
		if a == b {
			continue
		}
		side := []string{"buy", "sell"}[r.Intn(2)]
		id := fmt.Sprintf("o%d", i)
		alerts = append(alerts, run(d, []event{accept(id, a, prod, side, qty, px, ts)})...)
		switch r.Intn(10) {
		case 0: // the occasional ordinary cancel, after a while
			alerts = append(alerts, run(d, []event{cancel(id, a, "user_cancel", "0", ts.Add(time.Duration(30+r.Intn(120))*time.Second))})...)
		default: // fills against someone
			buyer, seller := a, b
			if side == "sell" {
				buyer, seller = b, a
			}
			alerts = append(alerts, run(d, []event{trade("t"+id, prod, buyer, seller, side, qty, px, ts.Add(time.Millisecond))})...)
		}
	}
	if len(alerts) != 0 {
		counts := map[string]int{}
		for _, a := range alerts {
			counts[a.Rule]++
		}
		t.Fatalf("benign market raised %d alerts: %v (first: %+v)", len(alerts), counts, alerts[0].Evidence)
	}
}
