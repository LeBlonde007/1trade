// Package ledger is the gateway's client for credit-ledger: a pre-flight balance check so a request
// is rejected with 402 before it consumes a GPU when the tenant has no credit. The actual debit is
// event-driven (inference.usage.v1); this is only the read-side guard. The gateway acts on the
// authenticated tenant's behalf by minting a short-lived tenant JWT with the shared signing secret.
package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/exascale/inference-gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

// Client reads tenant balances from credit-ledger.
type Client struct {
	url       string
	jwtSecret string
	http      *http.Client
}

// NewClient builds a ledger client. Only construct it when a JWT secret is configured (the caller
// passes a nil CreditChecker otherwise, disabling the pre-flight).
func NewClient(cfg config.Config) *Client {
	return &Client{
		url:       strings.TrimRight(cfg.CreditLedgerURL, "/"),
		jwtSecret: cfg.JWTSecret,
		http:      &http.Client{Timeout: cfg.HTTPTimeout},
	}
}

// balancesResponse is the shape of GET /v1/credits/balances.
type balancesResponse struct {
	Balances []struct {
		CreditType string `json:"credit_type"`
		Balance    string `json:"balance"`
		IsPaper    bool   `json:"is_paper"`
	} `json:"balances"`
}

// Sufficient reports whether the tenant holds a positive balance of creditType (matching is_paper),
// and returns that balance string. A precise token-cost pre-authorization is a later refinement;
// "has any credit" is the pre-flight guard for now.
func (c *Client) Sufficient(ctx context.Context, tenantID string, isPaper bool, creditType string) (bool, string, error) {
	tok, err := c.mintToken(tenantID, isPaper)
	if err != nil {
		return false, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/v1/credits/balances", nil)
	if err != nil {
		return false, "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.http.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("balances: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("balances: unexpected status %d", resp.StatusCode)
	}
	var out balancesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, "", err
	}
	for _, b := range out.Balances {
		if b.CreditType == creditType && b.IsPaper == isPaper {
			r, ok := new(big.Rat).SetString(b.Balance)
			if !ok {
				return false, b.Balance, fmt.Errorf("unparseable balance %q", b.Balance)
			}
			return r.Sign() > 0, b.Balance, nil
		}
	}
	return false, "0.000000", nil // no balance row for this credit_type → treat as zero
}

// mintToken issues a short-lived (60s) tenant JWT the ledger accepts (HS256, shared secret).
func (c *Client) mintToken(tenantID string, isPaper bool) (string, error) {
	if c.jwtSecret == "" {
		return "", fmt.Errorf("no JWT secret configured")
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"tenant_id": tenantID, "is_paper": isPaper,
		"iat": now.Unix(), "exp": now.Add(60 * time.Second).Unix(), "iss": "inference-gateway",
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(c.jwtSecret))
}
