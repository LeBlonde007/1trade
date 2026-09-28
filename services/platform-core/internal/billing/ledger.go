package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/trade1/platform-core/internal/config"
	"github.com/trade1/platform-core/internal/obs"
)

// PurchaseBooking is a settled purchase to mint into the ledger.
type PurchaseBooking struct {
	TenantID       string
	Amount         string // fixed-point decimal string
	CreditType     string
	IsPaper        bool
	ReferenceID    string // the purchase id
	IdempotencyKey string // the Stripe event id — the ledger dedupes the mint on this
}

// PurchaseBooker books a settled purchase into the credit ledger.
type PurchaseBooker interface {
	BookPurchase(ctx context.Context, b PurchaseBooking) error
}

// LedgerClient books purchases via credit-ledger's service-authenticated purchase endpoint.
type LedgerClient struct {
	url          string
	serviceToken string
	http         *http.Client
}

// NewLedgerClient builds a ledger client from config.
func NewLedgerClient(cfg config.Config) *LedgerClient {
	return &LedgerClient{
		url:          strings.TrimRight(cfg.CreditLedgerURL, "/"),
		serviceToken: cfg.ServiceToken,
		http:         &http.Client{Timeout: 5 * time.Second, Transport: obs.Transport(nil)}, // traced: the ledger is ours
	}
}

// BookPurchase POSTs /v1/credits/purchase with the service token and Idempotency-Key set to the
// Stripe event id, so a webhook replay never double-mints (the ledger is idempotent on it).
func (c *LedgerClient) BookPurchase(ctx context.Context, b PurchaseBooking) error {
	body, _ := json.Marshal(map[string]any{
		"tenant_id":    b.TenantID,
		"amount":       b.Amount,
		"credit_type":  b.CreditType,
		"is_paper":     b.IsPaper,
		"reference_id": b.ReferenceID,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/v1/credits/purchase", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.serviceToken)
	req.Header.Set("Idempotency-Key", b.IdempotencyKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("book purchase: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("book purchase: ledger returned %d", resp.StatusCode)
	}
	return nil
}

// PaperCashGranter grants a tenant's starting paper cash (credit.yaml v1.1, ADR-0004). It is a
// separate interface from PurchaseBooker because it is not a purchase: nothing was paid, and the cash
// is paper-only by contract.
type PaperCashGranter interface {
	GrantPaperCash(ctx context.Context, tenantID, amount, idempotencyKey string) error
}

// GrantPaperCash POSTs /v1/credits/paper-cash/grant with the service token. The ledger dedupes on
// the Idempotency-Key, so a retried activation never grants twice.
func (c *LedgerClient) GrantPaperCash(ctx context.Context, tenantID, amount, idempotencyKey string) error {
	body, _ := json.Marshal(map[string]any{
		"tenant_id": tenantID,
		"currency":  "USD",
		"amount":    amount,
		"is_paper":  true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/v1/credits/paper-cash/grant", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.serviceToken)
	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("grant paper cash: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("grant paper cash: ledger returned %d", resp.StatusCode)
	}
	return nil
}
