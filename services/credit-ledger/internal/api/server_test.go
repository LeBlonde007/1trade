package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/trade1/credit-ledger/internal/api"
	"github.com/trade1/credit-ledger/internal/config"
	"github.com/trade1/credit-ledger/internal/events"
	"github.com/trade1/credit-ledger/internal/store"
	"github.com/google/uuid"
)

// TestAPIIntegration drives the real HTTP handlers against real Postgres via httptest (in-process,
// no ports/servers to juggle). Skips unless DATABASE_URL is set (schema must be applied). Verifies
// the purchase→debit→idempotent-replay→balances→chain-verify→insufficient flow end to end.
func TestAPIIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping API integration test")
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	srv := httptest.NewServer(api.New(config.Config{Env: "dev"}, st, events.LogPublisher{}))
	defer srv.Close()

	tenant := uuid.NewString()
	dev := map[string]string{"X-Dev-Tenant": tenant}

	// helper: do a request, decode JSON body into out, return status.
	do := func(method, path string, body any, out any) int {
		var rdr *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rdr = bytes.NewReader(b)
		} else {
			rdr = bytes.NewReader(nil)
		}
		req, _ := http.NewRequest(method, srv.URL+path, rdr)
		for k, v := range dev {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		defer resp.Body.Close()
		if out != nil {
			_ = json.NewDecoder(resp.Body).Decode(out)
		}
		return resp.StatusCode
	}

	// purchase +250 text
	var buy map[string]any
	if code := do("POST", "/v1/credits/purchase",
		map[string]any{"tenant_id": tenant, "credit_type": "text", "amount": "250", "reference_id": "s1", "is_paper": true}, &buy); code != 200 {
		t.Fatalf("purchase status %d (%v)", code, buy)
	}
	if buy["balance_after"] != "250.000000" {
		t.Fatalf("purchase balance_after = %v, want 250.000000", buy["balance_after"])
	}

	// debit -10.25 (idempotency anchor = usage_event_id u1)
	var deb map[string]any
	if code := do("POST", "/v1/credits/debit",
		map[string]any{"tenant_id": tenant, "credit_type": "text", "amount": "10.25", "usage_event_id": "u1", "is_paper": true}, &deb); code != 200 {
		t.Fatalf("debit status %d (%v)", code, deb)
	}
	if deb["balance_after"] != "239.750000" {
		t.Fatalf("debit balance_after = %v, want 239.750000", deb["balance_after"])
	}

	// idempotent replay → same tx_id, no double debit
	var deb2 map[string]any
	do("POST", "/v1/credits/debit",
		map[string]any{"tenant_id": tenant, "credit_type": "text", "amount": "10.25", "usage_event_id": "u1", "is_paper": true}, &deb2)
	if deb2["tx_id"] != deb["tx_id"] {
		t.Fatalf("idempotent replay made a new tx (%v != %v)", deb2["tx_id"], deb["tx_id"])
	}

	// balances → exactly one row at 239.75
	var bals struct {
		Balances []map[string]any `json:"balances"`
	}
	do("GET", "/v1/credits/balances?is_paper=true", nil, &bals)
	if len(bals.Balances) != 1 || bals.Balances[0]["balance"] != "239.750000" {
		t.Fatalf("balances = %v, want one row at 239.750000", bals.Balances)
	}

	// chain-verify → ok
	var cv map[string]any
	do("GET", "/v1/credits/audit/chain-verify?tenant_id="+tenant, nil, &cv)
	if cv["ok"] != true {
		t.Fatalf("chain-verify ok = %v, want true (%v)", cv["ok"], cv)
	}

	// insufficient credit → 402
	if code := do("POST", "/v1/credits/debit",
		map[string]any{"tenant_id": tenant, "credit_type": "text", "amount": "9999", "usage_event_id": "u2", "is_paper": true}, nil); code != http.StatusPaymentRequired {
		t.Fatalf("over-debit status = %d, want 402", code)
	}
}
