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
	Currency   string `json:"currency"`
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
	if b.CreditType == "" || (b.Currency != "usd" && b.Currency != "jpy") {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "credit_type and a supported currency (usd|jpy) are required")
		return
	}

	// KYC gate (F22): a real-money purchase (is_paper=false) requires a verified tenant. Sandbox/paper
	// flows carry no real-money/AML exposure and are exempt. This server-side check is authoritative —
	// the web app's gate is convenience only and is never trusted.
	if !p.IsPaper {
		kyc, found, err := s.st.GetKYC(r.Context(), p.TenantID)
		if err != nil {
			serverError(w, err)
			return
		}
		if !found || !domain.CanPurchaseRealMoney(kyc.Status) {
			status := "unverified"
			if found {
				status = string(kyc.Status)
			}
			slog.Warn("audit: real-money checkout blocked — kyc not verified", "tenant_id", p.TenantID, "kyc_status", status)
			_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
				TenantID: p.TenantID, ActorID: p.UserID, Action: "billing.checkout.blocked",
				TargetType: "tenant", TargetID: p.TenantID,
				After: map[string]any{"reason": "kyc_required", "kyc_status": status}, IsPaper: p.IsPaper,
			})
			writeErr(w, http.StatusForbidden, "kyc_required", "identity verification is required before real-money purchases")
			return
		}
	}

	purchaseID := uuid.NewString()
	session, err := s.stripe.CreateCheckoutSession(r.Context(), billing.CheckoutParams{
		PurchaseID: purchaseID, TenantID: p.TenantID, Amount: b.Amount, CreditType: b.CreditType, Currency: b.Currency,
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
		UnitPriceUSD: unitPrice, ChargedUSDCents: cents,
	}, session.ID); err != nil {
		serverError(w, err)
		return
	}
	slog.Info("audit: checkout created", "tenant_id", p.TenantID, "purchase_id", purchaseID, "amount", b.Amount, "credit_type", b.CreditType)

	// Dev/sandbox: MockStripe has no hosted checkout page and no webhook ever fires, so settle the
	// purchase inline (mark paid + book the credits) exactly as the webhook would. Real Stripe
	// deployments use a different StripeClient, so this never runs in prod — there, the signed
	// checkout.session.completed webhook books the credits. Idempotent on a synthetic event id.
	settled := false
	if _, isMock := s.stripe.(billing.MockStripe); s.cfg.BillingAutoSettle && isMock {
		evID := "evt_mock_" + purchaseID
		if err := s.st.MarkPurchasePaid(r.Context(), session.ID, evID); err != nil {
			serverError(w, err)
			return
		}
		if err := s.booker.BookPurchase(r.Context(), billing.PurchaseBooking{
			TenantID: p.TenantID, Amount: b.Amount, CreditType: b.CreditType, IsPaper: p.IsPaper,
			ReferenceID: purchaseID, IdempotencyKey: evID,
		}); err != nil {
			serverError(w, err)
			return
		}
		settled = true
		slog.Info("audit: mock checkout settled instantly (dev)", "tenant_id", p.TenantID, "purchase_id", purchaseID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"purchase_id": purchaseID, "checkout_url": session.URL, "settled": settled})
}

// stripeEvent is the slice of a Stripe Event payload we act on.
type stripeEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object struct {
			ID string `json:"id"`
		} `json:"object"`
	} `json:"data"`
}

// stripeWebhook receives Stripe events. Authenticity is the Stripe-Signature header (HMAC), never a
// bearer token. On checkout.session.completed it marks the purchase paid and books the credits to
// the ledger, idempotent on the Stripe event id (a replay never double-mints).
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
	if ev.Type != "checkout.session.completed" {
		w.WriteHeader(http.StatusOK) // not a settlement event — ack and ignore
		return
	}
	pur, found, err := s.st.GetPurchaseBySession(r.Context(), ev.Data.Object.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	if !found {
		w.WriteHeader(http.StatusOK) // unknown session — ack so Stripe stops retrying
		return
	}
	if err := s.st.MarkPurchasePaid(r.Context(), ev.Data.Object.ID, ev.ID); err != nil {
		serverError(w, err)
		return
	}
	// Book the credits. Idempotent on the event id, so a retry (e.g. after a transient failure here)
	// recovers without double-minting. A failure returns 500 → Stripe retries.
	if err := s.booker.BookPurchase(r.Context(), billing.PurchaseBooking{
		TenantID: pur.TenantID, Amount: pur.Amount, CreditType: pur.CreditType, IsPaper: pur.IsPaper,
		ReferenceID: pur.ID, IdempotencyKey: ev.ID,
	}); err != nil {
		serverError(w, err)
		return
	}
	slog.Info("audit: purchase booked", "tenant_id", pur.TenantID, "purchase_id", pur.ID, "event_id", ev.ID, "amount", pur.Amount)
	w.WriteHeader(http.StatusOK)
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
		}
		if pu.PaidAt != nil {
			row["paid_at"] = pu.PaidAt.UTC().Format(time.RFC3339)
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"purchases": out})
}
