// Package liquidity is the paper liquidity account: a quoting bot that keeps every paper book
// two-sided so the paper market is tradeable from the first minute (commitment #1: paper mode must
// feel like a real market).
//
// It is deliberately narrow:
//   - it trades on paper books only — the engine refuses its orders anywhere else (ErrLiquidityReal);
//   - it is not an internal account, so the insider-risk rule is untouched: internal accounts still
//     never meet customer paper accounts;
//   - it holds paper value only: its cash and credits are paper grants, never real money;
//   - its orders reserve and settle through the ledger exactly like a customer's.
package liquidity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Funder gives the liquidity account its paper inventory through the credit ledger (service token).
// Every grant carries an idempotency key, so a restart or retry never funds twice.
type Funder struct {
	BaseURL string
	Token   string // the shared service token (/v1/credits/mint, /v1/credits/paper-cash/grant)
	Tenant  string
	HTTP    *http.Client
}

// Seed funds the account once for its lifetime: cash plus each credit's starting inventory. The keys
// are fixed, so calling it on every start is a no-op after the first.
func (f *Funder) Seed(ctx context.Context, cashUSD string, credits map[string]string) error {
	if err := f.cash(ctx, "paper-liquidity:seed:v1:usd", cashUSD); err != nil {
		return err
	}
	for credit, amount := range credits {
		if err := f.mint(ctx, "paper-liquidity:seed:v1:"+credit, credit, amount); err != nil {
			return err
		}
	}
	return nil
}

// TopUp refills one asset after the ledger refused an order for lack of it. The key carries the hour,
// so however many orders are refused, an asset is refilled at most once per hour: the account's
// inventory grows only as fast as customers actually take it.
func (f *Funder) TopUp(ctx context.Context, asset, amount string, now time.Time) error {
	key := fmt.Sprintf("paper-liquidity:topup:%s:%s", asset, now.UTC().Format("2006-01-02T15"))
	if strings.EqualFold(asset, "USD") {
		return f.cash(ctx, key, amount)
	}
	return f.mint(ctx, key, asset, amount)
}

// cash grants paper USD.
func (f *Funder) cash(ctx context.Context, key, amount string) error {
	return f.post(ctx, "/v1/credits/paper-cash/grant", key, map[string]any{
		"tenant_id": f.Tenant, "currency": "USD", "amount": amount, "is_paper": true,
	})
}

// mint creates paper credits. is_paper is always true: nothing here can mint real credits.
func (f *Funder) mint(ctx context.Context, key, credit, amount string) error {
	return f.post(ctx, "/v1/credits/mint", key, map[string]any{
		"tenant_id": f.Tenant, "credit_type": credit, "amount": amount, "is_paper": true, "reference_id": key,
	})
}

// post sends one idempotent grant and treats any 2xx as done.
func (f *Funder) post(ctx context.Context, path, key string, body any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(f.BaseURL, "/")+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+f.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	client := f.HTTP
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("liquidity: %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("liquidity: %s returned %d: %s", path, resp.StatusCode, bytes.TrimSpace(msg))
	}
	return nil
}
