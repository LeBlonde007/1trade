package settle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/trade1/matching-engine/internal/engine"
)

// ErrInsufficient means the ledger could not reserve the hold: not enough available balance.
var ErrInsufficient = errors.New("settle: insufficient available balance to reserve")

// Reject reasons the risk adapter returns (orders.state.v1 `reason`).
const (
	ReasonInsufficientCredit = "insufficient_credit"
	ReasonInsufficientCash   = "insufficient_cash"
	ReasonRiskUnavailable    = "risk_unavailable"
)

// post sends one engine-authenticated JSON call and returns the status and the ApiError code.
func (c *Client) post(ctx context.Context, path, idemKey string, body any) (int, string, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return 0, "", fmt.Errorf("settle: encode %s: %w", path, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return 0, "", fmt.Errorf("settle: build %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("settle: %s: %w", path, err)
	}
	defer resp.Body.Close()
	detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	var e struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(detail, &e)
	return resp.StatusCode, e.Code, nil
}

// Reserve reserves an order's hold in the ledger (credit.yaml v1.2 /reserve), keyed on order_id.
// A zero hold needs no reservation and returns nil without calling the ledger.
func (c *Client) Reserve(ctx context.Context, o engine.Order, h engine.Hold) error {
	if h.Amount <= 0 {
		return nil
	}
	var sub *string
	if o.SubAccountID != "" {
		s := o.SubAccountID
		sub = &s
	}
	status, code, err := c.post(ctx, "/v1/credits/reserve", o.OrderID, map[string]any{
		"order_id": o.OrderID, "tenant_id": o.TenantID, "sub_account_id": sub,
		"asset_kind": h.Kind, "asset": h.Asset, "amount": h.Amount.String(), "is_paper": o.IsPaper,
	})
	switch {
	case err != nil:
		return err
	case status == http.StatusOK:
		return nil
	case status == http.StatusPaymentRequired:
		return fmt.Errorf("%w: order %s: %s", ErrInsufficient, o.OrderID, code)
	case status == http.StatusConflict:
		return fmt.Errorf("%w: order %s", ErrConflict, o.OrderID)
	default:
		return fmt.Errorf("settle: reserve order %s: ledger returned %d %s", o.OrderID, status, code)
	}
}

// Release frees whatever an order's reservation still holds. A 404 (nothing was reserved) is success:
// there is nothing to free.
func (c *Client) Release(ctx context.Context, orderID string) error {
	status, code, err := c.post(ctx, "/v1/credits/release", "", map[string]any{"order_id": orderID})
	switch {
	case err != nil:
		return err
	case status == http.StatusOK, status == http.StatusNotFound:
		return nil
	default:
		return fmt.Errorf("settle: release order %s: ledger returned %d %s", orderID, status, code)
	}
}

// ReserveRisk adapts the client into the engine's pre-trade risk check: an order is accepted only if
// its hold is reserved. It fails CLOSED — if the ledger cannot answer, the order is rejected
// (risk_unavailable) rather than accepted unfunded. The engine journals the answer, so replay never
// calls this again.
func ReserveRisk(c *Client, timeout time.Duration) engine.RiskCheck {
	return func(o engine.Order, h engine.Hold) string {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := c.Reserve(ctx, o, h)
		switch {
		case err == nil:
			return ""
		case errors.Is(err, ErrInsufficient) && h.Kind == "credit":
			return ReasonInsufficientCredit
		case errors.Is(err, ErrInsufficient):
			return ReasonInsufficientCash
		default:
			slog.Error("reserve failed; rejecting order (fail closed)", "order_id", o.OrderID, "err", err)
			return ReasonRiskUnavailable
		}
	}
}
