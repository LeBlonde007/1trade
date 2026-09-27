package engine

import "sort"

// level is one price level: resting orders at a single price in arrival (time-priority) order.
type level struct {
	price  Fixed
	orders []*Order
}

// ladder is one side of a book: price levels sorted best-first (bids descending, asks ascending).
// Levels live in a sorted slice, so iteration order is deterministic — no map iteration anywhere in
// the matching path.
type ladder struct {
	bids   bool
	levels []*level
}

// better reports whether price a has priority over price b on this side.
func (l *ladder) better(a, b Fixed) bool {
	if l.bids {
		return a > b
	}
	return a < b
}

// best returns the best level, or nil when the side is empty.
func (l *ladder) best() *level {
	if len(l.levels) == 0 {
		return nil
	}
	return l.levels[0]
}

// find returns the index where a level at price is or would be inserted, and whether it exists.
func (l *ladder) find(price Fixed) (int, bool) {
	i := sort.Search(len(l.levels), func(i int) bool { return !l.better(l.levels[i].price, price) })
	return i, i < len(l.levels) && l.levels[i].price == price
}

// add appends o to the back of its price level (time priority), creating the level if needed.
func (l *ladder) add(o *Order) {
	i, ok := l.find(o.Price)
	if !ok {
		l.levels = append(l.levels, nil)
		copy(l.levels[i+1:], l.levels[i:])
		l.levels[i] = &level{price: o.Price}
	}
	l.levels[i].orders = append(l.levels[i].orders, o)
}

// remove takes o out of its level, dropping the level when it empties. It reports whether o was found.
func (l *ladder) remove(o *Order) bool {
	i, ok := l.find(o.Price)
	if !ok {
		return false
	}
	lv := l.levels[i]
	for j, r := range lv.orders {
		if r == o {
			lv.orders = append(lv.orders[:j], lv.orders[j+1:]...)
			if len(lv.orders) == 0 {
				l.levels = append(l.levels[:i], l.levels[i+1:]...)
			}
			return true
		}
	}
	return false
}

// popFront removes the first order of the best level (after it fully filled).
func (l *ladder) popFront() {
	lv := l.levels[0]
	lv.orders[0] = nil
	lv.orders = lv.orders[1:]
	if len(lv.orders) == 0 {
		l.levels = l.levels[1:]
	}
}

// DepthLevel is one aggregated price level of a book snapshot.
type DepthLevel struct {
	Price    Fixed
	Quantity Fixed
	Orders   int
}

// depth aggregates up to n levels (n <= 0 means all).
func (l *ladder) depth(n int) []DepthLevel {
	if n <= 0 || n > len(l.levels) {
		n = len(l.levels)
	}
	out := make([]DepthLevel, 0, n)
	for _, lv := range l.levels[:n] {
		d := DepthLevel{Price: lv.price, Orders: len(lv.orders)}
		for _, o := range lv.orders {
			d.Quantity += o.Remaining()
		}
		out = append(out, d)
	}
	return out
}

// bookKey identifies a book. Paper and real-money flow live in different books and can never meet.
type bookKey struct {
	product string
	paper   bool
}

// String renders the key for ids and hashing ("EAI-IDX/paper").
func (k bookKey) String() string {
	if k.paper {
		return k.product + "/paper"
	}
	return k.product + "/real"
}

// book is one product's order book for one of paper / real money.
type book struct {
	key        bookKey
	creditType string
	tick       Fixed
	bids, asks ladder
	seq        uint64 // per-book monotonic sequence over order events and trades (orders.state.v1)
	chainHead  string // ChainHash of the last trade on this book
}

// newBook returns an empty book.
func newBook(k bookKey, creditType string, tick Fixed) *book {
	return &book{key: k, creditType: creditType, tick: tick, bids: ladder{bids: true}, asks: ladder{}}
}

// own returns the ladder an order of side s rests on; opposite returns the one it matches against.
func (b *book) own(s Side) *ladder {
	if s == Buy {
		return &b.bids
	}
	return &b.asks
}

// opposite returns the ladder an order of side s matches against.
func (b *book) opposite(s Side) *ladder {
	if s == Buy {
		return &b.asks
	}
	return &b.bids
}

// crosses reports whether an incoming order o can trade at resting price p.
func crosses(o *Order, p Fixed) bool {
	if o.Type == Market {
		return true
	}
	if o.Side == Buy {
		return o.Price >= p
	}
	return o.Price <= p
}
