package events

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	yaml "go.yaml.in/yaml/v2"

	"github.com/trade1/matching-engine/internal/engine"
)

// loadSchema reads payload_schema from a contract file, normalised to JSON-shaped maps.
func loadSchema(t *testing.T, file string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile("../../../../docs/contracts/events/" + file)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[any]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return normalise(doc["payload_schema"]).(map[string]any)
}

// normalise converts yaml.v2's map[any]any tree into map[string]any.
func normalise(v any) any {
	switch x := v.(type) {
	case map[any]any:
		m := map[string]any{}
		for k, val := range x {
			m[fmt.Sprint(k)] = normalise(val)
		}
		return m
	case []any:
		for i := range x {
			x[i] = normalise(x[i])
		}
		return x
	}
	return v
}

var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// validate checks value against the subset of JSON Schema the event contracts use: type (incl.
// [T, "null"]), enum, required, properties, additionalProperties:false, $ref to definitions, and the
// uuid / date-time formats.
func validate(t *testing.T, root, schema map[string]any, value any, path string) {
	t.Helper()
	if ref, ok := schema["$ref"].(string); ok {
		name := ref[len("#/definitions/"):]
		validate(t, root, root["definitions"].(map[string]any)[name].(map[string]any), value, path)
		return
	}
	var types []string
	switch ty := schema["type"].(type) {
	case string:
		types = []string{ty}
	case []any:
		for _, x := range ty {
			types = append(types, x.(string))
		}
	}
	if value == nil {
		for _, ty := range types {
			if ty == "null" {
				return
			}
		}
		t.Errorf("%s: null not allowed (types %v)", path, types)
		return
	}
	kind := map[bool]string{}[false]
	switch value.(type) {
	case string:
		kind = "string"
	case bool:
		kind = "boolean"
	case float64:
		kind = "integer"
	case map[string]any:
		kind = "object"
	}
	okType := len(types) == 0
	for _, ty := range types {
		okType = okType || ty == kind
	}
	if !okType {
		t.Errorf("%s: got %s, want %v", path, kind, types)
		return
	}
	if enum, ok := schema["enum"].([]any); ok {
		found := false
		for _, e := range enum {
			found = found || e == value
		}
		if !found {
			t.Errorf("%s: %v not in enum %v", path, value, enum)
		}
	}
	if s, ok := value.(string); ok {
		switch schema["format"] {
		case "uuid":
			if !uuidRE.MatchString(s) {
				t.Errorf("%s: %q is not a uuid", path, s)
			}
		case "date-time":
			if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
				t.Errorf("%s: %q is not RFC 3339", path, s)
			}
		}
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return
	}
	props, _ := schema["properties"].(map[string]any)
	for _, r := range schema["required"].([]any) {
		if _, ok := obj[r.(string)]; !ok {
			t.Errorf("%s: missing required %q", path, r)
		}
	}
	for k, v := range obj {
		p, ok := props[k]
		if !ok {
			if schema["additionalProperties"] == false {
				t.Errorf("%s: property %q not in contract", path, k)
			}
			continue
		}
		validate(t, root, p.(map[string]any), v, path+"."+k)
	}
}

// uid makes a deterministic uuid-shaped id for test tenants and orders.
func uid(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }

// sampleEvents drives an engine through rests, partial and full fills, a market order, a cancel, a
// sub-account and a self-trade cancel, and returns every event.
func sampleEvents(t *testing.T) []engine.Event {
	t.Helper()
	e := engine.New(engine.Config{})
	ts := time.Date(2026, 9, 27, 12, 0, 0, 123456789, time.UTC)
	var all []engine.Event
	submit := func(c engine.SubmitCmd) {
		c.ProductID, c.IsPaper, c.TS = "H100-SPOT", true, ts
		ts = ts.Add(time.Millisecond)
		r, err := e.Submit(c)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, r.Events...)
	}
	fx := engine.MustFixed
	submit(engine.SubmitCmd{OrderID: uid(1), TenantID: uid(101), SubAccountID: uid(201), Side: engine.Sell, Type: engine.Limit, Price: fx("2.99"), Quantity: fx("10")})
	submit(engine.SubmitCmd{OrderID: uid(2), TenantID: uid(102), Side: engine.Sell, Type: engine.Limit, Price: fx("3.00"), Quantity: fx("10")})
	submit(engine.SubmitCmd{OrderID: uid(3), TenantID: uid(103), Side: engine.Buy, Type: engine.Market, Quantity: fx("15")})
	submit(engine.SubmitCmd{OrderID: uid(4), TenantID: uid(102), Side: engine.Buy, Type: engine.Limit, Price: fx("3.00"), Quantity: fx("1")})
	r, err := e.Cancel(engine.CancelCmd{OrderID: uid(2), TenantID: uid(102), TS: ts})
	if err != nil {
		t.Fatal(err)
	}
	return append(all, r.Events...)
}

// TestPayloadsMatchContracts validates every encoded event against its contract schema.
func TestPayloadsMatchContracts(t *testing.T) {
	schemas := map[string]map[string]any{
		SubjectOrdersState:    loadSchema(t, "orders.state.v1.yaml"),
		SubjectTradesExecuted: loadSchema(t, "trades.executed.v1.yaml"),
	}
	seen := map[string]int{}
	for i, ev := range sampleEvents(t) {
		subj, raw, err := Encode(ev)
		if err != nil {
			t.Fatal(err)
		}
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatal(err)
		}
		validate(t, schemas[subj], schemas[subj], v, fmt.Sprintf("%s[%d]", subj, i))
		seen[subj]++
	}
	if seen[SubjectTradesExecuted] < 2 || seen[SubjectOrdersState] < 6 {
		t.Fatalf("sample too thin to prove anything: %v", seen)
	}
}

// TestTradeChainVerifiableFromPayloads re-verifies the trade chain using only decoded payloads and the
// construction documented in SPEC.md §4 — what the ledger or surveillance must be able to do.
func TestTradeChainVerifiableFromPayloads(t *testing.T) {
	prev := ""
	n := 0
	for _, ev := range sampleEvents(t) {
		if ev.Trade == nil {
			continue
		}
		raw, err := TradeExecuted(*ev.Trade)
		if err != nil {
			t.Fatal(err)
		}
		var p map[string]any
		if err := json.Unmarshal(raw, &p); err != nil {
			t.Fatal(err)
		}
		if got := p["prev_chain_hash"]; (prev == "" && got != nil) || (prev != "" && got != prev) {
			t.Fatalf("prev_chain_hash = %v, want %q", got, prev)
		}
		sum := sha256.Sum256([]byte(prev + canonicalFromPayload(p)))
		if hex.EncodeToString(sum[:]) != p["chain_hash"] {
			t.Fatalf("chain_hash does not recompute from the payload for trade %v", p["trade_id"])
		}
		prev = p["chain_hash"].(string)
		n++
	}
	if n < 2 {
		t.Fatalf("only %d trades; need a chain of at least 2", n)
	}
}

// canonicalFromPayload rebuilds the hashed row from a decoded payload, in contract key order,
// without chain_hash / prev_chain_hash.
func canonicalFromPayload(p map[string]any) string {
	str := func(v any) string {
		if v == nil {
			return "null"
		}
		return strconv.Quote(v.(string))
	}
	party := func(v any) string {
		m := v.(map[string]any)
		return `{"tenant_id":` + str(m["tenant_id"]) + `,"sub_account_id":` + str(m["sub_account_id"]) +
			`,"order_id":` + str(m["order_id"]) + `,"fee":` + str(m["fee"]) + `,"liquidity":` + str(m["liquidity"]) +
			`,"is_internal":` + strconv.FormatBool(m["is_internal"].(bool)) + `}`
	}
	return `{"trade_id":` + str(p["trade_id"]) + `,"product_id":` + str(p["product_id"]) +
		`,"credit_type":` + str(p["credit_type"]) + `,"price":` + str(p["price"]) + `,"quantity":` + str(p["quantity"]) +
		`,"buyer":` + party(p["buyer"]) + `,"seller":` + party(p["seller"]) +
		`,"aggressor_side":` + str(p["aggressor_side"]) + `,"is_paper":` + strconv.FormatBool(p["is_paper"].(bool)) +
		`,"executed_at":` + str(p["executed_at"]) + `}`
}
