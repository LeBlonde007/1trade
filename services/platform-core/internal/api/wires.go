package api

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/trade1/platform-core/internal/billing"
	"github.com/trade1/platform-core/internal/domain"
	"github.com/trade1/platform-core/internal/store"
)

// wireMinimumCents is the smallest wire we invoice ($1,000): below that, card or ACH is cheaper for
// everyone.
const wireMinimumCents = 100_000

// wireRoutes registers wire-transfer purchases (platform-core.yaml v1.9).
func (s *Server) wireRoutes() {
	s.mux.HandleFunc("POST /v1/billing/wires", s.createWire)
	s.mux.HandleFunc("POST /v1/billing/wires/{id}/received", s.wireReceived) // treasury (service token)
}

// centsToUSD renders integer cents as a dollar string ("1234.50").
func centsToUSD(c int64) string { return fmt.Sprintf("%d.%02d", c/100, c%100) }

// usdRe is a dollar amount with at most two decimals.
var usdRe = regexp.MustCompile(`^(0|[1-9][0-9]{0,11})(\.[0-9]{1,2})?$`)

// parseUSDCents parses "1234.5" → 123450 exactly (no floats).
func parseUSDCents(s string) (int64, bool) {
	if !usdRe.MatchString(s) {
		return 0, false
	}
	whole, frac, _ := strings.Cut(s, ".")
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, false
	}
	f := int64(0)
	if frac != "" {
		frac += strings.Repeat("0", 2-len(frac))
		f, _ = strconv.ParseInt(frac, 10, 64)
	}
	return w*100 + f, true
}

// newWireReference is the code the customer puts on the wire so treasury can match it: "1T-" and
// ten characters that avoid look-alikes.
func newWireReference() (string, error) {
	b := make([]byte, 7)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "1T-" + base32.StdEncoding.EncodeToString(b)[:10], nil
}

// createWire serves POST /v1/billing/wires {amount, credit_type} (admin or billing): an invoice to pay
// by USD wire — our bank details, the exact dollar amount and a unique reference. Credits are booked
// when treasury records the wire.
func (s *Server) createWire(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok || !requireRole(w, p, domain.RoleBilling) {
		return
	}
	if !s.cfg.Wire.Configured() {
		writeErr(w, http.StatusServiceUnavailable, "wire_unavailable", "wire transfers are not available in this environment")
		return
	}
	var b struct {
		Amount     string `json:"amount"`
		CreditType string `json:"credit_type"`
	}
	if !decodeBody(w, r, &b) {
		return
	}
	amt, ok := new(big.Rat).SetString(b.Amount)
	if !ok || amt.Sign() <= 0 || !domain.ValidCreditType(b.CreditType) {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "a positive amount and a known credit_type are required")
		return
	}
	unitPrice, cents, err := billing.QuoteUSD(b.CreditType, b.Amount)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "unpriced", "that credit type has no published price")
		return
	}
	if cents < wireMinimumCents {
		writeErr(w, http.StatusUnprocessableEntity, "below_wire_minimum", "wires start at $"+centsToUSD(wireMinimumCents)+"; use card or ACH for smaller purchases")
		return
	}
	if !s.realMoneyAllowed(w, r, p) {
		return
	}
	ref, err := newWireReference()
	if err != nil {
		serverError(w, err)
		return
	}
	id := uuid.NewString()
	if err := s.st.CreatePurchase(r.Context(), store.Purchase{
		ID: id, TenantID: p.TenantID, Amount: b.Amount, CreditType: b.CreditType, Currency: "usd", IsPaper: p.IsPaper,
		UnitPriceUSD: unitPrice, ChargedUSDCents: cents, Method: "wire", WireReference: ref,
	}, ""); err != nil {
		serverError(w, err)
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{TenantID: p.TenantID, ActorID: p.UserID, Action: "billing.wire.invoice",
		TargetType: "purchase", TargetID: id, After: map[string]any{"amount_usd": centsToUSD(cents), "credits": b.Amount,
			"credit_type": b.CreditType, "reference": ref}, IsPaper: p.IsPaper})
	writeJSON(w, http.StatusCreated, map[string]any{
		"purchase_id": id, "status": "pending", "method": "wire", "currency": "usd",
		"amount_usd": centsToUSD(cents), "credits": b.Amount, "credit_type": b.CreditType,
		"reference": ref, "instructions": s.cfg.Wire, "is_paper": p.IsPaper,
	})
}

// wireReceived serves POST /v1/billing/wires/{id}/received {amount_usd, bank_reference} — treasury
// (service token) records an incoming wire. The amount must equal the invoice to the cent (anything
// else is 409 and handled by hand); then the credits are booked, at most once per purchase. A replay
// of the same receipt books again idempotently, which also recovers a booking that failed.
func (s *Server) wireReceived(w http.ResponseWriter, r *http.Request) {
	if !serviceAuthorized(s.cfg, r) {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "service authorization required")
		return
	}
	var b struct {
		AmountUSD     string `json:"amount_usd"`
		BankReference string `json:"bank_reference"`
	}
	if !decodeBody(w, r, &b) {
		return
	}
	cents, ok := parseUSDCents(strings.TrimSpace(b.AmountUSD))
	b.BankReference = strings.TrimSpace(b.BankReference)
	if !ok || cents <= 0 || b.BankReference == "" || len(b.BankReference) > 128 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "amount_usd (dollars, at most 2 decimals) and bank_reference are required")
		return
	}
	id := r.PathValue("id")
	if uuid.Validate(id) != nil {
		writeErr(w, http.StatusNotFound, "not_found", "no such wire purchase")
		return
	}
	pur, replayed, err := s.st.MarkWireReceived(r.Context(), id, cents, b.BankReference)
	switch {
	case errors.Is(err, store.ErrWireNotFound):
		writeErr(w, http.StatusNotFound, "not_found", "no such wire purchase")
		return
	case errors.Is(err, store.ErrWireAmount):
		writeErr(w, http.StatusConflict, "amount_mismatch", "the received amount does not match the invoice; reconcile by hand")
		return
	case errors.Is(err, store.ErrWireSettled):
		writeErr(w, http.StatusConflict, "not_pending", "this wire purchase is already settled")
		return
	case err != nil:
		serverError(w, err)
		return
	}
	if err := s.booker.BookPurchase(r.Context(), billing.PurchaseBooking{
		TenantID: pur.TenantID, Amount: pur.Amount, CreditType: pur.CreditType, IsPaper: pur.IsPaper,
		ReferenceID: pur.ID, IdempotencyKey: bookingKey(pur.ID),
	}); err != nil {
		slog.Error("wire received but booking failed; replay the receipt to retry", "purchase_id", pur.ID, "err", err)
		writeErr(w, http.StatusBadGateway, "booking_failed", "recorded, but the credits were not booked; send the same receipt again")
		return
	}
	if !replayed {
		_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{TenantID: pur.TenantID, Action: "billing.wire.received",
			TargetType: "purchase", TargetID: pur.ID, After: map[string]any{"amount_usd": centsToUSD(cents), "bank_reference": b.BankReference,
				"at": time.Now().UTC().Format(time.RFC3339)}, IsPaper: pur.IsPaper})
	}
	writeJSON(w, http.StatusOK, map[string]any{"purchase_id": pur.ID, "status": "paid", "credits": pur.Amount, "credit_type": pur.CreditType})
}
