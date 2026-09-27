package engine

import (
	"errors"
	"testing"
)

// TestParseFixed checks parsing is exact, rejects anything the ledger's NUMERIC(20,6) could not hold
// verbatim, and round-trips through String.
func TestParseFixed(t *testing.T) {
	ok := map[string]string{
		"0": "0.000000", "1": "1.000000", "0.001005": "0.001005", "2.5": "2.500000",
		"9223372036854.775807": "9223372036854.775807",
	}
	for in, want := range ok {
		f, err := ParseFixed(in)
		if err != nil || f.String() != want {
			t.Errorf("ParseFixed(%q) = %s, %v; want %s", in, f, err, want)
		}
	}
	for _, in := range []string{"", "-1", "1.0000001", "1e3", "abc", "1.", ".5", "9223372036855", "9223372036854.775808"} {
		if _, err := ParseFixed(in); !errors.Is(err, ErrBadDecimal) {
			t.Errorf("ParseFixed(%q) err = %v, want ErrBadDecimal", in, err)
		}
	}
}

// TestNotional checks price × quantity truncates to six places and reports overflow.
func TestNotional(t *testing.T) {
	n, ok := Notional(MustFixed("0.001005"), MustFixed("1000"))
	if !ok || n.String() != "1.005000" {
		t.Errorf("notional = %s %v, want 1.005000", n, ok)
	}
	n, ok = Notional(MustFixed("0.000001"), MustFixed("0.5"))
	if !ok || n != 0 {
		t.Errorf("sub-micro notional = %s, want truncation to 0", n)
	}
	if _, ok := Notional(MustFixed("9000000000000"), MustFixed("9000000000000")); ok {
		t.Error("overflowing notional reported ok")
	}
}
