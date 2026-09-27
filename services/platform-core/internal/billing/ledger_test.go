package billing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGrantPaperCashRequest pins the wire shape of the paper-cash grant against credit.yaml v1.1:
// path, service bearer, Idempotency-Key, and a PaperCashGrantRequest body that is always paper USD.
func TestGrantPaperCashRequest(t *testing.T) {
	var got struct {
		path, auth, key string
		body            map[string]any
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path, got.auth, got.key = r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Idempotency-Key")
		_ = json.NewDecoder(r.Body).Decode(&got.body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &LedgerClient{url: srv.URL, serviceToken: "svc", http: srv.Client()}
	if err := c.GrantPaperCash(context.Background(), "t-1", "10000.000000", "paper-grant:t-1"); err != nil {
		t.Fatal(err)
	}
	if got.path != "/v1/credits/paper-cash/grant" || got.auth != "Bearer svc" || got.key != "paper-grant:t-1" {
		t.Errorf("request = %s auth=%q key=%q", got.path, got.auth, got.key)
	}
	want := map[string]any{"tenant_id": "t-1", "currency": "USD", "amount": "10000.000000", "is_paper": true}
	for k, v := range want {
		if got.body[k] != v {
			t.Errorf("body[%s] = %v, want %v", k, got.body[k], v)
		}
	}
	if len(got.body) != len(want) {
		t.Errorf("body has extra fields: %v", got.body)
	}
}

// TestGrantPaperCashSurfacesLedgerErrors checks a non-200 from the ledger is an error, not silence.
func TestGrantPaperCashSurfacesLedgerErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer srv.Close()
	c := &LedgerClient{url: srv.URL, serviceToken: "svc", http: srv.Client()}
	if err := c.GrantPaperCash(context.Background(), "t-1", "1", "k"); err == nil {
		t.Fatal("422 from the ledger was not reported")
	}
}
