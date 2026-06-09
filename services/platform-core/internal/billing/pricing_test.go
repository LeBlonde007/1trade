package billing

import "testing"

// TestUsdChargeCents checks the server-side credit→cents pricing is exact (fixed-point, half-up).
func TestUsdChargeCents(t *testing.T) {
	cases := []struct {
		creditType, amount string
		want               int64
	}{
		{"text", "100000", 12100},     // 100000 × 0.001210 × 100
		{"text", "100000.000000", 12100},
		{"ai_index", "50000", 5025},   // 50000 × 0.001005 = 50.25 → 5025c
		{"image", "10000", 8000},      // 10000 × 0.008 = 80.00
		{"video", "100", 2500},        // 100 × 0.25 = 25.00
		{"gpu_h100", "8", 2392},       // 8 × 2.99 = 23.92
		{"gpu_h200", "100", 34900},    // 100 × 3.49 = 349.00
	}
	for _, c := range cases {
		got, err := usdChargeCents(c.creditType, c.amount)
		if err != nil {
			t.Errorf("%s %s: unexpected error %v", c.creditType, c.amount, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s %s = %d cents, want %d", c.creditType, c.amount, got, c.want)
		}
	}
}

// TestUsdChargeCentsErrors checks unpriced types, bad amounts, and sub-minimum orders are rejected.
func TestUsdChargeCentsErrors(t *testing.T) {
	if _, err := usdChargeCents("not_a_type", "100"); err == nil {
		t.Error("an unpriced credit type must error")
	}
	if _, err := usdChargeCents("text", "0"); err == nil {
		t.Error("a non-positive amount must error")
	}
	if _, err := usdChargeCents("text", "abc"); err == nil {
		t.Error("a malformed amount must error")
	}
	if _, err := usdChargeCents("text", "1"); err == nil {
		t.Error("a sub-$0.50 order must error (1 × 0.00121 = 0.12c)")
	}
}
