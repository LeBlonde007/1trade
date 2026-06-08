package api

import (
	"encoding/json"
	"log/slog"
	"math/big"
	"net/http"
	"time"

	"github.com/exascale/platform-core/internal/domain"
	"github.com/exascale/platform-core/internal/email"
	"github.com/exascale/platform-core/internal/store"
)

// issueVerifyToken generates + stores a verification token for a user and returns the raw token to
// deliver (email in prod). Only the hash is stored. Best-effort: a failure is logged, never fatal.
func (s *Server) issueVerifyToken(r *http.Request, userID string) string {
	raw, err := domain.NewVerifyToken()
	if err != nil {
		return ""
	}
	if err := s.st.SetVerifyToken(r.Context(), userID, domain.HashAPIKey(raw)); err != nil {
		slog.Error("set verify token", "err", err)
		return ""
	}
	return raw
}

// verifyEmail consumes an email-verification token. No bearer auth — the token IS the credential.
func (s *Server) verifyEmail(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Token == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "token is required")
		return
	}
	userID, tenantID, isPaper, ok, err := s.st.VerifyEmail(r.Context(), domain.HashAPIKey(b.Token))
	if err != nil {
		serverError(w, err)
		return
	}
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid_token", "invalid or already-used verification token")
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: tenantID, ActorID: userID, Action: "user.email.verify", TargetType: "user", TargetID: userID, IsPaper: isPaper,
	})
	writeJSON(w, http.StatusOK, map[string]any{"verified": true})
}

// resendVerify issues a fresh email-verification token. Two modes:
//   - UNAUTHENTICATED with {"email": ...} — the login screen, where a user blocked by the verification
//     gate has no session yet. It resends only if that email exists and is still unverified, but ALWAYS
//     returns {"sent": true} so it never reveals whether an email is registered (account enumeration),
//     and it never returns the dev token on this path.
//   - AUTHED (valid session — e.g. an in-app resend button) — resend for the caller; in dev with no
//     mailer the raw token is returned so the flow stays testable.
func (s *Server) resendVerify(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Email string `json:"email"`
	}
	_ = json.NewDecoder(r.Body).Decode(&b) // body is optional (authed mode sends none)

	if b.Email != "" {
		if au, found, err := s.st.GetUserByEmail(r.Context(), b.Email); err != nil {
			serverError(w, err)
			return
		} else if found && !au.EmailVerified {
			if raw := s.issueVerifyToken(r, au.UserID); raw != "" {
				if err := s.mailer.SendVerification(b.Email, email.VerifyURL(s.cfg.AppBaseURL, raw)); err != nil {
					slog.Error("resend verification email (by email)", "err", err)
				}
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"sent": true}) // constant response — no enumeration
		return
	}

	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	raw := s.issueVerifyToken(r, p.UserID)
	if raw != "" {
		if idn, found, err := s.st.GetUserByID(r.Context(), p.UserID); err == nil && found {
			if err := s.mailer.SendVerification(idn.Email, email.VerifyURL(s.cfg.AppBaseURL, raw)); err != nil {
				slog.Error("resend verification email", "err", err, "user_id", p.UserID)
			}
		}
	}
	out := map[string]any{"sent": true}
	if s.cfg.IsDev() && raw != "" && !s.mailer.Enabled() {
		out["dev_token"] = raw // no SMTP configured — keep the flow testable
	}
	writeJSON(w, http.StatusOK, out)
}

// getBudget returns the tenant's monthly credit budget (budget=null if none set).
func (s *Server) getBudget(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	b, found, err := s.st.GetBudget(r.Context(), p.TenantID)
	if err != nil {
		serverError(w, err)
		return
	}
	if !found {
		writeJSON(w, http.StatusOK, map[string]any{"budget": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"budget": map[string]any{
		"credit_type": b.CreditType, "monthly_limit": b.MonthlyLimit, "updated_at": b.UpdatedAt.UTC().Format(time.RFC3339),
	}})
}

// setBudget upserts the tenant's monthly budget (billing/admin). Audited. Consumption alerts
// (50/80/100%) and auto-stop compare month-to-date usage against this limit (auto-stop is M3).
func (s *Server) setBudget(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	if !requireRole(w, p, domain.RoleBilling) {
		return
	}
	var b struct {
		CreditType   string `json:"credit_type"`
		MonthlyLimit string `json:"monthly_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.CreditType == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "credit_type and monthly_limit are required")
		return
	}
	amt, valid := new(big.Rat).SetString(b.MonthlyLimit)
	if !valid || amt.Sign() < 0 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_amount", "monthly_limit must be a non-negative decimal")
		return
	}
	if err := s.st.SetBudget(r.Context(), p.TenantID, b.CreditType, b.MonthlyLimit); err != nil {
		serverError(w, err)
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: p.TenantID, ActorID: p.UserID, Action: "budget.set", TargetType: "budget", TargetID: p.TenantID,
		After: map[string]any{"credit_type": b.CreditType, "monthly_limit": b.MonthlyLimit}, IsPaper: p.IsPaper,
	})
	writeJSON(w, http.StatusOK, map[string]any{"credit_type": b.CreditType, "monthly_limit": b.MonthlyLimit})
}
