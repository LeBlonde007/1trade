package detect

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	yaml "go.yaml.in/yaml/v2"
)

// allRules returns alerts of every rule, produced by the real detectors.
func allRules(t *testing.T) []Alert {
	t.Helper()
	cfg := DefaultConfig()
	cfg.BeneficialOwner = map[string]string{"W1": "X", "W2": "X"}
	cfg.DefaultMaxPosition = 1000 * unit
	cfg.LayerMinOrders = 1000 // keep the churn below from also counting as layering
	d := New(cfg)
	evs := scenario() // round-trip wash + spoofing
	evs = append(evs, trade("w", "EAI-IDX", "W1", "W2", "buy", "1", "1", at(20*time.Second)))
	for i := range 25 {
		id := fmt.Sprintf("k%d", i)
		evs = append(evs, accept(id, "K", "TEXT-SPOT", "buy", "1", "0.0009", at(30*time.Second+time.Duration(i)*time.Second)),
			cancel(id, "K", "user_cancel", "0", at(30*time.Second+time.Duration(i)*time.Second+time.Millisecond)))
	}
	evs = append(evs, trade("big", "H200-SPOT", "P", "Z", "buy", "2000", "3.8", at(time.Minute)))
	evs = append(evs, trade("i1", "EAI-IDX", "C", "Z", "buy", "5000", "0.001", at(2*time.Minute)))
	for i, px := range []string{"1.000", "1.005", "1.010", "1.020"} {
		evs = append(evs, trade(fmt.Sprintf("p%d", i), "TEXT-SPOT", "C", "mm", "buy", "100", px, at(3*time.Minute+time.Duration(i)*time.Second)))
	}
	close := time.Date(2026, 9, 27, 15, 40, 0, 0, time.UTC)
	evs = append(evs, trade("c1", "VIDEO-SPOT", "M", "Q1", "buy", "3000", "0.25", close))
	got := run(d, evs)

	lay := New(DefaultConfig())
	levs := make([]event, 0, 9)
	for i, p := range []string{"0.001200", "0.001199", "0.001198", "0.001197"} {
		levs = append(levs, accept(fmt.Sprintf("L%d", i), "L", "TEXT-SPOT", "buy", "50", p, at(time.Duration(i)*time.Second)))
	}
	levs = append(levs, trade("s1", "TEXT-SPOT", "M", "L", "sell", "40", "0.00121", at(10*time.Second)))
	for i := range 4 {
		levs = append(levs, cancel(fmt.Sprintf("L%d", i), "L", "user_cancel", "0", at(20*time.Second)))
	}
	return append(got, run(lay, levs)...)
}

// TestAlertsMatchContract validates every rule's alert against surveillance.alert.v1: exactly the
// declared properties, every required one, enums, date-times, and nullable fields.
func TestAlertsMatchContract(t *testing.T) {
	raw, err := os.ReadFile("../../../../docs/contracts/events/surveillance.alert.v1.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	schema := doc["payload_schema"].(map[any]any)
	props := schema["properties"].(map[any]any)
	required := schema["required"].([]any)

	alerts := allRules(t)
	seen := map[string]bool{}
	for _, a := range alerts {
		seen[a.Rule] = true
		b, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		for k := range m {
			if _, ok := props[k]; !ok {
				t.Errorf("%s: property %q not in contract", a.Rule, k)
			}
		}
		for _, r := range required {
			if _, ok := m[r.(string)]; !ok {
				t.Errorf("%s: missing required %q", a.Rule, r)
			}
		}
		for _, f := range []string{"rule", "severity", "action"} {
			enum := props[f].(map[any]any)["enum"].([]any)
			ok := false
			for _, e := range enum {
				ok = ok || e == m[f]
			}
			if !ok {
				t.Errorf("%s: %s=%v not in %v", a.Rule, f, m[f], enum)
			}
		}
		for _, f := range []string{"window_start", "window_end", "detected_at"} {
			if _, err := time.Parse(time.RFC3339Nano, m[f].(string)); err != nil {
				t.Errorf("%s: %s=%v is not RFC 3339", a.Rule, f, m[f])
			}
		}
		if !regexp.MustCompile(`^sva_[0-9a-f]{24}$`).MatchString(a.AlertID) {
			t.Errorf("%s: alert_id %q", a.Rule, a.AlertID)
		}
		if a.Evidence == nil {
			t.Errorf("%s: no evidence", a.Rule)
		}
	}
	for _, r := range []string{RuleWash, RuleSpoofing, RuleLayering, RuleMarkClose, RuleCrossProduct, RuleExcessCancel, RulePosLimit} {
		if !seen[r] {
			t.Errorf("rule %s produced no alert to validate", r)
		}
	}
}
