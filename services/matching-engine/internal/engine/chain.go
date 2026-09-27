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
// ChainHash and PrevChainHash are excluded from the hashed row (prev is the explicit prefix).
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
		`,"sequence":` + strconv.FormatUint(t.Sequence, 10) +
		`,"executed_at":` + strconv.Quote(t.ExecutedAt.UTC().Format(time.RFC3339Nano)) + `}`
}

// canonicalParty renders one counterparty's hashed fields in a fixed order.
func canonicalParty(c Counterparty) string {
	return `{"tenant_id":` + strconv.Quote(c.TenantID) +
		`,"sub_account_id":` + strconv.Quote(c.SubAccountID) +
		`,"order_id":` + strconv.Quote(c.OrderID) +
		`,"fee":"` + c.Fee.String() + `"` +
		`,"liquidity":` + strconv.Quote(c.Liquidity) +
		`,"is_internal":` + strconv.FormatBool(c.IsInternal) + `}`
}

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
