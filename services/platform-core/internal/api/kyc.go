package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/trade1/platform-core/internal/domain"
	"github.com/trade1/platform-core/internal/store"
)

// kycResponse renders a KYC record for the API (GET + submit return the same shape). can_purchase is
// the bottom line the web app cares about: may this tenant buy with real money right now.
func kycResponse(k store.KYCRecord) map[string]any {
	out := map[string]any{
		"status":       string(k.Status),
		"can_purchase": domain.CanPurchaseRealMoney(k.Status),
		"can_submit":   domain.CanSubmitKYC(k.Status),
	}
	if k.LegalName != "" {
		out["legal_name"] = k.LegalName
	}
	if k.Country != "" {
		out["country"] = k.Country
	}
	if k.EntityType != "" {
		out["entity_type"] = k.EntityType
	}
	if k.SubmittedAt != nil {
		out["submitted_at"] = k.SubmittedAt.UTC().Format(time.RFC3339)
	}
	if k.ReviewedAt != nil {
		out["reviewed_at"] = k.ReviewedAt.UTC().Format(time.RFC3339)
	}
	return out
}

// getKYC returns the caller's tenant KYC status (authed).
func (s *Server) getKYC(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	k, found, err := s.st.GetKYC(r.Context(), p.TenantID)
	if err != nil {
		serverError(w, err)
		return
	}
	if !found {
		writeErr(w, http.StatusNotFound, "not_found", "tenant not found")
		return
	}
	writeJSON(w, http.StatusOK, kycResponse(k))
}

// submitKYC records an identity-verification submission for the caller's tenant. In dev/sandbox the
// submission is auto-verified (KYCAutoApprove); in prod it lands `pending` for review. This gates only
// real-money purchases — credit usage and sandbox flows are unaffected. Audited (minimal PII).
func (s *Server) submitKYC(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	var b struct {
		LegalName  string `json:"legal_name"`
		Country    string `json:"country"`
		EntityType string `json:"entity_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	country := strings.ToUpper(strings.TrimSpace(b.Country))
	if strings.TrimSpace(b.LegalName) == "" || len(country) != 2 || !domain.ValidEntityType(b.EntityType) {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request",
			"legal_name, ISO-3166 alpha-2 country, and entity_type (individual|business) are required")
		return
	}
	cur, found, err := s.st.GetKYC(r.Context(), p.TenantID)
	if err != nil {
		serverError(w, err)
		return
	}
	if !found {
		writeErr(w, http.StatusNotFound, "not_found", "tenant not found")
		return
	}
	if !domain.CanSubmitKYC(cur.Status) {
		writeErr(w, http.StatusConflict, "kyc_not_submittable", "verification already "+string(cur.Status))
		return
	}
	newStatus, applied, err := s.st.SubmitKYC(r.Context(), p.TenantID,
		store.KYCSubmission{LegalName: strings.TrimSpace(b.LegalName), Country: country, EntityType: b.EntityType},
		s.cfg.KYCAutoApprove)
	if err != nil {
		serverError(w, err)
		return
	}
	if !applied { // lost a race against a concurrent submit — return the now-current state, don't error
		latest, _, _ := s.st.GetKYC(r.Context(), p.TenantID)
		writeJSON(w, http.StatusOK, kycResponse(latest))
		return
	}
	action := "kyc.submit"
	if newStatus == domain.KYCVerified {
		action = "kyc.verified"
	}
	slog.Info("audit: kyc submitted", "tenant_id", p.TenantID, "status", newStatus)
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: p.TenantID, ActorID: p.UserID, Action: action, TargetType: "tenant", TargetID: p.TenantID,
		Before: map[string]any{"status": string(cur.Status)},
		After:  map[string]any{"status": string(newStatus), "country": country, "entity_type": b.EntityType},
		IsPaper: p.IsPaper,
	})
	latest, _, _ := s.st.GetKYC(r.Context(), p.TenantID)
	writeJSON(w, http.StatusOK, kycResponse(latest))
}

// reviewKYC applies a compliance decision (verified|rejected) to a pending tenant submission. Internal:
// guarded by the service-to-service token (a review queue / ops tool calls it), never an end-user
// bearer — a tenant must not verify its own KYC. In dev this path is unused (submissions auto-verify).
func (s *Server) reviewKYC(w http.ResponseWriter, r *http.Request) {
	if !serviceAuthorized(s.cfg, r) {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "service authorization required")
		return
	}
	tenantID := r.PathValue("tenant_id")
	var b struct {
		Decision string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	decision := domain.KYCStatus(b.Decision)
	if !domain.ValidKYCDecision(decision) {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "decision must be verified or rejected")
		return
	}
	ok, err := s.st.ReviewKYC(r.Context(), tenantID, decision)
	if err != nil {
		serverError(w, err)
		return
	}
	if !ok {
		writeErr(w, http.StatusConflict, "not_pending", "no pending submission for tenant")
		return
	}
	slog.Info("audit: kyc reviewed", "tenant_id", tenantID, "decision", decision)
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: tenantID, Action: "kyc." + string(decision), TargetType: "tenant", TargetID: tenantID,
		After: map[string]any{"status": string(decision)},
	})
	writeJSON(w, http.StatusOK, map[string]any{"tenant_id": tenantID, "status": string(decision)})
}
