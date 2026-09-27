package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/trade1/matching-engine/internal/domain"
)

// ErrOrderIDTaken is returned when an order_id is already used by a different tenant.
var ErrOrderIDTaken = errors.New("engine: order_id already in use")

// Engine holds every book and matches orders. One mutex serializes all commands: matching is
// single-threaded (per product by construction, and here globally), which is what makes the output a
// pure function of the command sequence. Per-book locking is a later throughput optimisation that
// must not change any observable ordering within a book.
type Engine struct {
	mu      sync.Mutex
	cfg     Config
	books   map[bookKey]*book
	orders  map[string]*Order // every order ever accepted or risk-rejected, by id (idempotency + cancel)
	arrival uint64
	journal []Command
}

// New returns an empty engine. A zero FeeSchedule means DefaultFees.
func New(cfg Config) *Engine {
	if cfg.Fees == (FeeSchedule{}) {
		cfg.Fees = DefaultFees
	}
	return &Engine{cfg: cfg, books: map[bookKey]*book{}, orders: map[string]*Order{}}
}

// Submit validates and places an order, matching it immediately against the opposite side. A
// validation error changes nothing. Re-submitting a known order_id for the same tenant is idempotent:
// it returns the order's current state with Duplicate set and emits nothing.
func (e *Engine) Submit(c SubmitCmd) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.submit(c)
}

// Cancel cancels an open order owned by the tenant.
func (e *Engine) Cancel(c CancelCmd) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cancel(c)
}

// ExpireDay cancels every resting TIF=day order and returns the events, in book then price-time order.
func (e *Engine) ExpireDay(c ExpireDayCmd) ([]Event, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.expireDay(c)
}

// submit is Submit without the lock.
func (e *Engine) submit(c SubmitCmd) (Result, error) {
	if prev, ok := e.orders[c.OrderID]; ok && c.OrderID != "" {
		if prev.TenantID != c.TenantID {
			return Result{}, ErrOrderIDTaken
		}
		return Result{Order: *prev, Duplicate: true}, nil
	}
	b, o, err := e.validate(&c)
	if err != nil {
		return Result{}, err
	}
	if !c.RiskChecked {
		if e.cfg.Risk != nil {
			c.RiskReason = e.cfg.Risk(*o)
		}
		c.RiskChecked = true
	}
	e.journal = append(e.journal, Command{Submit: &c})

	e.arrival++
	o.arrival = e.arrival
	e.orders[o.OrderID] = o
	var evs []Event

	if c.RiskReason != "" {
		o.State, o.Reason = Rejected, c.RiskReason
		evs = append(evs, b.orderEvent(o, c.TS))
		return Result{Order: *o, Events: evs}, nil
	}
	evs = append(evs, b.orderEvent(o, c.TS))

	if o.Type == FOK && !b.fillable(o) {
		o.State, o.Reason = Cancelled, ReasonFOKUnfilled
		evs = append(evs, b.orderEvent(o, c.TS))
		return Result{Order: *o, Events: evs}, nil
	}

	evs = append(evs, e.match(b, o, c.TS)...)

	if o.Open() && o.Remaining() > 0 {
		if o.Type == Limit {
			b.own(o.Side).add(o) // rests; no transition, so no event
		} else {
			o.State, o.Reason = Cancelled, ReasonUnfilled
			evs = append(evs, b.orderEvent(o, c.TS))
		}
	}
	return Result{Order: *o, Events: evs}, nil
}

// validate checks a submit against the contract and the product catalog, defaults its TIF, and
// returns the target book (created on first use) and a new order. It changes no engine state except
// creating an empty book.
func (e *Engine) validate(c *SubmitCmd) (*book, *Order, error) {
	if c.OrderID == "" || c.TenantID == "" {
		return nil, nil, ErrMissingID
	}
	if c.TS.IsZero() {
		return nil, nil, ErrMissingTimestamp
	}
	p, ok := domain.Find(c.ProductID)
	if !ok {
		return nil, nil, ErrUnknownProduct
	}
	if !p.Tradeable {
		return nil, nil, ErrNotTradeable
	}
	if c.Side != Buy && c.Side != Sell {
		return nil, nil, ErrBadSide
	}
	tif, err := defaultTIF(c.Type, c.TIF)
	if err != nil {
		return nil, nil, err
	}
	c.TIF = tif
	if c.Quantity <= 0 {
		return nil, nil, ErrBadQuantity
	}
	tick, err := ParseFixed(p.TickSize)
	if err != nil || tick <= 0 {
		return nil, nil, fmt.Errorf("engine: product %s has no valid tick: %w", p.ID, ErrUnknownProduct)
	}
	if c.Type == Market {
		if c.Price != 0 {
			return nil, nil, ErrPriceOnMarket
		}
	} else {
		if c.Price <= 0 || c.Price%tick != 0 {
			return nil, nil, ErrBadPrice
		}
		if _, ok := Notional(c.Price, c.Quantity); !ok {
			return nil, nil, ErrNotional
		}
	}
	// Insider-risk rule (phase6_v2 §12.1): internal accounts never trade with customer paper
	// accounts. Books are already split paper/real, so this closes the only remaining path.
	if c.IsPaper && c.IsInternal {
		return nil, nil, ErrInternalPaper
	}

	k := bookKey{product: p.ID, paper: c.IsPaper}
	b := e.books[k]
	if b == nil {
		b = newBook(k, p.CreditType, tick)
		e.books[k] = b
	}
	o := &Order{
		OrderID: c.OrderID, TenantID: c.TenantID, SubAccountID: c.SubAccountID, ProductID: p.ID,
		Side: c.Side, Type: c.Type, TIF: c.TIF, Price: c.Price, Quantity: c.Quantity,
		IsPaper: c.IsPaper, IsInternal: c.IsInternal, State: Accepted, AcceptedAt: c.TS, UpdatedAt: c.TS,
	}
	return b, o, nil
}

// defaultTIF resolves the time in force for an order type. The contract carries both fields, so they
// must agree: ioc/fok types imply their TIF, market is immediate-or-cancel, and a limit order rests
// gtc or day. `gtc` is the contract's declared default, so a generated client may send it without the
// user choosing it; for market/ioc/fok it is read as "unspecified". A real conflict (ioc + day, market
// + fok, …) is still an error.
func defaultTIF(t OrderType, tif TIF) (TIF, error) {
	if t != Limit && tif == GTC {
		tif = tifNone
	}
	switch t {
	case Market:
		if tif == tifNone || tif == TIFIOC {
			return TIFIOC, nil
		}
	case IOC:
		if tif == tifNone || tif == TIFIOC {
			return TIFIOC, nil
		}
	case FOK:
		if tif == tifNone || tif == TIFFOK {
			return TIFFOK, nil
		}
	case Limit:
		if tif == tifNone {
			return GTC, nil
		}
		if tif == GTC || tif == Day {
			return tif, nil
		}
	default:
		return tifNone, ErrBadType
	}
	return tifNone, ErrBadTIF
}

// match trades incoming order o against the opposite side in price-time priority until it fills,
// stops crossing, or would trade with its own tenant. Every trade executes at the resting price.
//
// Self-trade prevention is cancel-newest: the incoming order's remainder is cancelled and the resting
// order is untouched, so a tenant can never print a trade with itself (wash-trade floor, KW05).
func (e *Engine) match(b *book, o *Order, ts time.Time) []Event {
	var evs []Event
	opp := b.opposite(o.Side)
	for o.Remaining() > 0 {
		lv := opp.best()
		if lv == nil || !crosses(o, lv.price) {
			break
		}
		r := lv.orders[0]
		if r.TenantID == o.TenantID {
			o.State, o.Reason, o.UpdatedAt = Cancelled, ReasonSelfTrade, ts
			evs = append(evs, b.orderEvent(o, ts))
			break
		}
		qty := min(o.Remaining(), r.Remaining())
		o.Filled += qty
		r.Filled += qty
		o.State, r.State = fillState(o), fillState(r)
		o.UpdatedAt, r.UpdatedAt = ts, ts

		evs = append(evs, Event{Trade: e.trade(b, o, r, qty, ts)})
		evs = append(evs, b.orderEvent(r, ts), b.orderEvent(o, ts))
		if r.State == Filled {
			opp.popFront()
		}
	}
	return evs
}

// fillState is the state an order is in after a fill.
func fillState(o *Order) State {
	if o.Remaining() == 0 {
		return Filled
	}
	return PartiallyFilled
}

// trade builds and chains one execution of qty between taker and resting maker, at the maker's price.
func (e *Engine) trade(b *book, taker, maker *Order, qty Fixed, ts time.Time) *Trade {
	// Notional cannot overflow: the maker's price × its full quantity was bounded at acceptance, and
	// qty never exceeds that quantity.
	notional, _ := Notional(maker.Price, qty)
	takerFee, _ := mulDiv(notional, Fixed(e.cfg.Fees.TakerPPM), unit)
	makerFee, _ := mulDiv(notional, Fixed(e.cfg.Fees.MakerPPM), unit)

	tk := Counterparty{TenantID: taker.TenantID, SubAccountID: taker.SubAccountID, OrderID: taker.OrderID, Fee: takerFee, Liquidity: Taker, IsInternal: taker.IsInternal}
	mk := Counterparty{TenantID: maker.TenantID, SubAccountID: maker.SubAccountID, OrderID: maker.OrderID, Fee: makerFee, Liquidity: Maker, IsInternal: maker.IsInternal}
	buyer, seller := tk, mk
	if taker.Side == Sell {
		buyer, seller = mk, tk
	}
	b.seq++
	t := &Trade{
		TradeID: uuidFrom("trade", b.key, b.seq), ProductID: b.key.product, CreditType: b.creditType,
		Price: maker.Price, Quantity: qty, Buyer: buyer, Seller: seller, AggressorSide: taker.Side,
		IsPaper: b.key.paper, Sequence: b.seq, PrevChainHash: b.chainHead, ExecutedAt: ts.UTC(),
	}
	t.ChainHash = tradeChainHash(t.PrevChainHash, t)
	b.chainHead = t.ChainHash
	return t
}

// fillable reports whether a FOK order can fill completely right now without hitting a self-trade.
func (b *book) fillable(o *Order) bool {
	var avail Fixed
	for _, lv := range b.opposite(o.Side).levels {
		if !crosses(o, lv.price) {
			return false
		}
		for _, r := range lv.orders {
			if r.TenantID == o.TenantID {
				return false // self-trade prevention would stop the fill here
			}
			avail += r.Remaining()
			if avail >= o.Quantity {
				return true
			}
		}
	}
	return false
}

// orderEvent records o's current state as the book's next sequenced transition.
func (b *book) orderEvent(o *Order, ts time.Time) Event {
	b.seq++
	return Event{Order: &OrderEvent{
		EventID: "oev_" + digest("order", b.key, b.seq)[:24], OrderID: o.OrderID, TenantID: o.TenantID,
		SubAccountID: o.SubAccountID, ProductID: o.ProductID, Side: o.Side, Type: o.Type, State: o.State,
		Reason: o.Reason, Quantity: o.Quantity, FilledQuantity: o.Filled, LimitPrice: o.Price,
		IsPaper: o.IsPaper, IsInternal: o.IsInternal, Sequence: b.seq, TS: ts.UTC(),
	}}
}

// cancel is Cancel without the lock.
func (e *Engine) cancel(c CancelCmd) (Result, error) {
	if c.TS.IsZero() {
		return Result{}, ErrMissingTimestamp
	}
	o, ok := e.orders[c.OrderID]
	if !ok || o.TenantID != c.TenantID {
		return Result{}, ErrOrderNotFound
	}
	if !o.Open() {
		return Result{Order: *o}, ErrOrderNotOpen
	}
	b := e.books[bookKey{product: o.ProductID, paper: o.IsPaper}]
	b.own(o.Side).remove(o)
	o.State, o.Reason, o.UpdatedAt = Cancelled, ReasonUserCancel, c.TS
	e.journal = append(e.journal, Command{Cancel: &c})
	return Result{Order: *o, Events: []Event{b.orderEvent(o, c.TS)}}, nil
}

// expireDay is ExpireDay without the lock.
func (e *Engine) expireDay(c ExpireDayCmd) ([]Event, error) {
	if c.TS.IsZero() {
		return nil, ErrMissingTimestamp
	}
	e.journal = append(e.journal, Command{ExpireDay: &c})
	var evs []Event
	for _, b := range e.sortedBooks() {
		for _, l := range []*ladder{&b.bids, &b.asks} {
			var day []*Order
			for _, lv := range l.levels {
				for _, o := range lv.orders {
					if o.TIF == Day {
						day = append(day, o)
					}
				}
			}
			for _, o := range day {
				l.remove(o)
				o.State, o.Reason, o.UpdatedAt = Cancelled, ReasonDayExpired, c.TS
				evs = append(evs, b.orderEvent(o, c.TS))
			}
		}
	}
	return evs, nil
}

// sortedBooks returns books in a fixed order (product, then real before paper), never map order.
func (e *Engine) sortedBooks() []*book {
	out := make([]*book, 0, len(e.books))
	for _, b := range e.books {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key.String() < out[j].key.String() })
	return out
}

// Order returns a tenant's order by id. Another tenant's order is reported as not found.
func (e *Engine) Order(orderID, tenantID string) (Order, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	o, ok := e.orders[orderID]
	if !ok || o.TenantID != tenantID {
		return Order{}, false
	}
	return *o, true
}

// OpenOrders returns a tenant's resting orders on paper or real books, oldest first.
func (e *Engine) OpenOrders(tenantID string, paper bool) []Order {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []Order
	for _, o := range e.orders {
		if o.TenantID == tenantID && o.IsPaper == paper && o.Open() {
			out = append(out, *o)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].arrival < out[j].arrival })
	return out
}

// Depth returns up to n aggregated levels per side of a book (n <= 0 means all), best first.
func (e *Engine) Depth(productID string, paper bool, n int) (bids, asks []DepthLevel) {
	e.mu.Lock()
	defer e.mu.Unlock()
	b := e.books[bookKey{product: productID, paper: paper}]
	if b == nil {
		return nil, nil
	}
	return b.bids.depth(n), b.asks.depth(n)
}

// Journal returns a copy of every command that changed state, in order — the log the engine replays
// from. Each Submit carries its recorded risk decision.
func (e *Engine) Journal() []Command {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Command, len(e.journal))
	for i, c := range e.journal {
		out[i] = cloneCommand(c)
	}
	return out
}

// cloneCommand deep-copies a command so callers cannot mutate the journal.
func cloneCommand(c Command) Command {
	switch {
	case c.Submit != nil:
		s := *c.Submit
		return Command{Submit: &s}
	case c.Cancel != nil:
		x := *c.Cancel
		return Command{Cancel: &x}
	case c.ExpireDay != nil:
		x := *c.ExpireDay
		return Command{ExpireDay: &x}
	}
	return c
}

// Replay rebuilds an engine from a journal and returns it with every event re-emitted. Recorded risk
// decisions are reused, so cfg.Risk is never consulted for journaled submits. Replaying a journal
// produced by an engine with the same fee schedule reproduces its events exactly.
func Replay(cfg Config, journal []Command) (*Engine, []Event, error) {
	e := New(cfg)
	var all []Event
	for i, c := range journal {
		var (
			evs []Event
			err error
		)
		switch {
		case c.Submit != nil:
			var r Result
			r, err = e.Submit(*c.Submit)
			evs = r.Events
		case c.Cancel != nil:
			var r Result
			r, err = e.Cancel(*c.Cancel)
			evs = r.Events
		case c.ExpireDay != nil:
			evs, err = e.ExpireDay(*c.ExpireDay)
		default:
			err = errors.New("empty command")
		}
		if err != nil {
			return nil, nil, fmt.Errorf("engine: replay command %d: %w", i, err)
		}
		all = append(all, evs...)
	}
	return e, all, nil
}

// digest returns the hex SHA-256 of a kind, book, and sequence — the seed for deterministic ids.
func digest(kind string, k bookKey, seq uint64) string {
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%s|%d", kind, k.String(), seq))
	return hex.EncodeToString(sum[:])
}

// uuidFrom derives a stable RFC 9562 version-8 UUID from (kind, book, seq). Trade ids must be UUIDs
// (trades.executed.v1) and must be identical on replay, so they are derived, not random.
func uuidFrom(kind string, k bookKey, seq uint64) string {
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%s|%d", kind, k.String(), seq))
	sum[6] = (sum[6] & 0x0f) | 0x80 // version 8
	sum[8] = (sum[8] & 0x3f) | 0x80 // RFC 9562 variant
	h := hex.EncodeToString(sum[:16])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}
