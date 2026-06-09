package billing

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestRealStripeCreateCheckout drives RealStripe against a fake Stripe API and asserts it sends the
// right auth, server-priced amount, metadata, and idempotency key, and parses the session back.
func TestRealStripeCreateCheckout(t *testing.T) {
	var gotForm url.Values
	var gotAuthUser, gotIdem string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/checkout/sessions" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		gotAuthUser, _, _ = r.BasicAuth()
		gotIdem = r.Header.Get("Idempotency-Key")
		body, _ := io.ReadAll(r.Body)
		gotForm, _ = url.ParseQuery(string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"cs_test_abc","url":"https://checkout.stripe.com/c/pay/cs_test_abc"}`)
	}))
	defer srv.Close()

	c := &RealStripe{secretKey: "sk_test_x", appBaseURL: "https://app.example", apiBase: srv.URL, http: srv.Client()}
	sess, err := c.CreateCheckoutSession(context.Background(), CheckoutParams{
		PurchaseID: "pur-1", TenantID: "ten-1", Amount: "100000", CreditType: "text", Currency: "usd",
	})
	if err != nil {
		t.Fatalf("CreateCheckoutSession: %v", err)
	}
	if sess.ID != "cs_test_abc" || !strings.Contains(sess.URL, "cs_test_abc") {
		t.Fatalf("session = %+v", sess)
	}
	if gotAuthUser != "sk_test_x" {
		t.Errorf("auth user = %q, want the secret key", gotAuthUser)
	}
	if gotIdem != "checkout_pur-1" {
		t.Errorf("idempotency key = %q", gotIdem)
	}
	if got := gotForm.Get("line_items[0][price_data][unit_amount]"); got != "12100" {
		t.Errorf("unit_amount = %q cents, want 12100 (server-priced)", got)
	}
	if got := gotForm.Get("line_items[0][price_data][currency]"); got != "usd" {
		t.Errorf("currency = %q", got)
	}
	if got := gotForm.Get("metadata[purchase_id]"); got != "pur-1" {
		t.Errorf("metadata purchase_id = %q", got)
	}
	if got := gotForm.Get("success_url"); !strings.Contains(got, "CHECKOUT_SESSION_ID") {
		t.Errorf("success_url missing session-id template: %q", got)
	}
}

// TestRealStripeRejectsNonUSD checks a non-USD currency fails before any network call.
func TestRealStripeRejectsNonUSD(t *testing.T) {
	c := NewRealStripe("sk_test_x", "https://app.example")
	if _, err := c.CreateCheckoutSession(context.Background(), CheckoutParams{
		PurchaseID: "p", TenantID: "t", Amount: "100000", CreditType: "text", Currency: "jpy",
	}); err == nil {
		t.Fatal("non-USD currency must be rejected")
	}
}

// TestRealStripeSurfacesAPIError checks a Stripe 4xx is surfaced with its message.
func TestRealStripeSurfacesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":{"message":"Invalid API Key provided"}}`)
	}))
	defer srv.Close()
	c := &RealStripe{secretKey: "sk_test_bad", appBaseURL: "https://app.example", apiBase: srv.URL, http: srv.Client()}
	_, err := c.CreateCheckoutSession(context.Background(), CheckoutParams{
		PurchaseID: "p", TenantID: "t", Amount: "100000", CreditType: "text", Currency: "usd",
	})
	if err == nil || !strings.Contains(err.Error(), "Invalid API Key") {
		t.Fatalf("want surfaced stripe error, got %v", err)
	}
}
