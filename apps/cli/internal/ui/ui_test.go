package ui

import (
	"strings"
	"testing"
)

// TestDeltaNoColor checks the credit-movement glyphs + magnitude formatting with colour disabled (the
// piped/CI shape): ▲ for non-negative, ▼ for negative, trailing zeros trimmed.
func TestDeltaNoColor(t *testing.T) {
	defer SetColor(SetColor(false)) // force off, restore after
	cases := []struct {
		in   float64
		want string
	}{
		{12.5, "▲ 12.5"},
		{-3, "▼ 3"},
		{0, "▲ 0"},
		{5.250000, "▲ 5.25"},
	}
	for _, c := range cases {
		if got := Delta(c.in); got != c.want {
			t.Errorf("Delta(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestColorWrapsOnlyWhenOn verifies styling adds ANSI codes only with colour on, and is a pure no-op off.
func TestColorWrapsOnlyWhenOn(t *testing.T) {
	defer SetColor(SetColor(false))
	if Green("x") != "x" {
		t.Fatal("colour off must not add ANSI codes")
	}
	SetColor(true)
	if !strings.Contains(Green("x"), "\x1b[32m") {
		t.Fatal("colour on must add the green ANSI code")
	}
}

// TestHint maps upstream errors to a next step; unknown/2xx yields no hint.
func TestHint(t *testing.T) {
	if !strings.Contains(Hint(402, ""), "top up") {
		t.Error("402 should hint at topping up credits")
	}
	if !strings.Contains(Hint(401, ""), "login") {
		t.Error("401 should hint at login")
	}
	if !strings.Contains(Hint(0, "email_unverified"), "verify") {
		t.Error("email_unverified should hint at verifying")
	}
	if Hint(200, "") != "" {
		t.Error("a 2xx/unknown status should yield no hint")
	}
}
