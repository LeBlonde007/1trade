package catalog

import (
	"testing"
	"time"
)

var now = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

// schedule swaps in a deprecation table for one test.
func schedule(t *testing.T, ds map[string]Deprecation) {
	t.Helper()
	prev := deprecations
	deprecations = ds
	t.Cleanup(func() { deprecations = prev })
}

// TestEveryModelHasMetadata: every entry has a category (the console groups by it) and a known
// credit type, and the shipped deprecation schedule is valid.
func TestEveryModelHasMetadata(t *testing.T) {
	for _, m := range List() {
		if m.Trade1.Category == "" {
			t.Errorf("%s has no category", m.ID)
		}
		switch m.Trade1.CreditType {
		case "text", "speech", "image", "video", "embeddings":
		default:
			t.Errorf("%s bills unknown credit type %q", m.ID, m.Trade1.CreditType)
		}
	}
	if err := validateDeprecations(deprecations); err != nil {
		t.Fatal(err)
	}
}

// TestLifecycle walks one model through active → deprecated → retired.
func TestLifecycle(t *testing.T) {
	d := Deprecation{AnnouncedAt: now, SunsetAt: now.Add(MinNotice), Replacement: "gpt-5"}
	schedule(t, map[string]Deprecation{"gpt-4o": d})
	for _, c := range []struct {
		at   time.Time
		want string
	}{
		{now.Add(-time.Hour), StatusActive},
		{now, StatusDeprecated},
		{now.Add(MinNotice - time.Second), StatusDeprecated},
		{now.Add(MinNotice), StatusRetired},
	} {
		if got, _ := StatusAt("gpt-4o", c.at); got != c.want {
			t.Errorf("at %s: %s, want %s", c.at, got, c.want)
		}
	}
	listed := func(at time.Time) bool {
		for _, m := range Listing(nil, at) {
			if m.ID == "gpt-4o" {
				return true
			}
		}
		return false
	}
	if !listed(now.Add(time.Hour)) || listed(now.Add(MinNotice)) {
		t.Fatal("a deprecated model must stay listed and a retired one must not")
	}
	if m, _ := LookupAt("gpt-4o", now.Add(time.Hour)); m.Trade1.Status != StatusDeprecated || m.Trade1.Deprecation.Replacement != "gpt-5" {
		t.Fatalf("lookup = %+v", m.Trade1)
	}
	if m, _ := LookupAt("gpt-5", now); m.Trade1.Status != StatusActive || m.Trade1.Deprecation != nil {
		t.Fatalf("an unscheduled model is %+v", m.Trade1)
	}
}

// TestScheduleRules: 30 days' notice, known models, a live replacement.
func TestScheduleRules(t *testing.T) {
	for name, ds := range map[string]map[string]Deprecation{
		"short notice":        {"gpt-4o": {AnnouncedAt: now, SunsetAt: now.Add(29 * 24 * time.Hour)}},
		"unknown model":       {"nope": {AnnouncedAt: now, SunsetAt: now.Add(MinNotice)}},
		"unknown replacement": {"gpt-4o": {AnnouncedAt: now, SunsetAt: now.Add(MinNotice), Replacement: "nope"}},
		"replacement retiring": {
			"gpt-4o": {AnnouncedAt: now, SunsetAt: now.Add(MinNotice), Replacement: "gpt-5"},
			"gpt-5":  {AnnouncedAt: now, SunsetAt: now.Add(MinNotice)},
		},
	} {
		if validateDeprecations(ds) == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
