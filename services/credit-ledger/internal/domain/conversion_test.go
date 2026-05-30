package domain

import "testing"

// TestConvertedAmount checks the fixed-point conversion math: amount × rate × (1 − spread), floored.
func TestConvertedAmount(t *testing.T) {
	mk := func(s string) Money { m, _ := ParseMoney(s); return m }
	cases := []struct{ amount, rate, spread, want string }{
		{"100", "0.830579", "0.01", "82.227321"},  // AI-index → text, 1% spread
		{"100", "1.203980", "0.01", "119.194020"}, // text → AI-index
		{"100", "2", "0", "200.000000"},           // whole rate, no spread
		{"0.000001", "0.830579", "0", "0.000000"}, // sub-micro floors to zero — house never over-credits
	}
	for _, c := range cases {
		got, err := ConvertedAmount(mk(c.amount), c.rate, c.spread)
		if err != nil || got.String() != c.want {
			t.Fatalf("ConvertedAmount(%s,%s,%s) = %q,%v; want %q", c.amount, c.rate, c.spread, got.String(), err, c.want)
		}
	}
	if _, err := ConvertedAmount(mk("1"), "bad", "0.01"); err == nil {
		t.Fatal("expected error on invalid rate")
	}
	if _, err := ConvertedAmount(mk("1"), "1", "1.5"); err == nil {
		t.Fatal("expected error on spread >= 1")
	}
}
