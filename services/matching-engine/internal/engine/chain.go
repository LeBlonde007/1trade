package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

// tradeChainHash computes SHA-256(prev || canonical_json(trade)) — the construction the credit ledger
// and the index already use. The canonical form is written by hand with a fixed key order and
// fixed-point strings, so it does not depend on a JSON library's map ordering or float formatting.
//
// The hashed fields are exactly the trades.executed.v1 payload fields, minus chain_hash and
// prev_chain_hash (prev is the explicit prefix), in contract order. Empty sub_account_id renders as
// null, as in the payload. So any consumer can re-verify the chain from the events alone. Sequence is
// deliberately not hashed: the contract does not carry it; the prev links already fix the order.
func tradeChainHash(prev string, t *Trade) string {
	sum := sha256.Sum256([]byte(prev + canonicalTrade(t)))
	return hex.EncodeToString(sum[:])
}

// canonicalTrade renders the trade's hashed fields in a fixed order.
func canonicalTrade(t *Trade) string {
	return `{"trade_id":` + strconv.Quote(t.TradeID) +
		`,"product_id":` + strconv.Quote(t.ProductID) +
		`,"credit_type":` + strconv.Quote(t.CreditType) +
		`,"price":"` + t.Price.String() + `"` +
		`,"quantity":"` + t.Quantity.String() + `"` +
		`,"buyer":` + canonicalParty(t.Buyer) +
		`,"seller":` + canonicalParty(t.Seller) +
		`,"aggressor_side":` + strconv.Quote(string(t.AggressorSide)) +
		`,"is_paper":` + strconv.FormatBool(t.IsPaper) +
		`,"executed_at":` + strconv.Quote(FormatTS(t.ExecutedAt)) + `}`
}

// canonicalParty renders one counterparty's hashed fields in a fixed order.
func canonicalParty(c Counterparty) string {
	return `{"tenant_id":` + strconv.Quote(c.TenantID) +
		`,"sub_account_id":` + nullable(c.SubAccountID) +
		`,"order_id":` + strconv.Quote(c.OrderID) +
		`,"fee":"` + c.Fee.String() + `"` +
		`,"liquidity":` + strconv.Quote(c.Liquidity) +
		`,"is_internal":` + strconv.FormatBool(c.IsInternal) + `}`
}

// nullable renders s as a JSON string, or null when empty.
func nullable(s string) string {
	if s == "" {
		return "null"
	}
	return strconv.Quote(s)
}

// FormatTS is the one timestamp format for hashed rows and event payloads: RFC 3339, UTC, with
// nanoseconds when present. Payload and hash must use the same string or verification fails.
func FormatTS(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

// VerifyTradeChain reports whether trades (one book, in sequence order) form an unbroken chain: each
// links to its predecessor and its hash recomputes from its own fields.
func VerifyTradeChain(trades []Trade) bool {
	prev := ""
	for i := range trades {
		t := &trades[i]
		if t.PrevChainHash != prev || tradeChainHash(prev, t) != t.ChainHash {
			return false
		}
		prev = t.ChainHash
	}
	return true
}
