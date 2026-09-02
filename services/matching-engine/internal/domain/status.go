package domain

// Exchange states from openapi/trading.yaml ExchangeStatus.
const (
	StatePaused = "paused"
	StateLive   = "live"
)

// PausedCode is the error code every write endpoint returns while the exchange is paused. Clients
// switch on this to render the paused state honestly instead of faking a fill.
const PausedCode = "EXCHANGE_PAUSED"

// PausedMessage explains the pause in customer-facing language. Per F22's regulatory framing, copy
// says the exchange is not open — it never implies an order was received, queued, or will fill.
const PausedMessage = "Order entry is paused pending exchange licensing. No orders are accepted."

// PausedReason is the longer form shown alongside market data, making clear the data is simulated.
const PausedReason = "Order entry is paused pending exchange licensing. Market data shown is simulated."

// ExchangeStatus tells a client whether order entry is open and where the methodology lives.
type ExchangeStatus struct {
	State          string
	Reason         string
	MethodologyURL string
}

// Status returns the current exchange status. Phase 1 is unconditionally paused: there is no flag,
// env var, or request that opens order entry. Switch-on is a licensed, deliberate code change
// (KW03 cutover) — not a runtime toggle someone can flip by accident.
func Status(methodologyURL string) ExchangeStatus {
	return ExchangeStatus{
		State:          StatePaused,
		Reason:         PausedReason,
		MethodologyURL: methodologyURL,
	}
}
