package venue

import (
	"testing"
	"time"

	"github.com/trade1/matching-engine/internal/engine"
)

var t0 = time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

// fx parses a fixed-point literal.
func fx(s string) engine.Fixed { return engine.MustFixed(s) }

// order builds a paper limit order.
func order(id, tenant string, side engine.Side, price, qty string, n int) engine.SubmitCmd {
	return engine.SubmitCmd{OrderID: id, TenantID: tenant, ProductID: "H100-SPOT", Side: side, Type: engine.Limit,
		TIF: engine.GTC, Price: fx(price), Quantity: fx(qty), IsPaper: true, TS: t0.Add(time.Duration(n) * time.Second)}
}

// submit places c and fails the test on error.
func submit(t *testing.T, v *Venue, c engine.SubmitCmd) engine.Result {
	t.Helper()
	r, err := v.Submit(c)
	if err != nil {
		t.Fatalf("submit %s: %v", c.OrderID, err)
	}
	return r
}

// TestViewTracksOrdersFillsAndPositions drives a customer through buy, partial sell and a flip, and
// checks orders, fills, the average entry and realized profit against hand-computed values.
func TestViewTracksOrdersFillsAndPositions(t *testing.T) {
	v := New(engine.New(engine.Config{}), nil, nil, "liq")
	liq := order("a1", "liq", engine.Sell, "3.00", "4", 1)
	liq.IsLiquidity = true
	submit(t, v, liq)
	submit(t, v, order("c1", "alice", engine.Buy, "3.00", "4", 2)) // buys 4 @ 3.00

	bid := order("b1", "liq", engine.Buy, "3.50", "10", 3)
	bid.IsLiquidity = true
	submit(t, v, bid)
	submit(t, v, order("c2", "alice", engine.Sell, "3.50", "6", 4)) // sells 4 to close (+2.00), 2 more short

	var orders []OrderView
	var fills []Fill
	var pos []Position
	v.Read(func(w *View) {
		orders, fills, pos = w.Orders("alice", "", 10), w.Fills("alice", "", 10), w.Positions("alice")
	})

	if len(orders) != 2 || orders[0].OrderID != "c2" || ContractState(orders[0].State) != "filled" {
		t.Fatalf("orders newest first, c2 filled: %+v", orders)
	}
	if avg, ok := orders[1].AvgFillPrice(); !ok || avg != fx("3.00") {
		t.Fatalf("c1 average fill = %v, want 3.000000", avg)
	}
	if len(fills) != 2 || fills[0].Side != engine.Sell || fills[0].Liquidity != engine.Taker {
		t.Fatalf("fills: %+v", fills)
	}
	if len(pos) != 1 {
		t.Fatalf("positions: %+v", pos)
	}
	p := pos[0]
	// Net −2 (short) at 3.50. Realized: (3.50−3.00)×4 = 2.00, minus taker fees on both fills:
	// 1% of 12.00 = 0.12 and 1% of 21.00 = 0.21.
	if p.Net != -int64(fx("2")) || p.AvgEntry() != int64(fx("3.50")) {
		t.Fatalf("net/avg = %d/%d, want short 2 @ 3.50", p.Net, p.AvgEntry())
	}
	if want := int64(fx("2.00") - fx("0.12") - fx("0.21")); p.RealizedPnL != want {
		t.Fatalf("realized = %s, want %s", FormatSigned(p.RealizedPnL), FormatSigned(want))
	}
	if u := p.Unrealized(fx("3.25")); u != int64(fx("0.50")) {
		t.Fatalf("unrealized at 3.25 = %s, want 0.500000 (short gains as price falls)", FormatSigned(u))
	}
}

// TestLiquidityAccountHiddenFromTenantViews checks the liquidity account is a market participant (its
// trades are prints) but not a customer: it has no orders, fills or positions in the view.
func TestLiquidityAccountHiddenFromTenantViews(t *testing.T) {
	v := New(engine.New(engine.Config{}), nil, nil, "liq")
	liq := order("a1", "liq", engine.Sell, "3.00", "10", 1)
	liq.IsLiquidity = true
	submit(t, v, liq)
	submit(t, v, order("c1", "bob", engine.Buy, "3.00", "1", 2))
	v.Read(func(w *View) {
		if len(w.Orders("liq", "", 10))+len(w.Fills("liq", "", 10))+len(w.Positions("liq")) != 0 {
			t.Fatal("the liquidity account appears in tenant views")
		}
		if pr := w.Prints("H100-SPOT", 10); len(pr) != 1 || pr[0].Price != fx("3.00") {
			t.Fatalf("print missing: %+v", pr)
		}
	})
}

// TestRecoverRebuildsView checks the view built from a replayed journal equals the live one, so a
// restart shows customers exactly what they saw before.
func TestRecoverRebuildsView(t *testing.T) {
	live := New(engine.New(engine.Config{Epoch: "e1"}), nil, nil, "liq")
	liq := order("a1", "liq", engine.Sell, "3.00", "10", 1)
	liq.IsLiquidity = true
	submit(t, live, liq)
	submit(t, live, order("c1", "alice", engine.Buy, "3.00", "4", 2))
	submit(t, live, order("c2", "alice", engine.Buy, "2.00", "1", 3))
	if _, err := live.Cancel(engine.CancelCmd{OrderID: "c2", TenantID: "alice", TS: t0.Add(4 * time.Second)}); err != nil {
		t.Fatal(err)
	}

	cmds := live.eng.Journal()
	eng, evs, err := engine.Replay(engine.Config{Epoch: "e1"}, cmds)
	if err != nil {
		t.Fatal(err)
	}
	re := New(eng, cmds, evs, "liq")
	var a, b []OrderView
	var fa, fb []Fill
	live.Read(func(w *View) { a, fa = w.Orders("alice", "", 10), w.Fills("alice", "", 10) })
	re.Read(func(w *View) { b, fb = w.Orders("alice", "", 10), w.Fills("alice", "", 10) })
	if len(a) != len(b) || len(fa) != len(fb) {
		t.Fatalf("replayed view differs: %d/%d orders, %d/%d fills", len(a), len(b), len(fa), len(fb))
	}
	for i := range a {
		if a[i].OrderID != b[i].OrderID || a[i].State != b[i].State || a[i].TIF != b[i].TIF || a[i].Filled != b[i].Filled {
			t.Fatalf("order %d differs after replay: %+v vs %+v", i, a[i], b[i])
		}
	}
	if fa[0].FillID != fb[0].FillID {
		t.Fatal("fill ids are not stable across replay")
	}
}

// TestBarsAggregate checks minute bars roll up into longer intervals with correct OHLCV.
func TestBarsAggregate(t *testing.T) {
	v := New(engine.New(engine.Config{}), nil, nil, "liq")
	for i, px := range []string{"3.00", "3.10", "2.90", "3.05"} {
		liq := order("a"+px, "liq", engine.Sell, px, "1", 2*i)
		liq.IsLiquidity = true
		liq.TS = t0.Add(time.Duration(i) * time.Minute)
		submit(t, v, liq)
		c := order("c"+px, "bob", engine.Buy, px, "1", 2*i+1)
		c.TS = liq.TS.Add(time.Second)
		submit(t, v, c)
	}
	var bars map[int64]Bar
	v.Read(func(w *View) { bars = w.Bars("H100-SPOT", 300, t0.Unix(), t0.Unix()+3600) })
	b, ok := bars[t0.Unix()/300*300]
	if len(bars) != 1 || !ok {
		t.Fatalf("want one 5m bar, got %+v", bars)
	}
	if b.Open != fx("3.00") || b.High != fx("3.10") || b.Low != fx("2.90") || b.Close != fx("3.05") || b.Volume != fx("4") {
		t.Fatalf("5m bar = %+v", b)
	}
}
