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
	"strings"
	"time"

	"github.com/trade1/matching-engine/internal/engine"
)

// ErrInsufficient means the ledger could not reserve the hold: not enough available balance.
var ErrInsufficient = errors.New("settle: insufficient available balance to reserve")

// ErrReservationUnusable means the ledger answered a reserve with a reservation that no longer holds
// the order's full amount: it was released, or partly consumed, under this order_id before. A released
// reservation locks nothing, so accepting the order would leave it unfunded. The order_id is spent.
var ErrReservationUnusable = errors.New("settle: the reservation for this order_id is closed or partly spent")

// Reject reasons the risk adapter returns (orders.state.v1 `reason`).
const (
	ReasonInsufficientCredit = "insufficient_credit"
	ReasonInsufficientCash   = "insufficient_cash"
	ReasonRiskUnavailable    = "risk_unavailable"
	// ReasonOrderIDReused: the order_id's reservation was already closed; submit with a new order_id.
	ReasonOrderIDReused = "order_id_reused"
)

// maxBody bounds how much of a ledger response is read: enough for a Reservation or an ApiError.
const maxBody = 4 << 10

// apiCode extracts an ApiError's code from a response body ("" when there is none).
func apiCode(body []byte) string {
	var e struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(body, &e)
	return e.Code
}

// post sends one engine-authenticated JSON call and returns the status and the response body.
func (c *Client) post(ctx context.Context, path, idemKey string, body any) (int, []byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return 0, nil, fmt.Errorf("settle: encode %s: %w", path, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return 0, nil, fmt.Errorf("settle: build %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("settle: %s: %w", path, err)
	}
	defer resp.Body.Close()
	detail, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return 0, nil, fmt.Errorf("settle: %s: read response: %w", path, err)
	}
	return resp.StatusCode, detail, nil
}

// checkReservation verifies that a 200 answer to reserve really locks the order's whole hold. The
// ledger answers a replay (an order_id it has seen) with the reservation's CURRENT state, so a replay
// of a released or partly consumed reservation is a 200 too. Taking that as "reserved" would accept an
// order that has nothing locked behind it.
func checkReservation(body []byte, orderID string, hold engine.Fixed) error {
	var r struct {
		OrderID   string `json:"order_id"`
		State     string `json:"state"`
		Amount    string `json:"amount"`
		Remaining string `json:"remaining"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("settle: reserve order %s: unreadable reservation: %w", orderID, err)
	}
	amount, errA := engine.ParseFixed(r.Amount)
	remaining, errR := engine.ParseFixed(r.Remaining)
	if errA != nil || errR != nil || !strings.EqualFold(r.OrderID, orderID) || amount != hold {
		return fmt.Errorf("settle: reserve order %s: the ledger answered with a different reservation (order %q, amount %q)", orderID, r.OrderID, r.Amount)
	}
	if r.State != "open" || remaining != hold {
		return fmt.Errorf("%w: order %s is %s with %s of %s left", ErrReservationUnusable, orderID, r.State, r.Remaining, r.Amount)
	}
	return nil
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
	status, body, err := c.post(ctx, "/v1/credits/reserve", o.OrderID, map[string]any{
		"order_id": o.OrderID, "tenant_id": o.TenantID, "sub_account_id": sub,
		"asset_kind": h.Kind, "asset": h.Asset, "amount": h.Amount.String(), "is_paper": o.IsPaper,
	})
	code := apiCode(body)
	switch {
	case err != nil:
		return err
	case status == http.StatusOK:
		return checkReservation(body, o.OrderID, h.Amount)
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
	status, body, err := c.post(ctx, "/v1/credits/release", "", map[string]any{"order_id": orderID})
	switch {
	case err != nil:
		return err
	case status == http.StatusOK, status == http.StatusNotFound:
		return nil
	default:
		return fmt.Errorf("settle: release order %s: ledger returned %d %s", orderID, status, apiCode(body))
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
		case errors.Is(err, ErrReservationUnusable):
			slog.Warn("reservation for this order_id is already closed; rejecting (the id is spent)", "order_id", o.OrderID, "err", err)
			return ReasonOrderIDReused
		default:
			slog.Error("reserve failed; rejecting order (fail closed)", "order_id", o.OrderID, "err", err)
			return ReasonRiskUnavailable
		}
	}
}
