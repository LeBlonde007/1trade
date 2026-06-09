package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// stripeAPIBase is Stripe's REST API origin. Test vs live mode is selected by the secret key prefix
// (sk_test_… / sk_live_…), not the URL.
const stripeAPIBase = "https://api.stripe.com"

// RealStripe creates real Stripe Checkout Sessions over the REST API (no SDK — a thin form-encoded
// POST + the same signed webhook the handler already verifies). The browser is redirected to the
// hosted session URL; settlement arrives asynchronously as a checkout.session.completed webhook,
// which books the credits. The secret key is held in memory only (injected from a k8s Secret).
type RealStripe struct {
	secretKey  string
	appBaseURL string
	apiBase    string
	http       *http.Client
}

// NewRealStripe builds a RealStripe from the Stripe secret key and the public app base URL (used for
// the post-payment success/cancel redirects).
func NewRealStripe(secretKey, appBaseURL string) *RealStripe {
	return &RealStripe{
		secretKey:  secretKey,
		appBaseURL: strings.TrimRight(appBaseURL, "/"),
		apiBase:    stripeAPIBase,
		http:       &http.Client{Timeout: 20 * time.Second},
	}
}

// CreateCheckoutSession opens a hosted Checkout Session for the pending purchase. The amount charged
// is computed SERVER-SIDE from the published per-credit reference price — the client never sets the
// money amount. The returned session ID is what the settlement webhook references back to the order.
func (c *RealStripe) CreateCheckoutSession(ctx context.Context, p CheckoutParams) (CheckoutSession, error) {
	if p.Currency != "usd" {
		return CheckoutSession{}, fmt.Errorf("card payments currently support USD only (got %q)", p.Currency)
	}
	cents, err := usdChargeCents(p.CreditType, p.Amount)
	if err != nil {
		return CheckoutSession{}, err
	}

	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("client_reference_id", p.PurchaseID)
	// {CHECKOUT_SESSION_ID} is expanded by Stripe on redirect — handy for the success screen.
	form.Set("success_url", c.appBaseURL+"/wallet?purchase=success&cs={CHECKOUT_SESSION_ID}")
	form.Set("cancel_url", c.appBaseURL+"/wallet/buy?canceled=1")
	form.Set("line_items[0][quantity]", "1")
	form.Set("line_items[0][price_data][currency]", "usd")
	form.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(cents, 10))
	form.Set("line_items[0][price_data][product_data][name]", productName(p))
	// Metadata ties the Stripe object back to our order for reconciliation/audit.
	form.Set("metadata[purchase_id]", p.PurchaseID)
	form.Set("metadata[tenant_id]", p.TenantID)
	form.Set("metadata[credit_type]", p.CreditType)
	form.Set("metadata[credits]", p.Amount)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase+"/v1/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return CheckoutSession{}, err
	}
	req.SetBasicAuth(c.secretKey, "") // Stripe auth: secret key as the basic-auth username, empty password
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Idempotency: a retried create for the same purchase returns the same session, never a duplicate.
	req.Header.Set("Idempotency-Key", "checkout_"+p.PurchaseID)

	resp, err := c.http.Do(req)
	if err != nil {
		return CheckoutSession{}, fmt.Errorf("stripe request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode/100 != 2 {
		return CheckoutSession{}, fmt.Errorf("stripe checkout create failed (%d): %s", resp.StatusCode, stripeErrMessage(body))
	}

	var out struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.ID == "" || out.URL == "" {
		return CheckoutSession{}, fmt.Errorf("stripe checkout: unexpected response")
	}
	return CheckoutSession{ID: out.ID, URL: out.URL}, nil
}

// productName is the human label shown on the Stripe checkout line item, e.g. "100,000 text credits".
func productName(p CheckoutParams) string {
	return fmt.Sprintf("%s %s credits", trimAmount(p.Amount), p.CreditType)
}

// trimAmount drops the fixed-point trailing zeros so the receipt reads "100000" not "100000.000000".
func trimAmount(a string) string {
	if strings.Contains(a, ".") {
		a = strings.TrimRight(a, "0")
		a = strings.TrimRight(a, ".")
	}
	return a
}

// stripeErrMessage pulls the human message out of a Stripe error body, falling back to the raw body.
func stripeErrMessage(body []byte) string {
	var e struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
		return e.Error.Message
	}
	return string(body)
}
