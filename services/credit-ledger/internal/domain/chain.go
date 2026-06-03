package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// chainPayload is the canonical, fixed-field-order view of a transaction that gets hashed.
// encoding/json marshals struct fields in declaration order, so this defines a deterministic
// canonical_json — the same row always produces the same bytes (and thus the same hash).
type chainPayload struct {
	TxID          string `json:"tx_id"`
	TenantID      string `json:"tenant_id"`
	SubAccountID  string `json:"sub_account_id"`
	CreditType    string `json:"credit_type"`
	Operation     string `json:"operation"`
	Amount        string `json:"amount"`
	ReferenceID   string `json:"reference_id"`
	BalanceBefore string `json:"balance_before"`
	BalanceAfter  string `json:"balance_after"`
	IsPaper       bool   `json:"is_paper"`
	CreatedAt     string `json:"created_at"`
}

// canonicalJSON renders the deterministic byte representation of a transaction row for hashing.
func canonicalJSON(p chainPayload) []byte {
	b, _ := json.Marshal(p) // cannot fail for this fixed struct of strings/bool
	return b
}

// computeChainHash extends the audit hash chain (credit-types.md §4 / CLAUDE.md):
//
//	chain_hash = sha256( prev_chain_hash || canonical_json(row) )
//
// hex-encoded. The genesis link uses an agreed seed string as prev_chain_hash.
func computeChainHash(prevChainHash string, p chainPayload) string {
	h := sha256.New()
	_, _ = h.Write([]byte(prevChainHash)) // hash.Hash.Write never returns an error
	_, _ = h.Write(canonicalJSON(p))
	return hex.EncodeToString(h.Sum(nil))
}
