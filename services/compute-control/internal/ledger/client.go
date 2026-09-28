// Package ledger is compute-control's client for the credit ledger (credit.yaml): it debits a
// tenant's GPU credits for a reservation (F14). Metered usage does not go through here; it is an
// event (compute.usage.v1) the ledger consumes.
package ledger

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

	"github.com/trade1/compute-control/internal/obs"
)

// ErrInsufficientCredit: the balance is too low (the ledger's 402).
var ErrInsufficientCredit = errors.New("ledger: insufficient credit")

// Client calls the ledger with the service token.
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// New builds a client with a bounded timeout and trace propagation.
func New(baseURL, token string, timeout time.Duration) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Token: token,
		HTTP: &http.Client{Timeout: timeout, Transport: obs.Transport(http.DefaultTransport)}}
}

// Debit is one consumption debit (credit.yaml DebitRequest).
type Debit struct {
	TenantID     string  `json:"tenant_id"`
	SubAccountID *string `json:"sub_account_id,omitempty"`
	CreditType   string  `json:"credit_type"`
	Amount       string  `json:"amount"`
	UsageEventID string  `json:"usage_event_id"`
	IsPaper      bool    `json:"is_paper"`
}

// Debit books d, idempotent on d.UsageEventID (a retry returns the original transaction). It returns
// the ledger transaction id.
func (c *Client) Debit(ctx context.Context, d Debit) (string, error) {
	body, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/credits/debit", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Idempotency-Key", d.UsageEventID)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("ledger: debit: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	switch {
	case resp.StatusCode == http.StatusPaymentRequired:
		return "", ErrInsufficientCredit
	case resp.StatusCode != http.StatusOK:
		return "", fmt.Errorf("ledger: debit: status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var tx struct {
		TxID string `json:"tx_id"`
	}
	if err := json.Unmarshal(raw, &tx); err != nil || tx.TxID == "" {
		return "", fmt.Errorf("ledger: debit: unreadable answer")
	}
	return tx.TxID, nil
}
