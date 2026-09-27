// Package events encodes matching-engine output as the frozen NATS contracts:
// docs/contracts/events/orders.state.v1.yaml and trades.executed.v1.yaml.
//
// Encoding only — nothing here publishes. Publishing waits on the order-entry cutover (SPEC.md §7);
// the encoders exist now so the wire shape is pinned by tests against the contract files, and so
// surveillance and the ledger can build consumers against real engine output.
package events

import (
	"encoding/json"
	"fmt"

	"github.com/trade1/matching-engine/internal/engine"
)

// Subjects from the contracts.
const (
	SubjectOrdersState    = "orders.state.v1"
	SubjectTradesExecuted = "trades.executed.v1"
)

// orderState is the orders.state.v1 payload. Field order follows the contract.
type orderState struct {
	EventID        string  `json:"event_id"`
	OrderID        string  `json:"order_id"`
	TenantID       string  `json:"tenant_id"`
	SubAccountID   *string `json:"sub_account_id"`
	ProductID      string  `json:"product_id"`
	Side           string  `json:"side"`
	OrderType      string  `json:"order_type"`
	State          string  `json:"state"`
	Reason         *string `json:"reason"`
	Quantity       string  `json:"quantity"`
	FilledQuantity string  `json:"filled_quantity"`
	LimitPrice     *string `json:"limit_price"`
	IsPaper        bool    `json:"is_paper"`
	IsInternal     bool    `json:"is_internal"`
	Sequence       uint64  `json:"sequence"`
	TS             string  `json:"ts"`
}

// counterparty is one side of a trades.executed.v1 payload.
type counterparty struct {
	TenantID     string  `json:"tenant_id"`
	SubAccountID *string `json:"sub_account_id"`
	OrderID      string  `json:"order_id"`
	Fee          string  `json:"fee"`
	Liquidity    string  `json:"liquidity"`
	IsInternal   bool    `json:"is_internal"`
}

// tradeExecuted is the trades.executed.v1 payload. Field order follows the contract.
type tradeExecuted struct {
	TradeID       string       `json:"trade_id"`
	ProductID     string       `json:"product_id"`
	CreditType    string       `json:"credit_type"`
	Price         string       `json:"price"`
	Quantity      string       `json:"quantity"`
	Buyer         counterparty `json:"buyer"`
	Seller        counterparty `json:"seller"`
	AggressorSide string       `json:"aggressor_side"`
	IsPaper       bool         `json:"is_paper"`
	ChainHash     string       `json:"chain_hash"`
	PrevChainHash *string      `json:"prev_chain_hash"`
	ExecutedAt    string       `json:"executed_at"`
}

// opt returns nil for an empty string, so the payload carries JSON null as the contract requires.
func opt(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// OrderState encodes one order transition as an orders.state.v1 payload.
func OrderState(e engine.OrderEvent) ([]byte, error) {
	var limit *string
	if e.Type != engine.Market {
		limit = opt(e.LimitPrice.String())
	}
	b, err := json.Marshal(orderState{
		EventID: e.EventID, OrderID: e.OrderID, TenantID: e.TenantID, SubAccountID: opt(e.SubAccountID),
		ProductID: e.ProductID, Side: string(e.Side), OrderType: string(e.Type), State: string(e.State),
		Reason: opt(e.Reason), Quantity: e.Quantity.String(), FilledQuantity: e.FilledQuantity.String(),
		LimitPrice: limit, IsPaper: e.IsPaper, IsInternal: e.IsInternal, Sequence: e.Sequence,
		TS: engine.FormatTS(e.TS),
	})
	if err != nil {
		return nil, fmt.Errorf("events: encode order %s: %w", e.OrderID, err)
	}
	return b, nil
}

// TradeExecuted encodes one trade as a trades.executed.v1 payload.
func TradeExecuted(t engine.Trade) ([]byte, error) {
	party := func(c engine.Counterparty) counterparty {
		return counterparty{TenantID: c.TenantID, SubAccountID: opt(c.SubAccountID), OrderID: c.OrderID,
			Fee: c.Fee.String(), Liquidity: c.Liquidity, IsInternal: c.IsInternal}
	}
	b, err := json.Marshal(tradeExecuted{
		TradeID: t.TradeID, ProductID: t.ProductID, CreditType: t.CreditType, Price: t.Price.String(),
		Quantity: t.Quantity.String(), Buyer: party(t.Buyer), Seller: party(t.Seller),
		AggressorSide: string(t.AggressorSide), IsPaper: t.IsPaper, ChainHash: t.ChainHash,
		PrevChainHash: opt(t.PrevChainHash), ExecutedAt: engine.FormatTS(t.ExecutedAt),
	})
	if err != nil {
		return nil, fmt.Errorf("events: encode trade %s: %w", t.TradeID, err)
	}
	return b, nil
}

// Encode returns the subject and payload for one engine event.
func Encode(ev engine.Event) (string, []byte, error) {
	if ev.Trade != nil {
		b, err := TradeExecuted(*ev.Trade)
		return SubjectTradesExecuted, b, err
	}
	if ev.Order != nil {
		b, err := OrderState(*ev.Order)
		return SubjectOrdersState, b, err
	}
	return "", nil, fmt.Errorf("events: empty event")
}
