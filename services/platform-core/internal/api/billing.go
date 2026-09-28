package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/trade1/platform-core/internal/billing"
	"github.com/trade1/platform-core/internal/domain"
	"github.com/trade1/platform-core/internal/store"
	"github.com/google/uuid"
)

// checkoutBody is the POST /v1/billing/checkout request.
type checkoutBody struct {
	Amount     string `json:"amount"`
	CreditType string `json:"credit_type"`
	Currency   string `json:"currency"` // usd only (defaults to usd)
	Method     string `json:"method"`   // card (default) | ach
}

// createCheckout validates an order, records a pending purchase, creates a Stripe checkout session,
// and returns its URL. Credits are minted only after settlement (the webhook books them).
func (s *Server) createCheckout(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	var b checkoutBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	amt, ok := new(big.Rat).SetString(b.Amount)
	if !ok || amt.Sign() <= 0 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_amount", "amount must be a positive decimal")
		return
	}
	if b.Currency == "" {
		b.Currency = "usd"
	}
	if b.Method == "" {
		b.Method = "card"
	}
	if b.CreditType == "" || b.Currency != "usd" {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "credit_type is required; payments are in US dollars only (currency usd)")
		return
	}
	if b.Method != "card" && b.Method != "ach" {
		writeErr(w, http.StatusUnprocessableEntity, "bad_method", "method must be card or ach (for a wire transfer use /v1/billing/wires)")
		return
	}
	if !s.realMoneyAllowed(w, r, p) {
		return
	}

	purchaseID := uuid.NewString()
	session, err := s.stripe.CreateCheckoutSession(r.Context(), billing.CheckoutParams{
		PurchaseID: purchaseID, TenantID: p.TenantID, Amount: b.Amount, CreditType: b.CreditType, Currency: b.Currency, Method: b.Method,
	})
	if err != nil {
		serverError(w, err)
		return
	}
	// Record the price this purchase happened at. Best-effort: an unpriced credit type must not
	// block a checkout that Stripe already accepted — the columns are nullable precisely so a
	// missing price is visible as missing rather than guessed at later.
	unitPrice, cents, priceErr := billing.QuoteUSD(b.CreditType, b.Amount)
	if priceErr != nil {
		slog.Warn("purchase price not recorded", "purchase_id", purchaseID, "credit_type", b.CreditType, "err", priceErr)
	}
	if err := s.st.CreatePurchase(r.Context(), store.Purchase{
		ID: purchaseID, TenantID: p.TenantID, Amount: b.Amount, CreditType: b.CreditType, Currency: b.Currency, IsPaper: p.IsPaper,
		UnitPriceUSD: unitPrice, ChargedUSDCents: cents, Method: b.Method,
	}, session.ID); err != nil {
		serverError(w, err)
		return
	}
	slog.Info("audit: checkout created", "tenant_id", p.TenantID, "purchase_id", purchaseID, "amount", b.Amount, "credit_type", b.CreditType)

	// Dev/sandbox: MockStripe has no hosted checkout page and no webhook ever fires, so settle the
	// purchase inline (mark paid + book the credits) exactly as the webhook would. Real Stripe
	// deployments use a different StripeClient, so this never runs in prod — there, the signed
	// checkout.session.completed webhook books the credits. Idempotent on a synthetic event id.
	// An ACH debit never settles at checkout, even in dev: it waits for the bank (a signed
	// async_payment_succeeded webhook), which is the behaviour worth exercising.
	settled := false
	if _, isMock := s.stripe.(billing.MockStripe); s.cfg.BillingAutoSettle && isMock && b.Method == "card" {
		evID := "evt_mock_" + purchaseID
		if err := s.st.MarkPurchasePaid(r.Context(), session.ID, evID); err != nil {
			serverError(w, err)
			return
		}
		if err := s.booker.BookPurchase(r.Context(), billing.PurchaseBooking{
			TenantID: p.TenantID, Amount: b.Amount, CreditType: b.CreditType, IsPaper: p.IsPaper,
			ReferenceID: purchaseID, IdempotencyKey: bookingKey(purchaseID),
		}); err != nil {
			serverError(w, err)
			return
		}
		settled = true
		slog.Info("audit: mock checkout settled instantly (dev)", "tenant_id", p.TenantID, "purchase_id", purchaseID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"purchase_id": purchaseID, "checkout_url": session.URL, "settled": settled, "method": b.Method})
}

// stripeEvent is the slice of a Stripe Event payload we act on.
type stripeEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object struct {
			ID            string `json:"id"`
			PaymentStatus string `json:"payment_status"` // paid | unpaid (ACH still settling) | no_payment_required
		} `json:"object"`
	} `json:"data"`
}

// bookingKey is the ledger idempotency key for a purchase: credits are booked at most once per
// purchase, whichever event (checkout completed, or ACH settled) or retry triggers it.
func bookingKey(purchaseID string) string { return "purchase:" + purchaseID }

// stripeWebhook receives Stripe events. Authenticity is the Stripe-Signature header (HMAC), never a
// bearer token. Credits are booked only when money has settled:
//   - checkout.session.completed with payment_status=paid (card) → paid, book;
//   - checkout.session.completed with payment_status=unpaid (ACH debit started) → processing;
//   - checkout.session.async_payment_succeeded (ACH cleared) → paid, book;
//   - checkout.session.async_payment_failed (ACH returned) → failed, nothing booked.
//
// Booking is idempotent on the purchase, so a replay or a retry never double-mints.
func (s *Server) stripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "could not read body")
		return
	}
	if err := domain.VerifyStripeSignature(payload, r.Header.Get("Stripe-Signature"),
		s.cfg.StripeWebhookSecret, domain.StripeSignatureTolerance, time.Now()); err != nil {
		slog.Warn("stripe webhook signature rejected", "err", err)
		writeErr(w, http.StatusUnauthorized, "invalid_signature", "signature verification failed")
		return
	}
	var ev stripeEvent
	if err := json.Unmarshal(payload, &ev); err != nil || ev.ID == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid event payload")
		return
	}
	session := ev.Data.Object.ID
	settle := false
	switch ev.Type {
	case "checkout.session.completed":
		switch ev.Data.Object.PaymentStatus {
		case "paid":
			settle = true
		case "unpaid":
			if err := s.st.MarkPurchaseProcessing(r.Context(), session); err != nil {
				serverError(w, err)
				return
			}
			slog.Info("audit: purchase processing (bank debit settling)", "session", session, "event_id", ev.ID)
		}
	case "checkout.session.async_payment_succeeded":
		settle = true
	case "checkout.session.async_payment_failed":
		if err := s.st.MarkPurchaseFailed(r.Context(), session, "bank debit failed"); err != nil {
			serverError(w, err)
			return
		}
		slog.Warn("audit: purchase failed (bank debit returned)", "session", session, "event_id", ev.ID)
	}
	if !settle {
		w.WriteHeader(http.StatusOK) // nothing to book — ack so Stripe stops retrying
		return
	}
	if err := s.st.MarkPurchasePaid(r.Context(), session, ev.ID); err != nil {
		serverError(w, err)
		return
	}
	pur, found, err := s.st.GetPurchaseBySession(r.Context(), session)
	if err != nil {
		serverError(w, err)
		return
	}
	if !found || pur.Status != "paid" {
		w.WriteHeader(http.StatusOK) // unknown session, or a failed purchase that must not be revived
		return
	}
	// A failure returns 500 → Stripe retries; the booking key makes the retry safe.
	if err := s.booker.BookPurchase(r.Context(), billing.PurchaseBooking{
		TenantID: pur.TenantID, Amount: pur.Amount, CreditType: pur.CreditType, IsPaper: pur.IsPaper,
		ReferenceID: pur.ID, IdempotencyKey: bookingKey(pur.ID),
	}); err != nil {
		serverError(w, err)
		return
	}
	slog.Info("audit: purchase booked", "tenant_id", pur.TenantID, "purchase_id", pur.ID, "event_id", ev.ID, "amount", pur.Amount)
	w.WriteHeader(http.StatusOK)
}

// realMoneyAllowed is the KYC gate (F22): a real-money purchase (is_paper=false) requires a verified
// tenant. Sandbox/paper flows carry no real-money/AML exposure and are exempt. Server-side and
// authoritative — the web app's gate is convenience only. ok=false means a 403 was written.
func (s *Server) realMoneyAllowed(w http.ResponseWriter, r *http.Request, p principal) bool {
	if p.IsPaper {
		return true
	}
	kyc, found, err := s.st.GetKYC(r.Context(), p.TenantID)
	if err != nil {
		serverError(w, err)
		return false
	}
	if found && domain.CanPurchaseRealMoney(kyc.Status) {
		return true
	}
	status := "unverified"
	if found {
		status = string(kyc.Status)
	}
	slog.Warn("audit: real-money purchase blocked — kyc not verified", "tenant_id", p.TenantID, "kyc_status", status)
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: p.TenantID, ActorID: p.UserID, Action: "billing.checkout.blocked",
		TargetType: "tenant", TargetID: p.TenantID,
		After: map[string]any{"reason": "kyc_required", "kyc_status": status}, IsPaper: p.IsPaper,
	})
	writeErr(w, http.StatusForbidden, "kyc_required", "identity verification is required before real-money purchases")
	return false
}

// listPurchases returns the tenant's purchase history.
func (s *Server) listPurchases(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 200 {
			limit = n
		}
	}
	purs, err := s.st.ListPurchases(r.Context(), p.TenantID, limit)
	if err != nil {
		serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(purs))
	for _, pu := range purs {
		row := map[string]any{
			"id": pu.ID, "amount": pu.Amount, "credit_type": pu.CreditType,
			"currency": pu.Currency, "status": pu.Status, "created_at": pu.CreatedAt.UTC().Format(time.RFC3339),
			"method": pu.Method,
		}
		if pu.ChargedUSDCents > 0 {
			row["amount_usd"] = centsToUSD(pu.ChargedUSDCents)
		}
		if pu.WireReference != "" {
			row["wire_reference"] = pu.WireReference
		}
		if pu.FailureReason != "" {
			row["failure_reason"] = pu.FailureReason
		}
		if pu.PaidAt != nil {
			row["paid_at"] = pu.PaidAt.UTC().Format(time.RFC3339)
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"purchases": out})
}
