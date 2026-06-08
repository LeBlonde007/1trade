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

// TestUnitsForCount checks the per-item billing math: price × count, to 6 decimals.
func TestUnitsForCount(t *testing.T) {
	cases := []struct {
		price string
		count int
		want  string
	}{
		{"80.000000", 1, "80.000000"},   // one image
		{"80.000000", 3, "240.000000"},  // three images
		{"120.000000", 0, "0.000000"},   // nothing generated
		{"600.000000", 1, "600.000000"}, // one video clip
		{"30.000000", -2, "0.000000"},   // negative clamped to zero
	}
	for _, c := range cases {
		got, err := UnitsForCount(c.price, c.count)
		if err != nil || got != c.want {
			t.Fatalf("UnitsForCount(%q,%d) = %q,%v; want %q", c.price, c.count, got, err, c.want)
		}
	}
	if _, err := UnitsForCount("not-a-number", 1); err == nil {
		t.Fatal("expected error on invalid price")
	}
}
