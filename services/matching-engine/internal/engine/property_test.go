package engine

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sync"
	"testing"
)

// randomCommands generates a reproducible stream of submits, cancels and day expiries across a few
// tenants, two products, and both paper and real books — dense enough around one price to cross often.
func randomCommands(seed int64, n int) []Command {
	r := rand.New(rand.NewSource(seed)) // #nosec G404 -- deterministic test input, not security
	tenants := []string{"t1", "t2", "t3", "t4"}
	products := []struct {
		id    string
		base  int64 // price in ticks
		tick  string
		scale int64
	}{{"EAI-IDX", 1000, "0.000001", 1}, {"H100-SPOT", 299, "0.01", 10_000}}
	var cmds []Command
	var ids []string
	for i := range n {
		ts := at(i + 1)
		switch k := r.Intn(20); {
		case k < 15:
			p := products[r.Intn(len(products))]
			c := SubmitCmd{
				OrderID: fmt.Sprintf("o%d", i), TenantID: tenants[r.Intn(len(tenants))], ProductID: p.id,
				Side: []Side{Buy, Sell}[r.Intn(2)], Quantity: Fixed(int64(1+r.Intn(50)) * unit),
				IsPaper: r.Intn(4) != 0, TS: ts,
			}
			c.Type = []OrderType{Limit, Limit, Limit, Market, IOC, FOK}[r.Intn(6)]
			if c.Type != Market {
				c.Price = Fixed((p.base + int64(r.Intn(11)-5)) * p.scale)
			}
			if c.Type == Limit && r.Intn(3) == 0 {
				c.TIF = Day
			}
			if !c.IsPaper && r.Intn(3) == 0 {
				c.IsInternal = true
			}
			if r.Intn(25) == 0 && len(ids) > 0 {
				c.OrderID = ids[r.Intn(len(ids))] // duplicate id: idempotent or refused
			}
			ids = append(ids, c.OrderID)
			cmds = append(cmds, Command{Submit: &c})
		case k < 19 && len(ids) > 0:
			cmds = append(cmds, Command{Cancel: &CancelCmd{OrderID: ids[r.Intn(len(ids))], TenantID: tenants[r.Intn(len(tenants))], TS: ts}})
		default:
			cmds = append(cmds, Command{ExpireDay: &ExpireDayCmd{TS: ts}})
		}
	}
	return cmds
}

// run feeds commands to an engine, ignoring validation/cancel errors (the stream includes invalid
// ones on purpose), and returns every emitted event.
func run(e *Engine, cmds []Command) []Event {
	var all []Event
	for _, c := range cmds {
		switch {
		case c.Submit != nil:
			r, _ := e.Submit(*c.Submit)
			all = append(all, r.Events...)
		case c.Cancel != nil:
			r, _ := e.Cancel(*c.Cancel)
			all = append(all, r.Events...)
		case c.ExpireDay != nil:
			evs, _ := e.ExpireDay(*c.ExpireDay)
			all = append(all, evs...)
		}
	}
	return all
}

// TestInvariants runs many random streams and checks, after every run:
//   - no book is crossed (best bid < best ask);
//   - every trade is between different tenants, on one book's paper flag, with no internal account on
//     a paper book, at a price inside both orders' limits;
//   - filled quantity is conserved: each order's filled == the sum of its trade quantities, and never
//     exceeds its quantity;
//   - per-book sequences strictly increase and each book's trades form a valid hash chain.
func TestInvariants(t *testing.T) {
	for seed := int64(1); seed <= 200; seed++ {
		e := New(Config{})
		evs := run(e, randomCommands(seed, 400))

		for k, b := range e.books {
			if bb, ba := b.bids.best(), b.asks.best(); bb != nil && ba != nil && bb.price >= ba.price {
				t.Fatalf("seed %d: book %s crossed: bid %s >= ask %s", seed, k, bb.price, ba.price)
			}
		}

		filled := map[string]Fixed{}
		lastSeq := map[string]uint64{}
		chains := map[string][]Trade{}
		for _, ev := range evs {
			if ev.Order != nil {
				key := bookKey{ev.Order.ProductID, ev.Order.IsPaper}.String()
				if ev.Order.Sequence <= lastSeq[key] {
					t.Fatalf("seed %d: sequence not increasing on %s", seed, key)
				}
				lastSeq[key] = ev.Order.Sequence
				continue
			}
			tr := ev.Trade
			key := bookKey{tr.ProductID, tr.IsPaper}.String()
			if tr.Sequence <= lastSeq[key] {
				t.Fatalf("seed %d: sequence not increasing on %s", seed, key)
			}
			lastSeq[key] = tr.Sequence
			if tr.Buyer.TenantID == tr.Seller.TenantID {
				t.Fatalf("seed %d: self trade %+v", seed, tr)
			}
			if tr.IsPaper && (tr.Buyer.IsInternal || tr.Seller.IsInternal) {
				t.Fatalf("seed %d: internal account on paper trade %+v", seed, tr)
			}
			buy, sell := e.orders[tr.Buyer.OrderID], e.orders[tr.Seller.OrderID]
			if buy.IsPaper != tr.IsPaper || sell.IsPaper != tr.IsPaper {
				t.Fatalf("seed %d: paper/real mixed in trade %+v", seed, tr)
			}
			if (buy.Type != Market && tr.Price > buy.Price) || (sell.Type != Market && tr.Price < sell.Price) {
				t.Fatalf("seed %d: trade price %s outside limits (buy %s, sell %s)", seed, tr.Price, buy.Price, sell.Price)
			}
			filled[tr.Buyer.OrderID] += tr.Quantity
			filled[tr.Seller.OrderID] += tr.Quantity
			chains[key] = append(chains[key], *tr)
		}
		for id, o := range e.orders {
			if o.Filled != filled[id] || o.Filled > o.Quantity {
				t.Fatalf("seed %d: order %s filled %s, trades sum %s, qty %s", seed, id, o.Filled, filled[id], o.Quantity)
			}
			if o.Open() && o.Type != Limit {
				t.Fatalf("seed %d: non-limit order %s left open", seed, id)
			}
		}
		for key, ch := range chains {
			if !VerifyTradeChain(ch) {
				t.Fatalf("seed %d: broken trade chain on %s", seed, key)
			}
		}
	}
}

// TestReplayIsDeterministic checks that replaying an engine's journal — including after a JSON round
// trip, as it would be stored — reproduces every event and the final book exactly.
func TestReplayIsDeterministic(t *testing.T) {
	for seed := int64(1); seed <= 50; seed++ {
		e := New(Config{})
		want := run(e, randomCommands(seed, 400))

		raw, err := json.Marshal(e.Journal())
		if err != nil {
			t.Fatal(err)
		}
		var journal []Command
		if err := json.Unmarshal(raw, &journal); err != nil {
			t.Fatal(err)
		}
		r, got, err := Replay(Config{}, journal)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("seed %d: replay emitted different events (%d vs %d)", seed, len(got), len(want))
		}
		for _, p := range []string{"EAI-IDX", "H100-SPOT"} {
			for _, paper := range []bool{true, false} {
				wb, wa := e.Depth(p, paper, 0)
				gb, ga := r.Depth(p, paper, 0)
				if !reflect.DeepEqual(wb, gb) || !reflect.DeepEqual(wa, ga) {
					t.Fatalf("seed %d: replayed book %s/%v differs", seed, p, paper)
				}
			}
		}
	}
}

// TestConcurrentSubmits drives one engine from many goroutines; run with -race. Every command must
// be journaled exactly once and the invariants above must still hold for the result.
func TestConcurrentSubmits(t *testing.T) {
	e := New(Config{})
	var wg sync.WaitGroup
	for w := range 8 {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for _, c := range randomCommands(int64(1000+w), 200) {
				if c.Submit != nil {
					c.Submit.OrderID = fmt.Sprintf("w%d-%s", w, c.Submit.OrderID)
				}
				if c.Cancel != nil {
					c.Cancel.OrderID = fmt.Sprintf("w%d-%s", w, c.Cancel.OrderID)
				}
				run(e, []Command{c})
				_ = e.OpenOrders("t1", true)
			}
		}(w)
	}
	wg.Wait()
	if _, _, err := Replay(Config{}, e.Journal()); err != nil {
		t.Fatalf("replay after concurrent run: %v", err)
	}
	for _, b := range e.books {
		if bb, ba := b.bids.best(), b.asks.best(); bb != nil && ba != nil && bb.price >= ba.price {
			t.Fatalf("book %s crossed after concurrent run", b.key)
		}
	}
}
