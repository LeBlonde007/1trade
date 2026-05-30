// Package domain is the pure, IO-free core of the credit ledger: fixed-point money, the
// append-only hash chain, and the transaction-apply invariants. No database, no HTTP — so it is
// fully unit-testable. Persistence + handlers wrap this (internal/store, cmd/).
package domain

import (
	"fmt"
	"math/big"
	"strings"
)

// scale is the number of fractional digits we keep — matches SQL NUMERIC(20,6) (credit-types.md §4).
const scale = 6

// scaleFactor is 10^scale, the number of micro-units per whole credit.
var scaleFactor = big.NewInt(1_000_000)

// Money is a fixed-point decimal with 6 fractional digits. Internally it is an arbitrary-precision
// integer count of micro-units (10^-6), so credit math is exact — never a float (ENGINEERING_STANDARDS §1).
type Money struct{ micros *big.Int }

// Zero returns the additive identity (0.000000).
func Zero() Money { return Money{big.NewInt(0)} }

// FromMicros builds Money from a raw micro-unit count (mainly for tests/storage).
func FromMicros(m int64) Money { return Money{big.NewInt(m)} }

// ParseMoney parses a decimal string like "123.456789" (at most 6 fractional digits) into Money.
// It rejects floats-as-text with too much precision so we never silently lose sub-micro value.
func ParseMoney(s string) (Money, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Money{}, fmt.Errorf("empty money string")
	}
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	intPart, fracPart := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, fracPart = s[:i], s[i+1:]
	}
	if len(fracPart) > scale {
		return Money{}, fmt.Errorf("money %q exceeds %d decimal places", s, scale)
	}
	if intPart == "" {
		intPart = "0"
	}
	fracPart += strings.Repeat("0", scale-len(fracPart)) // pad to exactly `scale` digits
	m, ok := new(big.Int).SetString(intPart+fracPart, 10)
	if !ok {
		return Money{}, fmt.Errorf("invalid money %q", s)
	}
	if neg {
		m.Neg(m)
	}
	return Money{m}, nil
}

// i returns the underlying micro-unit integer, treating a nil/zero-value Money as 0.
func (m Money) i() *big.Int {
	if m.micros == nil {
		return big.NewInt(0)
	}
	return m.micros
}

// String renders the canonical fixed-point form with exactly 6 decimals, e.g. "100.000000".
func (m Money) String() string {
	n := new(big.Int).Abs(m.i())
	q, r := new(big.Int), new(big.Int)
	q.QuoRem(n, scaleFactor, r)
	sign := ""
	if m.i().Sign() < 0 {
		sign = "-"
	}
	return fmt.Sprintf("%s%s.%06d", sign, q.String(), r.Int64())
}

// Add returns m + o.
func (m Money) Add(o Money) Money { return Money{new(big.Int).Add(m.i(), o.i())} }

// Sub returns m - o.
func (m Money) Sub(o Money) Money { return Money{new(big.Int).Sub(m.i(), o.i())} }

// Cmp compares: -1 if m<o, 0 if equal, +1 if m>o.
func (m Money) Cmp(o Money) int { return m.i().Cmp(o.i()) }

// Sign reports -1, 0, or +1.
func (m Money) Sign() int { return m.i().Sign() }

// IsNegative reports whether the value is below zero.
func (m Money) IsNegative() bool { return m.i().Sign() < 0 }

// MarshalJSON serialises Money as a decimal STRING (never a JSON number) to preserve precision.
func (m Money) MarshalJSON() ([]byte, error) { return []byte(`"` + m.String() + `"`), nil }

// UnmarshalJSON parses Money from a decimal string.
func (m *Money) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	parsed, err := ParseMoney(s)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// ConvertedAmount returns floor(amount × rate × (1 − spread)) at 6-dp (F07). `rate` is `to`-units per
// 1 `from`-unit (pre-spread); `spread` is the house fraction (e.g. "0.01"). It floors — rounds down —
// so the house never over-credits, and uses exact big.Rat math (never a float).
func ConvertedAmount(amount Money, rate, spread string) (Money, error) {
	r, ok := new(big.Rat).SetString(rate)
	if !ok || r.Sign() <= 0 {
		return Money{}, fmt.Errorf("invalid rate %q", rate)
	}
	sp, ok := new(big.Rat).SetString(spread)
	if !ok || sp.Sign() < 0 || sp.Cmp(big.NewRat(1, 1)) >= 0 {
		return Money{}, fmt.Errorf("invalid spread %q", spread)
	}
	amt := new(big.Rat).SetFrac(amount.i(), scaleFactor) // micros → whole-credit rational
	oneMinus := new(big.Rat).Sub(big.NewRat(1, 1), sp)
	res := new(big.Rat).Mul(amt, r)
	res.Mul(res, oneMinus)
	scaled := new(big.Rat).Mul(res, new(big.Rat).SetInt(scaleFactor)) // back to micro-units (rational)
	micros := new(big.Int).Div(scaled.Num(), scaled.Denom())          // floor (operands non-negative)
	return Money{micros}, nil
}
