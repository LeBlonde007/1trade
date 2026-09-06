package billing

import (
	"fmt"
	"math/big"
)

// refUSDPerCredit is the authoritative published USD reference price PER CREDIT, by credit type. The
// server owns pricing — a money charge is never derived from a client-supplied dollar amount, only
// from (credits requested × this table). Mirrors the buy-credits screen; keep the two in sync. These
// are indicative reference prices; real pricing follows the published index once the exchange ships.
var refUSDPerCredit = map[string]*big.Rat{
	"text":     ratMust("0.001210"),
	"ai_index": ratMust("0.001005"),
	"speech":   ratMust("0.001200"),
	"image":    ratMust("0.008000"),
	"video":    ratMust("0.250000"),
	"gpu_h100": ratMust("2.990000"),
	"gpu_h200": ratMust("3.490000"),
}

// stripeMinUSDCents is Stripe's minimum card charge ($0.50). Orders below it are rejected up front
// with a clear message rather than failing opaquely at Stripe.
const stripeMinUSDCents = 50

// ratMust parses a decimal string into a *big.Rat or panics — only ever called on the price literals
// above, so a panic means a bad constant, caught at first use in tests.
func ratMust(s string) *big.Rat {
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		panic("billing: bad price literal " + s)
	}
	return r
}

// usdChargeCents computes the integer USD cents to charge for amountCredits of creditType using
// fixed-point math throughout (big.Rat, never floats — money math rule). Returns an error for an
// unpriced credit type, a non-positive/malformed amount, or a total below the card minimum.
func usdChargeCents(creditType, amountCredits string) (int64, error) {
	price, ok := refUSDPerCredit[creditType]
	if !ok {
		return 0, fmt.Errorf("no card price for credit type %q", creditType)
	}
	amt, ok := new(big.Rat).SetString(amountCredits)
	if !ok || amt.Sign() <= 0 {
		return 0, fmt.Errorf("invalid amount %q", amountCredits)
	}
	// cents = round(credits × price_per_credit × 100)
	cents := new(big.Rat).Mul(amt, price)
	cents.Mul(cents, big.NewRat(100, 1))
	n := roundRat(cents)
	if n < stripeMinUSDCents {
		return 0, fmt.Errorf("order total $%d.%02d is below the $0.50 card minimum", n/100, n%100)
	}
	return n, nil
}

// roundRat rounds a non-negative rational to the nearest integer (half up).
func roundRat(r *big.Rat) int64 {
	den := r.Denom()
	num := new(big.Int).Set(r.Num())
	num.Add(num, new(big.Int).Rsh(new(big.Int).Set(den), 1)) // + floor(den/2)
	num.Quo(num, den)
	return num.Int64()
}

// QuoteUSD reports the unit price and the integer cents charged for a purchase, so the caller can
// PERSIST the price a transaction actually happened at.
//
// The price table is administered, not observed, and it is not versioned — if it changes, any
// purchase that did not record its own price becomes unreconstructable. KW01's index is specified
// to observe "prepaid purchase prices"; storing the price at the transaction is what makes those
// observations exist at all.
func QuoteUSD(creditType, amountCredits string) (unitPriceUSD string, cents int64, err error) {
	price, ok := refUSDPerCredit[creditType]
	if !ok {
		return "", 0, fmt.Errorf("no card price for credit type %q", creditType)
	}
	c, err := usdChargeCents(creditType, amountCredits)
	if err != nil {
		return "", 0, err
	}
	// 6dp fixed-point, matching the ledger's money format. FloatString rounds half-away-from-zero
	// on an exact rational — no binary floating point is involved.
	return price.FloatString(6), c, nil
}
