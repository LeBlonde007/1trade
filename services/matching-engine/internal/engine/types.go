// Package engine is the real matching engine core (KW03): per-book price-time priority matching for
// market, limit, IOC and FOK orders, with paper/real isolation, self-trade prevention, a hash-chained
// trade log, and deterministic replay from a command journal.
//
// The package is pure: it never reads the wall clock, the network, or randomness. Every timestamp
// arrives on a command, and every external decision (the pre-trade risk check) is recorded into the
// journal when first made, so replaying the journal reproduces the same events byte for byte. That is
// what "event-sourced and replayable" means here — see SPEC.md.
//
// Order entry over HTTP is still closed (503 EXCHANGE_PAUSED, F22). Building the engine does not open
// the venue; wiring it to the API is a separate, licence-gated cutover step.
package engine

import (
	"errors"
	"time"
)

// Side is the order side, matching openapi/trading.yaml.
type Side string

// Order sides.
const (
	Buy  Side = "buy"
	Sell Side = "sell"
)

// OrderType is the order type, matching openapi/trading.yaml OrderRequest.order_type.
type OrderType string

// Order types.
const (
	Market OrderType = "market"
	Limit  OrderType = "limit"
	IOC    OrderType = "ioc"
	FOK    OrderType = "fok"
)

// TIF is the time in force, matching openapi/trading.yaml OrderRequest.time_in_force.
type TIF string

// Times in force.
const (
	GTC     TIF = "gtc"
	Day     TIF = "day"
	TIFIOC  TIF = "ioc"
	TIFFOK  TIF = "fok"
	tifNone TIF = ""
)

// State is an order state, matching events/orders.state.v1.
type State string

// Order states.
const (
	Accepted        State = "accepted"
	PartiallyFilled State = "partially_filled"
	Filled          State = "filled"
	Cancelled       State = "cancelled"
	Rejected        State = "rejected"
)

// Reasons attached to cancelled / rejected transitions (orders.state.v1 `reason`).
const (
	ReasonUserCancel  = "user_cancel"
	ReasonSelfTrade   = "self_trade"
	ReasonUnfilled    = "unfilled_remainder"
	ReasonFOKUnfilled = "fok_unfilled"
	ReasonDayExpired  = "day_expired"
)

// Liquidity roles on a trade.
const (
	Maker = "maker"
	Taker = "taker"
)

// Validation errors. A command that fails validation changes nothing and emits nothing; the API
// layer maps these to 422.
var (
	ErrUnknownProduct   = errors.New("engine: unknown product")
	ErrNotTradeable     = errors.New("engine: product is not tradeable")
	ErrBadSide          = errors.New("engine: side must be buy or sell")
	ErrBadType          = errors.New("engine: order_type must be market, limit, ioc or fok")
	ErrBadTIF           = errors.New("engine: time_in_force does not match order_type")
	ErrBadQuantity      = errors.New("engine: quantity must be positive")
	ErrBadPrice         = errors.New("engine: limit_price is required, positive, and on the product tick")
	ErrPriceOnMarket    = errors.New("engine: market orders take no limit_price")
	ErrNotional         = errors.New("engine: price × quantity is too large")
	ErrMissingID        = errors.New("engine: order_id and tenant_id are required")
	ErrInternalPaper    = errors.New("engine: internal accounts may not trade on paper books")
	ErrOrderNotFound    = errors.New("engine: order not found")
	ErrOrderNotOpen     = errors.New("engine: order is not open")
	ErrMissingTimestamp = errors.New("engine: command timestamp is required")
	ErrJournal          = errors.New("engine: journal write failed; command not applied")
)

// Order is one order and its live state. Values are copies; the engine never hands out a pointer
// into its books.
type Order struct {
	OrderID      string
	TenantID     string
	SubAccountID string
	ProductID    string
	Side         Side
	Type         OrderType
	TIF          TIF
	Price        Fixed // zero for market orders
	Quantity     Fixed
	Filled       Fixed
	IsPaper      bool
	IsInternal   bool
	State        State
	Reason       string
	AcceptedAt   time.Time
	UpdatedAt    time.Time

	arrival uint64 // global arrival order: the "time" in price-time priority
}

// Remaining is the unfilled quantity.
func (o Order) Remaining() Fixed { return o.Quantity - o.Filled }

// Open reports whether the order can still trade.
func (o Order) Open() bool { return o.State == Accepted || o.State == PartiallyFilled }

// OrderEvent is one order lifecycle transition, shaped for events/orders.state.v1.
type OrderEvent struct {
	EventID        string
	OrderID        string
	TenantID       string
	SubAccountID   string
	ProductID      string
	Side           Side
	Type           OrderType
	State          State
	Reason         string
	Quantity       Fixed
	FilledQuantity Fixed
	LimitPrice     Fixed // zero for market
	IsPaper        bool
	IsInternal     bool
	Sequence       uint64
	TS             time.Time
}

// Counterparty is one side of a trade, shaped for trades.executed.v1.
type Counterparty struct {
	TenantID     string
	SubAccountID string
	OrderID      string
	Fee          Fixed
	Liquidity    string
	IsInternal   bool
}

// Trade is one execution, shaped for events/trades.executed.v1. Trades on a book form an append-only
// hash chain: ChainHash = SHA-256(PrevChainHash || canonical_json(trade)).
type Trade struct {
	TradeID       string
	ProductID     string
	CreditType    string
	Price         Fixed
	Quantity      Fixed
	Buyer         Counterparty
	Seller        Counterparty
	AggressorSide Side
	IsPaper       bool
	Sequence      uint64
	PrevChainHash string
	ChainHash     string
	ExecutedAt    time.Time
}

// Event is one output of the engine, in emission order: exactly one of Order or Trade is set.
type Event struct {
	Order *OrderEvent
	Trade *Trade
}

// Result is what a command produced: the affected order's final snapshot and every event emitted,
// in order. Duplicate is true when a submit re-used a known order_id and changed nothing.
type Result struct {
	Order     Order
	Events    []Event
	Duplicate bool
}

// SubmitCmd places an order. TS is the acceptance time, set by the caller (the engine has no clock).
type SubmitCmd struct {
	OrderID      string
	TenantID     string
	SubAccountID string
	ProductID    string
	Side         Side
	Type         OrderType
	TIF          TIF // optional; defaulted from Type
	Price        Fixed
	Quantity     Fixed
	IsPaper      bool
	IsInternal   bool
	TS           time.Time

	// RiskReason and RiskChecked are filled by the engine from Config.Risk the first time the command
	// runs and replayed verbatim afterwards, so replay never re-asks an external system for a decision.
	// Both are exported so the decision survives serialising the journal.
	RiskReason  string
	RiskChecked bool
}

// CancelCmd cancels an open order. TenantID must own the order; otherwise it is reported not found
// (no information about other tenants' orders leaks).
type CancelCmd struct {
	OrderID  string
	TenantID string
	TS       time.Time
}

// ExpireDayCmd cancels every resting TIF=day order, across all books. The caller issues it at the
// session boundary.
type ExpireDayCmd struct {
	TS time.Time
}

// Command is one journaled input: exactly one field is set.
type Command struct {
	Submit    *SubmitCmd
	Cancel    *CancelCmd
	ExpireDay *ExpireDayCmd
}

// FeeSchedule is the fee rate per liquidity role, in parts per million of notional.
type FeeSchedule struct {
	TakerPPM int64
	MakerPPM int64
}

// DefaultFees is 1% taker (the platform's conversion spread, credit-types.md §3) and 0.5% maker —
// the same schedule the Phase 1 paper portfolio simulates.
var DefaultFees = FeeSchedule{TakerPPM: 10_000, MakerPPM: 5_000}

// RiskCheck is the pre-trade risk seam (credit sufficiency, position limits, surveillance holds). It
// returns a non-empty reason to reject the order. It runs once per order; its answer is journaled.
type RiskCheck func(o Order) string

// Config configures an Engine.
type Config struct {
	Fees FeeSchedule
	Risk RiskCheck // nil accepts everything

	// Persist is the write-ahead hook: it durably records a command before the engine applies it.
	// seq is the command's 1-based position in the journal. If Persist fails the command is refused
	// and nothing changes, so the books never hold state the journal cannot reproduce. Nil keeps the
	// journal in memory only (tests, replay).
	Persist func(seq uint64, c Command) error
}
