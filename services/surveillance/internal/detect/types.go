// Package detect is the surveillance rule engine (KW05): pure, deterministic detectors over the
// matching engine's event contracts (orders.state.v1, trades.executed.v1) that raise
// surveillance.alert.v1 alerts.
//
// Deterministic by construction: every window is measured in EVENT time (the events' own
// timestamps), never the wall clock, and alert ids derive from (rule, tenant, product, is_paper,
// window bucket). Replaying a stream therefore yields the same alerts, and a replayed alert is never
// raised twice. Paper and real-money flow are tracked in separate state and never mix.
//
// Surveillance detects and recommends; it never mutates another service (Alert.Action says what it
// asks the owning service to do).
package detect

import (
	"errors"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Order is an orders.state.v1 payload.
type Order struct {
	EventID        string    `json:"event_id"`
	OrderID        string    `json:"order_id"`
	TenantID       string    `json:"tenant_id"`
	SubAccountID   *string   `json:"sub_account_id"`
	ProductID      string    `json:"product_id"`
	Side           string    `json:"side"`
	OrderType      string    `json:"order_type"`
	State          string    `json:"state"`
	Reason         *string   `json:"reason"`
	Quantity       string    `json:"quantity"`
	FilledQuantity string    `json:"filled_quantity"`
	LimitPrice     *string   `json:"limit_price"`
	IsPaper        bool      `json:"is_paper"`
	IsInternal     bool      `json:"is_internal"`
	Sequence       uint64    `json:"sequence"`
	TS             time.Time `json:"ts"`
}

// Party is one side of a trades.executed.v1 payload.
type Party struct {
	TenantID     string  `json:"tenant_id"`
	SubAccountID *string `json:"sub_account_id"`
	OrderID      string  `json:"order_id"`
	Fee          string  `json:"fee"`
	Liquidity    string  `json:"liquidity"`
	IsInternal   bool    `json:"is_internal"`
}

// Trade is a trades.executed.v1 payload.
type Trade struct {
	TradeID       string    `json:"trade_id"`
	ProductID     string    `json:"product_id"`
	CreditType    string    `json:"credit_type"`
	Price         string    `json:"price"`
	Quantity      string    `json:"quantity"`
	Buyer         Party     `json:"buyer"`
	Seller        Party     `json:"seller"`
	AggressorSide string    `json:"aggressor_side"`
	IsPaper       bool      `json:"is_paper"`
	ChainHash     string    `json:"chain_hash"`
	PrevChainHash *string   `json:"prev_chain_hash"`
	ExecutedAt    time.Time `json:"executed_at"`
}

// Rules (surveillance.alert.v1 `rule`).
const (
	RuleWash         = "wash_trade"
	RuleSpoofing     = "spoofing"
	RuleLayering     = "layering"
	RuleMarkClose    = "marking_the_close"
	RuleCrossProduct = "cross_product"
	RuleExcessCancel = "excessive_cancellation"
	RulePosLimit     = "position_limit"
)

// Actions (surveillance.alert.v1 `action`) — recommendations to the owning service.
const (
	ActionReview     = "review"
	ActionRateLimit  = "rate_limit"
	ActionExclude    = "exclude_from_index"
	ActionSuspendRec = "suspend_recommended"
)

// Alert is a surveillance.alert.v1 payload.
type Alert struct {
	AlertID              string         `json:"alert_id"`
	Rule                 string         `json:"rule"`
	Severity             string         `json:"severity"`
	Action               string         `json:"action"`
	TenantID             string         `json:"tenant_id"`
	CounterpartyTenantID *string        `json:"counterparty_tenant_id"`
	ProductID            string         `json:"product_id"`
	RelatedProductID     *string        `json:"related_product_id"`
	IsPaper              bool           `json:"is_paper"`
	WindowStart          time.Time      `json:"window_start"`
	WindowEnd            time.Time      `json:"window_end"`
	DetectedAt           time.Time      `json:"detected_at"`
	Evidence             map[string]any `json:"evidence"`
	TradeIDs             []string       `json:"trade_ids"`
	OrderIDs             []string       `json:"order_ids"`
}

// Fixed is a fixed-point amount in micro-units (6 dp) — the contracts' Decimal. No floats.
type Fixed int64

const unit = 1_000_000

var decRE = regexp.MustCompile(`^(-)?(\d+)(?:\.(\d{1,6}))?$`)

// errDecimal is returned for a malformed decimal string.
var errDecimal = errors.New("detect: malformed decimal")

// parseFixed parses a contract decimal string.
func parseFixed(s string) (Fixed, error) {
	m := decRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, errDecimal
	}
	w, err := strconv.ParseInt(m[2], 10, 64)
	if err != nil || w > (1<<62)/unit {
		return 0, errDecimal
	}
	var f int64
	if m[3] != "" {
		f, _ = strconv.ParseInt(m[3]+strings.Repeat("0", 6-len(m[3])), 10, 64)
	}
	v := Fixed(w*unit + f)
	if m[1] != "" {
		v = -v
	}
	return v, nil
}

// fx parses leniently: a malformed value counts as zero (the engine never emits one; a detector
// must not panic on a bad event).
func fx(s string) Fixed {
	v, _ := parseFixed(s)
	return v
}

// String renders 6-dp fixed point.
func (f Fixed) String() string {
	neg := f < 0
	v := int64(f)
	if neg {
		v = -v
	}
	frac := strconv.FormatInt(v%unit, 10)
	s := strconv.FormatInt(v/unit, 10) + "." + strings.Repeat("0", 6-len(frac)) + frac
	if neg {
		return "-" + s
	}
	return s
}

// abs returns |f|.
func (f Fixed) abs() Fixed {
	if f < 0 {
		return -f
	}
	return f
}

// mul returns floor(a×b) at 6 dp, saturating instead of overflowing (surveillance compares against
// thresholds; a saturated huge value still crosses them).
func mul(a, b Fixed) Fixed {
	p := new(big.Int).Mul(big.NewInt(int64(a)), big.NewInt(int64(b)))
	p.Quo(p, big.NewInt(unit))
	if !p.IsInt64() {
		if p.Sign() < 0 {
			return Fixed(-1 << 62)
		}
		return Fixed(1 << 62)
	}
	return Fixed(p.Int64())
}

// ratio returns a/b as fixed point (b > 0).
func ratio(a, b int64) Fixed {
	return Fixed(a * unit / b)
}
