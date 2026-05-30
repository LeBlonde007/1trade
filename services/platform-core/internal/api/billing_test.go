package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/exascale/platform-core/internal/api"
	"github.com/exascale/platform-core/internal/billing"
	"github.com/exascale/platform-core/internal/config"
	"github.com/exascale/platform-core/internal/domain"
	"github.com/exascale/platform-core/internal/store"
	"github.com/google/uuid"
)

const webhookSecret = "whsec_test_f06_abc"

// stubBooker records BookPurchase calls instead of hitting the ledger.
type stubBooker struct {
	mu    sync.Mutex
	calls []billing.PurchaseBooking
}

// BookPurchase records the booking.
func (b *stubBooker) BookPurchase(_ context.Context, pb billing.PurchaseBooking) error {
	b.mu.Lock()
	b.calls = append(b.calls, pb)
	b.mu.Unlock()
	return nil
}

// snapshot returns the recorded bookings.
func (b *stubBooker) snapshot() []billing.PurchaseBooking {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]billing.PurchaseBooking(nil), b.calls...)
}

// TestBillingCheckoutAndWebhook drives the money-in loop end to end against real Postgres with a
// fake Stripe + a stub booker: checkout → signed webhook → booked; replay carries the same
// idempotency key (so the ledger dedupes); a bad signature is 401; the purchase shows as paid.
func TestBillingCheckoutAndWebhook(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping billing integration test")
	}
	st, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	booker := &stubBooker{}
	cfg := config.Config{Env: "dev", JWTSecret: jwtSecret, StripeWebhookSecret: webhookSecret, TokenTTL: 3600_000_000_000}
	srv := httptest.NewServer(api.NewWithBilling(cfg, st, billing.MockStripe{}, booker))
	defer srv.Close()

	doJSON := func(method, path, bearer string, body, out any) int {
		var r *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			r = bytes.NewReader(b)
		} else {
			r = bytes.NewReader(nil)
		}
		req, _ := http.NewRequest(method, srv.URL+path, r)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
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
	postWebhook := func(payload []byte, sig string) int {
		req, _ := http.NewRequest("POST", srv.URL+"/v1/billing/webhook/stripe", bytes.NewReader(payload))
		req.Header.Set("Stripe-Signature", sig)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		return resp.StatusCode
	}

	// signup → JWT
	var signup map[string]any
	doJSON("POST", "/v1/auth/signup", "", map[string]string{
		"email": "buyer+" + uuid.NewString() + "@acme.ai", "password": "pw-123456", "tenant_name": "Acme",
	}, &signup)
	token, _ := signup["token"].(string)
	if token == "" {
		t.Fatal("no token from signup")
	}

	// checkout → pending purchase + mock session
	var co map[string]any
	if code := doJSON("POST", "/v1/billing/checkout", token,
		map[string]string{"amount": "500.000000", "credit_type": "text", "currency": "usd"}, &co); code != 200 {
		t.Fatalf("checkout status %d (%v)", code, co)
	}
	purchaseID, _ := co["purchase_id"].(string)
	if purchaseID == "" {
		t.Fatalf("no purchase_id: %v", co)
	}
	sessionID := "cs_mock_" + purchaseID

	// signed webhook → booked
	eventID := "evt_" + uuid.NewString()
	payload, _ := json.Marshal(map[string]any{
		"id": eventID, "type": "checkout.session.completed",
		"data": map[string]any{"object": map[string]any{"id": sessionID}},
	})
	sign := func(p []byte) string {
		ts := time.Now().Unix()
		return fmt.Sprintf("t=%d,v1=%s", ts, domain.SignStripePayload(p, ts, webhookSecret))
	}
	if code := postWebhook(payload, sign(payload)); code != 200 {
		t.Fatalf("webhook status %d", code)
	}
	calls := booker.snapshot()
	if len(calls) != 1 || calls[0].Amount != "500.000000" || calls[0].CreditType != "text" || calls[0].IdempotencyKey != eventID || calls[0].TenantID == "" {
		t.Fatalf("unexpected booking: %+v", calls)
	}

	// replay → still 200, same idempotency key (ledger dedupes → no double-mint)
	if code := postWebhook(payload, sign(payload)); code != 200 {
		t.Fatalf("replay webhook status %d", code)
	}
	calls = booker.snapshot()
	if len(calls) != 2 || calls[1].IdempotencyKey != eventID {
		t.Fatalf("replay should re-book under the same idempotency key: %+v", calls)
	}

	// bad signature → 401
	if code := postWebhook(payload, "t=1,v1=deadbeef"); code != 401 {
		t.Fatalf("bad-signature webhook = %d, want 401", code)
	}

	// purchases list shows it paid
	var pl struct {
		Purchases []map[string]any `json:"purchases"`
	}
	doJSON("GET", "/v1/billing/purchases", token, nil, &pl)
	if len(pl.Purchases) != 1 || pl.Purchases[0]["status"] != "paid" {
		t.Fatalf("purchase not paid in history: %+v", pl.Purchases)
	}
}
