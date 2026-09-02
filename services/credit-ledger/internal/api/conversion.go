package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/trade1/credit-ledger/internal/domain"
	"github.com/trade1/credit-ledger/internal/store"
)

// convertBody is the POST /v1/credits/convert request (credit.yaml ConvertRequest).
type convertBody struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount string `json:"amount"`
}

// convert converts between credit types atomically at the published rate (bidirectional, house
// spread). Customer-authenticated; is_paper comes from the tenant. Returns both ledger legs.
func (s *Server) convert(w http.ResponseWriter, r *http.Request) {
	p, err := tenantPrincipal(s.cfg, r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	idem := r.Header.Get("Idempotency-Key")
	if idem == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "Idempotency-Key header is required")
		return
	}
	var b convertBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if b.From == "" || b.To == "" || b.From == b.To {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "from and to must be set and differ")
		return
	}
	amt, err := domain.ParseMoney(b.Amount)
	if err != nil || amt.Sign() <= 0 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_amount", "amount must be a positive fixed-point decimal")
		return
	}

	debit, credit, _, _, err := s.st.ApplyConversion(r.Context(), store.Conversion{
		TenantID: p.TenantID, From: domain.CreditType(b.From), To: domain.CreditType(b.To),
		Amount: amt, IsPaper: p.IsPaper, IdempotencyKey: idem, ReferenceID: idem,
	})
	switch {
	case errors.Is(err, store.ErrNoRate):
		writeErr(w, http.StatusUnprocessableEntity, "no_rate", "no conversion rate for that pair")
		return
	case errors.Is(err, domain.ErrInsufficientCredit):
		writeErr(w, http.StatusPaymentRequired, "INSUFFICIENT_CREDIT", "balance too low")
		return
	case err != nil:
		serverError(w, err)
		return
	}
	s.pub.PublishTx(debit)
	s.pub.PublishTx(credit)
	writeJSON(w, http.StatusOK, map[string]any{"debit": toTxDTO(debit), "credit": toTxDTO(credit)})
}

// conversionRates returns the current house spread + per-pair rates (public; rates are not secret).
func (s *Server) conversionRates(w http.ResponseWriter, r *http.Request) {
	rates, err := s.st.ListRates(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	spread := "0.010000"
	out := make([]map[string]any, 0, len(rates))
	for i, rt := range rates {
		if i == 0 {
			spread = rt.Spread
		}
		out = append(out, map[string]any{"from": rt.From, "to": rt.To, "rate": rt.Rate})
	}
	writeJSON(w, http.StatusOK, map[string]any{"spread": spread, "rates": out})
}
