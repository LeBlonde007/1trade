package domain

import (
	"errors"
	"time"
)

// CreditType mirrors docs/contracts/credit-types.md (validated against the credit_types table).
type CreditType string

// Operation is the kind of ledger movement (matches credit.tx.v1 + the SQL CHECK).
type Operation string

const (
	OpPurchase    Operation = "purchase"
	OpConsumption Operation = "consumption"
	OpConversion  Operation = "conversion"
	OpMint        Operation = "mint"
	OpBurn        Operation = "burn"
	OpRefund      Operation = "refund"
	OpTrade       Operation = "trade" // Phase 2 (settlement on fill) — unused in Phase 1
)

// ErrInsufficientCredit is returned when a movement would drive a balance negative.
var ErrInsufficientCredit = errors.New("insufficient credit")

// Transaction is one append-only ledger entry. Amount is the SIGNED delta applied to the balance
// (negative for consumption/burn). BalanceBefore/After and ChainHash are filled in by Apply.
type Transaction struct {
	TxID           string
	TenantID       string
	SubAccountID   string // "" when none
	CreditType     CreditType
	Operation      Operation
	Amount         Money
	ReferenceID    string
	IdempotencyKey string
	BalanceBefore  Money
	BalanceAfter   Money
	IsPaper        bool
	CreatedAt      time.Time
	ChainHash      string
}

// Apply is the pure core of the ledger. Given the current balance and the previous chain hash, it
// computes the resulting balance and the hash-chained transaction for a movement. No IO.
//
// `tx.Amount` is the signed delta (negative for debits). Returns ErrInsufficientCredit if the
// resulting balance would go negative. The caller MUST persist the returned Transaction and its
// BalanceAfter atomically — balance update + tx insert in one DB transaction (F05 invariant).
func Apply(prevBalance Money, prevChainHash string, tx Transaction) (Transaction, error) {
	after := prevBalance.Add(tx.Amount)
	if after.IsNegative() {
		return Transaction{}, ErrInsufficientCredit
	}
	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = time.Now().UTC()
	}
	tx.BalanceBefore = prevBalance
	tx.BalanceAfter = after
	tx.ChainHash = computeChainHash(prevChainHash, tx.payload())
	return tx, nil
}

// payload projects a Transaction onto the canonical hashed view (fixed field order + string money).
func (t Transaction) payload() chainPayload {
	return chainPayload{
		TxID:          t.TxID,
		TenantID:      t.TenantID,
		SubAccountID:  t.SubAccountID,
		CreditType:    string(t.CreditType),
		Operation:     string(t.Operation),
		Amount:        t.Amount.String(),
		ReferenceID:   t.ReferenceID,
		BalanceBefore: t.BalanceBefore.String(),
		BalanceAfter:  t.BalanceAfter.String(),
		IsPaper:       t.IsPaper,
		CreatedAt:     t.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// VerifyChain re-derives the hash chain over an ordered slice of transactions and returns the
// index of the first row whose stored ChainHash doesn't match (tampering/corruption), or -1 if the
// chain is intact. `genesis` is the seed used as prev_chain_hash for the first row.
func VerifyChain(genesis string, txs []Transaction) int {
	prev := genesis
	for i, t := range txs {
		if computeChainHash(prev, t.payload()) != t.ChainHash {
			return i
		}
		prev = t.ChainHash
	}
	return -1
}
