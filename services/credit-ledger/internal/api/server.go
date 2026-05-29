// Package api is the HTTP surface of the credit ledger — it maps docs/contracts/openapi/credit.yaml
// onto the store, enforcing auth (tenant JWT for reads, service token for internal movements) and
// the money/idempotency/is_paper conventions. Conversion (/convert) is F07 and returns 501 here.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/exascale/credit-ledger/internal/config"
	"github.com/exascale/credit-ledger/internal/domain"
	"github.com/exascale/credit-ledger/internal/events"
	"github.com/exascale/credit-ledger/internal/store"
)

// Server wires config + store + event publisher into an http.Handler.
type Server struct {
	cfg config.Config
	st  *store.Store
	pub events.Publisher
	mux *http.ServeMux
}

// New builds the routed HTTP handler.
func New(cfg config.Config, st *store.Store, pub events.Publisher) *Server {
	s := &Server{cfg: cfg, st: st, pub: pub, mux: http.NewServeMux()}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// routes registers every endpoint (Go 1.22+ method+path patterns).
func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	s.mux.HandleFunc("GET /readyz", s.readyz)
	s.mux.HandleFunc("GET /v1/credits/balances", s.getBalances)
	s.mux.HandleFunc("GET /v1/credits/transactions", s.listTransactions)
	s.mux.HandleFunc("POST /v1/credits/purchase", s.movement(domain.OpPurchase, +1))
	s.mux.HandleFunc("POST /v1/credits/debit", s.movement(domain.OpConsumption, -1))
	s.mux.HandleFunc("POST /v1/credits/mint", s.movement(domain.OpMint, +1))
	s.mux.HandleFunc("POST /v1/credits/burn", s.movement(domain.OpBurn, -1))
	s.mux.HandleFunc("GET /v1/credits/audit/chain-verify", s.chainVerify)
	s.mux.HandleFunc("POST /v1/credits/convert", notImplemented("conversion is F07"))
	s.mux.HandleFunc("GET /v1/credits/conversion-rates", notImplemented("conversion is F07"))
}

// readyz checks the DB is reachable.
func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	if _, _, _, err := s.st.VerifyChain(r.Context(), "00000000-0000-0000-0000-000000000000"); err != nil {
		writeErr(w, http.StatusServiceUnavailable, "not_ready", err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

// getBalances lists the authenticated tenant's balances for the requested ledger (paper/real).
func (s *Server) getBalances(w http.ResponseWriter, r *http.Request) {
	p, err := tenantPrincipal(s.cfg, r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}
	isPaper := r.URL.Query().Get("is_paper") != "false"
	bals, err := s.st.GetBalances(r.Context(), p.TenantID, isPaper)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	out := make([]balanceDTO, 0, len(bals))
	for _, b := range bals {
		out = append(out, balanceDTO{string(b.CreditType), b.Balance.String(), b.Locked.String(), b.IsPaper, b.SubAccountID})
	}
	writeJSON(w, http.StatusOK, map[string]any{"balances": out})
}

// listTransactions returns the tenant's transactions, newest first.
func (s *Server) listTransactions(w http.ResponseWriter, r *http.Request) {
	p, err := tenantPrincipal(s.cfg, r)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}
	isPaper := r.URL.Query().Get("is_paper") != "false"
	txs, err := s.st.ListTransactions(r.Context(), p.TenantID, isPaper, 50)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transactions": toTxDTOs(txs)})
}

// movementBody is the union of the single-leg movement request bodies (purchase/debit/mint/burn).
type movementBody struct {
	TenantID     string `json:"tenant_id"`
	SubAccountID string `json:"sub_account_id"`
	CreditType   string `json:"credit_type"`
	Amount       string `json:"amount"`
	ReferenceID  string `json:"reference_id"`
	UsageEventID string `json:"usage_event_id"`
	IsPaper      *bool  `json:"is_paper"`
}

// movement returns a handler for a single-leg movement. sign is +1 (credit) or -1 (debit); it is
// applied to the (positive) request amount to form the signed ledger delta.
func (s *Server) movement(op domain.Operation, sign int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := servicePrincipal(s.cfg, r); err != nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized", err.Error())
			return
		}
		var b movementBody
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			writeErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		amt, err := domain.ParseMoney(b.Amount)
		if err != nil || amt.Sign() < 0 {
			writeErr(w, http.StatusUnprocessableEntity, "bad_amount", "amount must be a non-negative fixed-point decimal")
			return
		}
		if sign < 0 {
			amt = domain.Zero().Sub(amt) // negate for debits/burns
		}
		if b.TenantID == "" || b.CreditType == "" || b.IsPaper == nil {
			writeErr(w, http.StatusUnprocessableEntity, "bad_request", "tenant_id, credit_type, is_paper required")
			return
		}
		// Idempotency anchor: header if present, else the natural per-op reference.
		idem := r.Header.Get("Idempotency-Key")
		if idem == "" {
			idem = b.UsageEventID
		}
		if idem == "" {
			idem = b.ReferenceID
		}
		ref := b.ReferenceID
		if ref == "" {
			ref = b.UsageEventID
		}

		tx, err := s.st.ApplyMovement(r.Context(), store.Movement{
			TenantID: b.TenantID, SubAccountID: b.SubAccountID, CreditType: domain.CreditType(b.CreditType),
			Operation: op, Amount: amt, ReferenceID: ref, IdempotencyKey: idem, IsPaper: *b.IsPaper,
		})
		if errors.Is(err, domain.ErrInsufficientCredit) {
			writeErr(w, http.StatusPaymentRequired, "INSUFFICIENT_CREDIT", "balance too low")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "store_error", err.Error())
			return
		}
		s.pub.PublishTx(tx)
		writeJSON(w, http.StatusOK, toTxDTO(tx))
	}
}

// chainVerify re-derives the hash chain for a tenant (service-only).
func (s *Server) chainVerify(w http.ResponseWriter, r *http.Request) {
	if _, err := servicePrincipal(s.cfg, r); err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "tenant_id required")
		return
	}
	ok, checked, firstBad, err := s.st.VerifyChain(r.Context(), tenantID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}
	resp := map[string]any{"ok": ok, "checked": checked}
	if !ok {
		resp["first_bad_tx_id"] = firstBad
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- DTOs (snake_case, money as strings — matches credit.yaml) ---

type balanceDTO struct {
	CreditType   string `json:"credit_type"`
	Balance      string `json:"balance"`
	Locked       string `json:"locked_amount"`
	IsPaper      bool   `json:"is_paper"`
	SubAccountID string `json:"sub_account_id,omitempty"`
}

type txDTO struct {
	TxID         string `json:"tx_id"`
	CreditType   string `json:"credit_type"`
	Operation    string `json:"operation"`
	Amount       string `json:"amount"`
	ReferenceID  string `json:"reference_id,omitempty"`
	BalanceAfter string `json:"balance_after"`
	IsPaper      bool   `json:"is_paper"`
	CreatedAt    string `json:"created_at"`
	ChainHash    string `json:"chain_hash"`
}

// toTxDTO maps a domain transaction to its contract shape.
func toTxDTO(t domain.Transaction) txDTO {
	return txDTO{
		TxID: t.TxID, CreditType: string(t.CreditType), Operation: string(t.Operation),
		Amount: t.Amount.String(), ReferenceID: t.ReferenceID, BalanceAfter: t.BalanceAfter.String(),
		IsPaper: t.IsPaper, CreatedAt: t.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		ChainHash: t.ChainHash,
	}
}

// toTxDTOs maps a slice.
func toTxDTOs(ts []domain.Transaction) []txDTO {
	out := make([]txDTO, 0, len(ts))
	for _, t := range ts {
		out = append(out, toTxDTO(t))
	}
	return out
}

// notImplemented returns a 501 handler (used for F07 conversion endpoints).
func notImplemented(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeErr(w, http.StatusNotImplemented, "not_implemented", msg)
	}
}

// writeJSON writes a JSON response with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr writes the contract's ApiError shape.
func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"code": code, "message": msg})
}
