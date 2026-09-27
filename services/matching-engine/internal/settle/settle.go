// Package settle sends matched trades to credit-ledger for atomic settlement
// (POST /v1/credits/settle-trade, credit.yaml v1.1, ADR-0004).
//
// The client is deliberately small and strict about outcomes, because what the caller must do next
// differs completely between them:
//   - nil            — settled (or already settled with this exact body: the ledger replays it).
//   - ErrUnsettleable — the ledger refused the trade for good (402: a side lacked credits or cash;
//     422: the trade breaks a rule). Retrying cannot help; the trade must be handled (SPEC.md §7.3).
//   - ErrConflict    — this trade_id was settled with a DIFFERENT body. A bug or tampering; never retry.
//   - any other error — transient (network, 5xx, timeout). Retry with the same trade: settlement is
//     idempotent on trade_id, so a retry after an unseen success is harmless.
//
// Not wired yet: order entry is closed (F22), so the engine produces no trades to settle.
package settle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/trade1/matching-engine/internal/engine"
)

// Outcome errors — see the package comment.
var (
	ErrUnsettleable = errors.New("settle: ledger refused the trade")
	ErrConflict     = errors.New("settle: trade_id already settled with a different body")
)

// Currency is the quote currency every product is priced in (credit-types.md §6).
const Currency = "USD"

// Client posts trades to the ledger with the engine's own settlement token.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New returns a client. token is SETTLE_SERVICE_TOKEN — the engine's own bearer, never the shared
// service token.
func New(baseURL, token string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: token, http: &http.Client{Timeout: timeout}}
}

// party is one SettleParty on the wire.
type party struct {
	TenantID     string  `json:"tenant_id"`
	SubAccountID *string `json:"sub_account_id"`
	OrderID      string  `json:"order_id"`
	Fee          string  `json:"fee"`
	IsInternal   bool    `json:"is_internal"`
}

// request is the SettleTradeRequest wire shape.
type request struct {
	TradeID    string `json:"trade_id"`
	ProductID  string `json:"product_id"`
	CreditType string `json:"credit_type"`
	Price      string `json:"price"`
	Quantity   string `json:"quantity"`
	Currency   string `json:"currency"`
	Buyer      party  `json:"buyer"`
	Seller     party  `json:"seller"`
	IsPaper    bool   `json:"is_paper"`
	ChainHash  string `json:"chain_hash"`
}

// toParty maps an engine counterparty onto the wire.
func toParty(c engine.Counterparty) party {
	var sub *string
	if c.SubAccountID != "" {
		s := c.SubAccountID
		sub = &s
	}
	return party{TenantID: c.TenantID, SubAccountID: sub, OrderID: c.OrderID, Fee: c.Fee.String(), IsInternal: c.IsInternal}
}

// Body builds the settle-trade request for a trade. Exported so the wire shape can be pinned by tests.
func Body(t engine.Trade) ([]byte, error) {
	b, err := json.Marshal(request{
		TradeID: t.TradeID, ProductID: t.ProductID, CreditType: t.CreditType, Price: t.Price.String(),
		Quantity: t.Quantity.String(), Currency: Currency, Buyer: toParty(t.Buyer), Seller: toParty(t.Seller),
		IsPaper: t.IsPaper, ChainHash: t.ChainHash,
	})
	if err != nil {
		return nil, fmt.Errorf("settle: encode trade %s: %w", t.TradeID, err)
	}
	return b, nil
}

// Settle settles one trade. Idempotency-Key is the trade_id, as the contract requires.
func (c *Client) Settle(ctx context.Context, t engine.Trade) error {
	body, err := Body(t)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/credits/settle-trade", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("settle: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Idempotency-Key", t.TradeID)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("settle: trade %s: %w", t.TradeID, err)
	}
	defer resp.Body.Close()
	detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	var e struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(detail, &e)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusPaymentRequired, http.StatusUnprocessableEntity:
		return fmt.Errorf("%w: trade %s: %d %s", ErrUnsettleable, t.TradeID, resp.StatusCode, e.Code)
	case http.StatusConflict:
		return fmt.Errorf("%w: trade %s", ErrConflict, t.TradeID)
	default:
		// 401/403 are configuration errors (wrong or missing token); 5xx are transient. Neither is a
		// verdict on the trade, so both are reported as retryable-with-attention, not unsettleable.
		return fmt.Errorf("settle: trade %s: ledger returned %d %s", t.TradeID, resp.StatusCode, e.Code)
	}
}
