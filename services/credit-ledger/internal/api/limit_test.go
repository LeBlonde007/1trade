package api

import "testing"

// TestParseTxLimit pins the listTransactions page-size rule from credit.yaml: default 50, 1–200 accepted,
// anything else rejected.
func TestParseTxLimit(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"", 50, true},
		{"1", 1, true},
		{"200", 200, true},
		{"0", 0, false},
		{"201", 0, false},
		{"-5", 0, false},
		{"abc", 0, false},
		{"10.5", 0, false},
	}
	for _, c := range cases {
		got, ok := parseTxLimit(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("parseTxLimit(%q) = (%d, %v), want (%d, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}
