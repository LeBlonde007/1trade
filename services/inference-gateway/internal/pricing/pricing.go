// Package pricing computes the billable `units` for a request from the catalog price. All math is
// exact fixed-point (math/big.Rat) — never float — and the result is a 6-decimal string to match the
// ledger's NUMERIC(20,6) and the inference.usage.v1 `units` field.
package pricing

import (
	"fmt"
	"math/big"
)

// tokensPerUnit is the token block a token-priced model bills in ("1K tokens").
const tokensPerUnit = 1000

// UnitsForTokens returns price × totalTokens / 1000 as a fixed-point 6-decimal string. `price` is the
// catalog's credits-per-1K-tokens decimal string. Errors if the price isn't a valid decimal.
func UnitsForTokens(price string, totalTokens int) (string, error) {
	p, ok := new(big.Rat).SetString(price)
	if !ok {
		return "", fmt.Errorf("invalid price %q", price)
	}
	if totalTokens < 0 {
		totalTokens = 0
	}
	units := new(big.Rat).Mul(p, big.NewRat(int64(totalTokens), tokensPerUnit))
	return units.FloatString(6), nil
}

// UnitsForCount returns price × count as a fixed-point 6-decimal string, for per-item priced models
// (image: credits per image; video: credits per clip). `price` is the catalog's credits-per-unit.
func UnitsForCount(price string, count int) (string, error) {
	p, ok := new(big.Rat).SetString(price)
	if !ok {
		return "", fmt.Errorf("invalid price %q", price)
	}
	if count < 0 {
		count = 0
	}
	units := new(big.Rat).Mul(p, new(big.Rat).SetInt64(int64(count)))
	return units.FloatString(6), nil
}
