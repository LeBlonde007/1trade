package settle

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

	yaml "go.yaml.in/yaml/v2"

	"github.com/trade1/matching-engine/internal/engine"
)

// uid returns a deterministic uuid-shaped id.
func uid(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }

// matchOne runs a real engine to produce one trade: seller rests 10 H100 @ 2.99, buyer lifts it.
func matchOne(t *testing.T, buyer, seller string) engine.Trade {
	t.Helper()
	e := engine.New(engine.Config{})
	ts := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	if _, err := e.Submit(engine.SubmitCmd{OrderID: "00000000-0000-4000-8003-" + seller[24:], TenantID: seller, ProductID: "H100-SPOT",
		Side: engine.Sell, Type: engine.Limit, Price: engine.MustFixed("2.99"), Quantity: engine.MustFixed("10"), IsPaper: true, TS: ts}); err != nil {
		t.Fatal(err)
	}
	r, err := e.Submit(engine.SubmitCmd{OrderID: "00000000-0000-4000-8004-" + buyer[24:], TenantID: buyer, ProductID: "H100-SPOT",
		Side: engine.Buy, Type: engine.Market, Quantity: engine.MustFixed("10"), IsPaper: true, TS: ts.Add(time.Millisecond)})
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range r.Events {
		if ev.Trade != nil {
			return *ev.Trade
		}
	}
	t.Fatal("no trade produced")
	return engine.Trade{}
}

// TestBodyMatchesContract pins the request body to SettleTradeRequest in credit.yaml: exactly the
// declared properties at both levels, every required one present, currency USD, money as strings.
func TestBodyMatchesContract(t *testing.T) {
	raw, err := os.ReadFile("../../../../docs/contracts/openapi/credit.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	schemas := doc["components"].(map[any]any)["schemas"].(map[any]any)
	keys := func(name string) (props, required []string) {
		s := schemas[name].(map[any]any)
		for k := range s["properties"].(map[any]any) {
			props = append(props, k.(string))
		}
		for _, r := range s["required"].([]any) {
			required = append(required, r.(string))
		}
		sort.Strings(props)
		return props, required
	}

	b, err := Body(matchOne(t, uid(101), uid(102)))
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(b, &body); err != nil {
		t.Fatal(err)
	}
	check := func(name string, obj map[string]any) {
		props, required := keys(name)
		var got []string
		for k := range obj {
			got = append(got, k)
		}
		sort.Strings(got)
		if fmt.Sprint(got) != fmt.Sprint(props) {
			t.Errorf("%s keys = %v, contract = %v", name, got, props)
		}
		for _, r := range required {
			if _, ok := obj[r]; !ok {
				t.Errorf("%s missing required %q", name, r)
			}
		}
	}
	check("SettleTradeRequest", body)
	check("SettleParty", body["buyer"].(map[string]any))
	check("SettleParty", body["seller"].(map[string]any))
	if body["currency"] != "USD" || body["price"] != "2.990000" || body["quantity"] != "10.000000" || body["is_paper"] != true {
		t.Errorf("body = %v", body)
	}
	if body["buyer"].(map[string]any)["sub_account_id"] != nil {
		t.Error("empty sub-account must be JSON null")
	}
}

// TestSettleOutcomes checks headers and the mapping of every ledger response to an outcome.
func TestSettleOutcomes(t *testing.T) {
	tr := matchOne(t, uid(201), uid(202))
	cases := []struct {
		status int
		code   string
		want   error // nil = success; errAny = transient
	}{
		{200, "", nil},
		{402, "INSUFFICIENT_CASH", ErrUnsettleable},
		{402, "INSUFFICIENT_CREDIT", ErrUnsettleable},
		{422, "REAL_MONEY_DISABLED", ErrUnsettleable},
		{409, "IDEMPOTENCY_CONFLICT", ErrConflict},
		{403, "forbidden", errAny},
		{500, "internal_error", errAny},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.status, tc.code), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/credits/settle-trade" || r.Header.Get("Authorization") != "Bearer engine-token" ||
					r.Header.Get("Idempotency-Key") != tr.TradeID {
					t.Errorf("request: %s auth=%q key=%q", r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Idempotency-Key"))
				}
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprintf(w, `{"code":%q}`, tc.code)
			}))
			defer srv.Close()
			err := New(srv.URL, "engine-token", time.Second).Settle(context.Background(), tr)
			switch {
			case tc.want == nil && err != nil:
				t.Fatalf("err = %v, want nil", err)
			case tc.want == errAny:
				if err == nil || errors.Is(err, ErrUnsettleable) || errors.Is(err, ErrConflict) {
					t.Fatalf("err = %v, want a transient error", err)
				}
			case tc.want != nil && !errors.Is(err, tc.want):
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

// errAny marks "any error that is neither ErrUnsettleable nor ErrConflict".
var errAny = errors.New("transient")

// TestSettleAgainstRealLedger is the cross-service proof: a trade the real engine produced is settled
// by the real credit-ledger, and balances move exactly. Runs only when LEDGER_E2E_URL,
// LEDGER_SERVICE_TOKEN and LEDGER_SETTLE_TOKEN point at a running ledger (dev mode).
func TestSettleAgainstRealLedger(t *testing.T) {
	base, svc, settleTok := os.Getenv("LEDGER_E2E_URL"), os.Getenv("LEDGER_SERVICE_TOKEN"), os.Getenv("LEDGER_SETTLE_TOKEN")
	if base == "" || svc == "" || settleTok == "" {
		t.Skip("LEDGER_E2E_URL / LEDGER_SERVICE_TOKEN / LEDGER_SETTLE_TOKEN not set")
	}
	stamp := time.Now().UnixNano() % 1_000_000_000_000
	buyer, seller := fmt.Sprintf("00000000-0000-4000-8001-%012d", stamp), fmt.Sprintf("00000000-0000-4000-8002-%012d", stamp)
	post := func(path, key string, body map[string]any) {
		b, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, base+path, bytes.NewReader(b))
		req.Header.Set("Authorization", "Bearer "+svc)
		req.Header.Set("Idempotency-Key", key)
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("%s: %v %v", path, err, resp)
		}
		resp.Body.Close()
	}
	post("/v1/credits/paper-cash/grant", "paper-grant:"+buyer, map[string]any{"tenant_id": buyer, "currency": "USD", "amount": "10000", "is_paper": true})
	post("/v1/credits/purchase", "seed:"+seller, map[string]any{"tenant_id": seller, "credit_type": "gpu_h100", "amount": "100", "reference_id": "seed:" + seller, "is_paper": true})

	tr := matchOne(t, buyer, seller)
	c := New(base, settleTok, 5*time.Second)
	if err := c.Settle(context.Background(), tr); err != nil {
		t.Fatalf("settle: %v", err)
	}
	if err := c.Settle(context.Background(), tr); err != nil {
		t.Fatalf("replayed settle: %v", err)
	}
	if err := New(base, svc, 5*time.Second).Settle(context.Background(), tr); err == nil {
		t.Fatal("settlement with the shared service token was accepted")
	}

	read := func(path, tenant string) map[string]string {
		req, _ := http.NewRequest(http.MethodGet, base+path, nil)
		req.Header.Set("X-Dev-Tenant", tenant)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out struct {
			Balances []map[string]any `json:"balances"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		m := map[string]string{}
		for _, b := range out.Balances {
			k, _ := b["credit_type"].(string)
			if k == "" {
				k, _ = b["currency"].(string)
			}
			m[k], _ = b["balance"].(string)
		}
		return m
	}
	// 10 @ 2.99 → notional 29.90; taker (buyer) fee 1% = 0.299; maker (seller) fee 0.5% = 0.1495.
	want := map[string]string{
		"buyer gpu_h100": "10.000000", "buyer USD": "9969.801000",
		"seller gpu_h100": "90.000000", "seller USD": "29.750500",
	}
	got := map[string]string{
		"buyer gpu_h100": read("/v1/credits/balances", buyer)["gpu_h100"], "buyer USD": read("/v1/credits/cash/balances", buyer)["USD"],
		"seller gpu_h100": read("/v1/credits/balances", seller)["gpu_h100"], "seller USD": read("/v1/credits/cash/balances", seller)["USD"],
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %s, want %s", k, got[k], v)
		}
	}
}
