package pricing

import "testing"

// TestUnitsForTokens checks the fixed-point billing math: price × tokens / 1000, to 6 decimals.
func TestUnitsForTokens(t *testing.T) {
	cases := []struct {
		price  string
		tokens int
		want   string
	}{
		{"5.000000", 1000, "5.000000"},  // exactly one 1K-token unit
		{"25.000000", 500, "12.500000"}, // half a unit
		{"5.000000", 0, "0.000000"},     // nothing served
		{"25.000000", 1, "0.025000"},    // single token
		{"5.000000", -10, "0.000000"},   // negative clamped to zero
	}
	for _, c := range cases {
		got, err := UnitsForTokens(c.price, c.tokens)
		if err != nil || got != c.want {
			t.Fatalf("UnitsForTokens(%q,%d) = %q,%v; want %q", c.price, c.tokens, got, err, c.want)
		}
	}
	if _, err := UnitsForTokens("not-a-number", 1000); err == nil {
		t.Fatal("expected error on invalid price")
	}
}
