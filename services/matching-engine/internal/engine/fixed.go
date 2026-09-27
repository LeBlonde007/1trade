package engine

import (
	"errors"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// Scale is the number of decimal places every price, quantity and fee carries — the same
// NUMERIC(20,6) scale the credit ledger stores, so a value round-trips between them exactly.
const Scale = 6

// unit is 10^Scale: the integer value of 1.000000.
const unit int64 = 1_000_000

// Fixed is a non-float decimal held as an integer count of micro-units (1 = 0.000001). All matching
// arithmetic is integer arithmetic on Fixed; floats never touch a price, quantity, or fee.
type Fixed int64

// ErrBadDecimal is returned for a string that is not a non-negative decimal with at most Scale places,
// or that does not fit in a Fixed.
var ErrBadDecimal = errors.New("engine: not a non-negative decimal with at most 6 places")

var decimalRE = regexp.MustCompile(`^(\d+)(?:\.(\d{1,6}))?$`)

// ParseFixed parses a non-negative fixed-point string such as "0.001005" or "1000". More than six
// decimal places is an error rather than a silent rounding: a price the caller did not send must never
// reach the book.
func ParseFixed(s string) (Fixed, error) {
	m := decimalRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, ErrBadDecimal
	}
	whole, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || whole > (1<<63-1)/unit {
		return 0, ErrBadDecimal
	}
	frac := int64(0)
	if m[2] != "" {
		f, err := strconv.ParseInt(m[2]+strings.Repeat("0", Scale-len(m[2])), 10, 64)
		if err != nil {
			return 0, ErrBadDecimal
		}
		frac = f
	}
	v := whole*unit + frac
	if v < 0 {
		return 0, ErrBadDecimal
	}
	return Fixed(v), nil
}

// MustFixed parses s and returns 0 on error. For tests and compile-time constants only — request
// paths use ParseFixed and handle the error.
func MustFixed(s string) Fixed {
	f, err := ParseFixed(s)
	if err != nil {
		return 0
	}
	return f
}

// String renders the value with exactly Scale decimal places ("1000.000000"), the wire format of every
// Decimal in the trading contract.
func (f Fixed) String() string {
	neg := f < 0
	v := int64(f)
	if neg {
		v = -v
	}
	s := strconv.FormatInt(v/unit, 10) + "." + leftPad(strconv.FormatInt(v%unit, 10), Scale)
	if neg {
		return "-" + s
	}
	return s
}

// leftPad pads s with leading zeros to width n.
func leftPad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return strings.Repeat("0", n-len(s)) + s
}

// mulDiv returns floor(a*b/d) for non-negative a, b and positive d, computed without intermediate
// overflow, and false if the result does not fit in a Fixed. It is how notional (price × quantity) and
// fees (notional × rate) are computed.
func mulDiv(a, b Fixed, d int64) (Fixed, bool) {
	if a < 0 || b < 0 || d <= 0 {
		return 0, false
	}
	p := new(big.Int).Mul(big.NewInt(int64(a)), big.NewInt(int64(b)))
	p.Quo(p, big.NewInt(d))
	if !p.IsInt64() {
		return 0, false
	}
	return Fixed(p.Int64()), true
}

// Notional returns price × quantity, truncated to Scale places, and whether it fits.
func Notional(price, qty Fixed) (Fixed, bool) {
	return mulDiv(price, qty, unit)
}
