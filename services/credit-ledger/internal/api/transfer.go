package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/trade1/credit-ledger/internal/domain"
	"github.com/trade1/credit-ledger/internal/store"
)

// transfer serves POST /v1/credits/transfer (credit.yaml v1.4, service token): move credits between
// a tenant's main balance and a sub-account (platform-core funds and returns team budgets). The
// tenant and side come from the body because only the platform calls it, after its own RBAC check —
// so, like settlement, it takes the real service token only (no dev shortcut).
func (s *Server) transfer(w http.ResponseWriter, r *http.Request) {
	if tok := bearer(r); s.cfg.ServiceToken == "" || subtle.ConstantTimeCompare([]byte(tok), []byte(s.cfg.ServiceToken)) != 1 {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	idem := r.Header.Get("Idempotency-Key")
	if idem == "" || len(idem) > 200 {
		writeErr(w, http.StatusBadRequest, "bad_request", "Idempotency-Key header is required")
		return
	}
	var b struct {
		TenantID   string  `json:"tenant_id"`
		FromSub    *string `json:"from_sub_account_id"`
		ToSub      *string `json:"to_sub_account_id"`
		CreditType string  `json:"credit_type"`
		Amount     string  `json:"amount"`
		IsPaper    *bool   `json:"is_paper"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil || b.IsPaper == nil || b.CreditType == "" {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "tenant_id, credit_type, amount and is_paper are required")
		return
	}
	from, to := deref(b.FromSub), deref(b.ToSub)
	if !isUUID(b.TenantID) || (from != "" && !isUUID(from)) || (to != "" && !isUUID(to)) {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "tenant and sub-account ids must be UUIDs (null = the main balance)")
		return
	}
	amt, err := domain.ParseMoney(b.Amount)
	if err != nil || amt.Sign() <= 0 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_amount", "amount must be a positive fixed-point decimal")
		return
	}
	debit, credit, err := s.st.ApplyTransfer(r.Context(), store.Transfer{
		TenantID: b.TenantID, FromSub: from, ToSub: to, CreditType: domain.CreditType(b.CreditType),
		Amount: amt, IsPaper: *b.IsPaper, IdempotencyKey: idem, ReferenceID: idem,
	})
	switch {
	case errors.Is(err, store.ErrSameAccount):
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "from and to must be different balances")
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

// isUUID reports whether s is a UUID.
func isUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// deref returns the string or "" for nil.
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
