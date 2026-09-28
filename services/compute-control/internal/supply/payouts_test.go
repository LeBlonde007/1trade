package supply

import (
	"testing"
	"time"
)

// TestStatementMath pins the formula: gross = Σ hours × rate; fee and holdback truncate to 6 places;
// payout = gross − fee; released = payout − holdback; unpriced tiers stay unpaid.
func TestStatementMath(t *testing.T) {
	ag := &Agreement{Rates: map[string]string{"gpu_h100": "1.800000"}, FeePercent: "10", HoldbackPercent: "15", DisputeDays: 14}
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	p, ids, err := statement(ag, "p", true, []group{
		{gpuType: "gpu_h100", seconds: "5400.000000", usageIDs: []string{"a", "b"}}, // 1.5 h
		{gpuType: "gpu_h200", seconds: "3600.000000", usageIDs: []string{"c"}},      // no rate: unpaid
	}, now.Add(-30*24*time.Hour), now, now)
	if err != nil {
		t.Fatal(err)
	}
	// 1.5 × 1.8 = 2.7; fee 0.27; payout 2.43; holdback 0.3645; released 2.0655.
	got := []string{p.Gross, p.Fee, p.Amount, p.Holdback, p.Released, p.GPUSeconds}
	want := []string{"2.700000", "0.270000", "2.430000", "0.364500", "2.065500", "5400.000000"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("field %d = %s, want %s (all: %v)", i, got[i], want[i], got)
		}
	}
	if len(ids) != 2 || p.UsageRecords != 2 || len(p.Lines) != 1 || p.Lines[0].GPUHours != "1.500000" {
		t.Fatalf("covered %v, lines %+v", ids, p.Lines)
	}
	if !p.DisputeUntil.Equal(now.Add(14 * 24 * time.Hour)) {
		t.Fatalf("dispute window ends %s", p.DisputeUntil)
	}
	// Truncation: 1 s at 1.800000/h = 0.0005 exactly; 7 s = 0.0035; fee 33.333333% of 0.0035 → 0.001166.
	ag2 := &Agreement{Rates: map[string]string{"gpu_h100": "1.800000"}, FeePercent: "33.333333", HoldbackPercent: "0", DisputeDays: 1}
	p2, _, _ := statement(ag2, "p", true, []group{{gpuType: "gpu_h100", seconds: "7.000000", usageIDs: []string{"x"}}}, now, now, now)
	if p2.Gross != "0.003500" || p2.Fee != "0.001166" || p2.Amount != "0.002334" || p2.Released != "0.002334" {
		t.Fatalf("truncation: %+v", p2)
	}
}

// TestAgreementValidation refuses bad terms.
func TestAgreementValidation(t *testing.T) {
	ok := Agreement{Rates: map[string]string{"gpu_h100": "1.5"}, FeePercent: "10", HoldbackPercent: "15", DisputeDays: 14}
	if validAgreement(ok) != nil {
		t.Fatal("valid terms refused")
	}
	for name, mut := range map[string]func(*Agreement){
		"no rates":      func(a *Agreement) { a.Rates = nil },
		"unknown tier":  func(a *Agreement) { a.Rates = map[string]string{"gpu_a100": "1"} },
		"negative rate": func(a *Agreement) { a.Rates = map[string]string{"gpu_h100": "-1"} },
		"7 decimals":    func(a *Agreement) { a.Rates = map[string]string{"gpu_h100": "1.0000001"} },
		"fee > 100":     func(a *Agreement) { a.FeePercent = "100.5" },
		"exponent":      func(a *Agreement) { a.HoldbackPercent = "1e1" },
		"window":        func(a *Agreement) { a.DisputeDays = 0 },
	} {
		a := ok
		a.Rates = map[string]string{"gpu_h100": "1.5"}
		mut(&a)
		if validAgreement(a) == nil {
			t.Errorf("%s accepted", name)
		}
	}
}
