package domain

import (
	"errors"
	"time"
)

// Currency is a quote currency (credit.yaml Currency). It is NOT a credit type (credit-types.md §6).
type Currency string

// USD is the only quote currency in credit.yaml v1.1.
const USD Currency = "USD"

// CashOperation is the kind of cash movement (matches cash.tx.v1 + the SQL CHECK).
type CashOperation string

// Cash operations.
const (
	CashPaperGrant CashOperation = "paper_grant"
	CashTrade      CashOperation = "trade"
	CashFee        CashOperation = "fee"
)

// ErrInsufficientCash is returned when a cash movement would drive a balance negative.
var ErrInsufficientCash = errors.New("insufficient cash")

// CashTx is one append-only cash ledger entry. Amount is the SIGNED delta. Cash is paper-only until
// counsel clears custody (ADR-0004); IsPaper is still carried so the rule is visible at every layer.
type CashTx struct {
	TxID           string
	TenantID       string
	SubAccountID   string
	Currency       Currency
	Operation      CashOperation
	Amount         Money
	ReferenceID    string
	IdempotencyKey string
	BalanceBefore  Money
	BalanceAfter   Money
	IsPaper        bool
	CreatedAt      time.Time
	ChainHash      string
}

// cashPayload is the canonical, fixed-order hashed view of a cash row. It is a separate struct from
// chainPayload so a cash row can never hash identically to a credit row (currency vs credit_type).
type cashPayload struct {
	TxID          string `json:"tx_id"`
	TenantID      string `json:"tenant_id"`
	SubAccountID  string `json:"sub_account_id"`
	Currency      string `json:"currency"`
	Operation     string `json:"operation"`
	Amount        string `json:"amount"`
	ReferenceID   string `json:"reference_id"`
	BalanceBefore string `json:"balance_before"`
	BalanceAfter  string `json:"balance_after"`
	IsPaper       bool   `json:"is_paper"`
	CreatedAt     string `json:"created_at"`
}

// payload projects a CashTx onto its hashed view.
func (t CashTx) payload() cashPayload {
	return cashPayload{
		TxID: t.TxID, TenantID: t.TenantID, SubAccountID: t.SubAccountID, Currency: string(t.Currency),
		Operation: string(t.Operation), Amount: t.Amount.String(), ReferenceID: t.ReferenceID,
		BalanceBefore: t.BalanceBefore.String(), BalanceAfter: t.BalanceAfter.String(),
		IsPaper: t.IsPaper, CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// ApplyCash is the pure core of a cash movement, the twin of Apply: it computes the resulting balance
// and the hash-chained row, or ErrInsufficientCash if the balance would go negative.
func ApplyCash(prevBalance Money, prevChainHash string, tx CashTx) (CashTx, error) {
	after := prevBalance.Add(tx.Amount)
	if after.IsNegative() {
		return CashTx{}, ErrInsufficientCash
	}
	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = time.Now().UTC()
	}
	tx.BalanceBefore = prevBalance
	tx.BalanceAfter = after
	tx.ChainHash = hashJSON(prevChainHash, tx.payload())
	return tx, nil
}

// VerifyCashChain re-derives a cash chain and returns the index of the first bad row, or -1.
func VerifyCashChain(genesis string, txs []CashTx) int {
	prev := genesis
	for i, t := range txs {
		if hashJSON(prev, t.payload()) != t.ChainHash {
			return i
		}
		prev = t.ChainHash
	}
	return -1
}
