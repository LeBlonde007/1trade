// Package billing orchestrates credit purchases: it creates Stripe checkout sessions and books
// settled payments into the credit ledger. Both sit behind interfaces so the whole flow runs locally
// on a mock with no Stripe keys; real Stripe swaps in behind the same interface (F06 M3).
package billing

import "context"

// CheckoutParams describes a checkout session to create.
type CheckoutParams struct {
	PurchaseID string
	TenantID   string
	Amount     string
	CreditType string
	Currency   string
}

// CheckoutSession is a created checkout; its ID links the later webhook back to the order.
type CheckoutSession struct {
	ID  string
	URL string
}

// StripeClient creates checkout sessions. MockStripe (dev/tests) or a real Stripe client (prod).
type StripeClient interface {
	CreateCheckoutSession(ctx context.Context, p CheckoutParams) (CheckoutSession, error)
}

// mockCheckoutBase is the fake hosted-checkout origin the mock returns.
const mockCheckoutBase = "https://checkout.stripe.test/pay"

// MockStripe returns a deterministic session with no network call, so local dev and CI run without
// Stripe keys. In dev the webhook is simulated and verified by domain.VerifyStripeSignature.
type MockStripe struct{}

// CreateCheckoutSession returns a fake session derived from the purchase id.
func (MockStripe) CreateCheckoutSession(_ context.Context, p CheckoutParams) (CheckoutSession, error) {
	id := "cs_mock_" + p.PurchaseID
	return CheckoutSession{ID: id, URL: mockCheckoutBase + "/" + id}, nil
}
