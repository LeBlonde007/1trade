package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/trade1/platform-core/internal/billing"
	"github.com/trade1/platform-core/internal/domain"
	"github.com/trade1/platform-core/internal/email"
	"github.com/trade1/platform-core/internal/store"
)

// inviteTTL is how long an invitation link stays valid.
const inviteTTL = 7 * 24 * time.Hour

// teamRoutes registers team management (platform-core.yaml v1.x: members, invitations, sub-accounts).
func (s *Server) teamRoutes() {
	s.mux.HandleFunc("GET /v1/account/members", s.listMembers)
	s.mux.HandleFunc("DELETE /v1/account/members/{id}", s.removeMember)
	s.mux.HandleFunc("PUT /v1/account/members/{id}/sub-account", s.setMemberSubAccount)
	s.mux.HandleFunc("POST /v1/account/invites", s.createInvite)
	s.mux.HandleFunc("GET /v1/account/invites", s.listInvites)
	s.mux.HandleFunc("DELETE /v1/account/invites/{id}", s.revokeInvite)
	s.mux.HandleFunc("GET /v1/account/invites/lookup", s.lookupInvite)  // the token is the credential
	s.mux.HandleFunc("POST /v1/account/invites/accept", s.acceptInvite) // the token is the credential
	s.mux.HandleFunc("POST /v1/account/sub-accounts", s.createSubAccount)
	s.mux.HandleFunc("GET /v1/account/sub-accounts", s.listSubAccounts)
	s.mux.HandleFunc("POST /v1/account/sub-accounts/{id}/transfer", s.transferSubAccount)
}

// SetBudgetMover sets the ledger client that funds sub-accounts (tests inject a fake).
func (s *Server) SetBudgetMover(t billing.BudgetMover) { s.transfer = t }

// decodeBody reads a small JSON body, refusing unknown fields.
func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := jsonDecoder(w, r, 8192).Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return false
	}
	return true
}

// jsonDecoder reads at most limit bytes of JSON, refusing unknown fields.
func jsonDecoder(w http.ResponseWriter, r *http.Request, limit int64) *json.Decoder {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	dec.DisallowUnknownFields()
	return dec
}

// writeTeamErr maps team errors onto statuses.
func writeTeamErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotMember):
		writeErr(w, http.StatusNotFound, "not_found", "no such member")
	case errors.Is(err, store.ErrNoSubAccount):
		writeErr(w, http.StatusNotFound, "not_found", "no such sub-account")
	case errors.Is(err, store.ErrLastAdmin):
		writeErr(w, http.StatusConflict, "last_admin", "a tenant must keep at least one admin")
	case errors.Is(err, store.ErrInvitePending):
		writeErr(w, http.StatusConflict, "invite_pending", "an invitation to that email is already pending")
	case errors.Is(err, store.ErrEmailTaken):
		writeErr(w, http.StatusConflict, "email_taken", "that email already has a 1Trade account")
	case errors.Is(err, store.ErrInviteInvalid):
		writeErr(w, http.StatusNotFound, "invite_invalid", "this invitation is invalid, used, revoked or expired")
	case errors.Is(err, store.ErrSubAccountName):
		writeErr(w, http.StatusConflict, "name_taken", "a sub-account with that name exists")
	default:
		serverError(w, err)
	}
}

// memberJSON renders a member.
func memberJSON(m store.Member) map[string]any {
	return map[string]any{
		"id": m.ID, "email": m.Email, "roles": m.Roles, "sub_account_id": m.SubAccountID,
		"email_verified": m.EmailVerified, "mfa_enabled": m.MFAEnabled, "created_at": m.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// listMembers serves GET /v1/account/members: the tenant's users (any member may see the team).
func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	ms, err := s.st.ListMembers(r.Context(), p.TenantID)
	if err != nil {
		serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(ms))
	for _, m := range ms {
		out = append(out, memberJSON(m))
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": out})
}

// removeMember serves DELETE /v1/account/members/{id} (admin): not yourself, never the last admin.
func (s *Server) removeMember(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok || !requireRole(w, p, domain.RoleAdmin) {
		return
	}
	id := r.PathValue("id")
	if id == p.UserID {
		writeErr(w, http.StatusConflict, "self", "you cannot remove yourself")
		return
	}
	m, err := s.st.RemoveMember(r.Context(), p.TenantID, id)
	if err != nil {
		writeTeamErr(w, err)
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: p.TenantID, ActorID: p.UserID, Action: "user.remove", TargetType: "user", TargetID: id,
		Before: map[string]any{"email": m.Email, "roles": m.Roles}, IsPaper: p.IsPaper,
	})
	w.WriteHeader(http.StatusNoContent)
}

// setMemberSubAccount serves PUT /v1/account/members/{id}/sub-account (admin): the member's usage is
// billed to that sub-account from their next sign-in (null = the main balance).
func (s *Server) setMemberSubAccount(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok || !requireRole(w, p, domain.RoleAdmin) {
		return
	}
	var b struct {
		SubAccountID *string `json:"sub_account_id"`
	}
	if !decodeBody(w, r, &b) {
		return
	}
	id := r.PathValue("id")
	prev, err := s.st.SetMemberSubAccount(r.Context(), p.TenantID, id, b.SubAccountID)
	if err != nil {
		writeTeamErr(w, err)
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: p.TenantID, ActorID: p.UserID, Action: "user.sub_account.set", TargetType: "user", TargetID: id,
		Before: map[string]any{"sub_account_id": prev}, After: map[string]any{"sub_account_id": b.SubAccountID}, IsPaper: p.IsPaper,
	})
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "sub_account_id": b.SubAccountID})
}

// inviteJSON renders an invitation.
func inviteJSON(inv store.Invite) map[string]any {
	return map[string]any{
		"id": inv.ID, "email": inv.Email, "roles": inv.Roles, "sub_account_id": inv.SubAccountID,
		"invited_by": inv.InvitedBy, "created_at": inv.CreatedAt.UTC().Format(time.RFC3339),
		"expires_at": inv.ExpiresAt.UTC().Format(time.RFC3339),
	}
}

// createInvite serves POST /v1/account/invites (admin): invite an email with roles (and optionally a
// sub-account). The link goes by email; in dev with no mail transport the token is returned.
func (s *Server) createInvite(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok || !requireRole(w, p, domain.RoleAdmin) {
		return
	}
	var b struct {
		Email        string   `json:"email"`
		Roles        []string `json:"roles"`
		SubAccountID *string  `json:"sub_account_id"`
	}
	if !decodeBody(w, r, &b) {
		return
	}
	b.Email = strings.TrimSpace(b.Email)
	if !strings.Contains(b.Email, "@") || len(b.Email) > 254 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_email", "a valid email is required")
		return
	}
	if len(b.Roles) == 0 {
		b.Roles = []string{string(domain.RoleViewer)}
	}
	for _, rl := range b.Roles {
		if !domain.ValidRole(domain.Role(rl)) {
			writeErr(w, http.StatusUnprocessableEntity, "bad_role", "unknown role: "+rl)
			return
		}
	}
	raw, err := domain.NewVerifyToken()
	if err != nil {
		serverError(w, err)
		return
	}
	inv, err := s.st.CreateInvite(r.Context(), p.TenantID, b.Email, b.Roles, b.SubAccountID, domain.HashAPIKey(raw), p.UserID, inviteTTL)
	if err != nil {
		writeTeamErr(w, err)
		return
	}
	tenantName := "your team"
	if t, found, err := s.st.GetTenant(r.Context(), p.TenantID); err == nil && found {
		tenantName = t.Name
	}
	if err := s.mailer.SendInvite(inv.Email, tenantName, email.InviteURL(s.cfg.AppBaseURL, raw)); err != nil {
		slog.Error("send invite email", "err", err, "invite_id", inv.ID)
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: p.TenantID, ActorID: p.UserID, Action: "invite.create", TargetType: "invite", TargetID: inv.ID,
		After: map[string]any{"email": inv.Email, "roles": inv.Roles, "sub_account_id": inv.SubAccountID}, IsPaper: p.IsPaper,
	})
	out := inviteJSON(inv)
	if s.cfg.IsDev() && !s.mailer.Enabled() {
		out["dev_token"] = raw // no mail transport configured — keep the flow testable
	}
	writeJSON(w, http.StatusCreated, out)
}

// listInvites serves GET /v1/account/invites (admin): pending invitations.
func (s *Server) listInvites(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok || !requireRole(w, p, domain.RoleAdmin) {
		return
	}
	invs, err := s.st.ListInvites(r.Context(), p.TenantID)
	if err != nil {
		serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(invs))
	for _, inv := range invs {
		out = append(out, inviteJSON(inv))
	}
	writeJSON(w, http.StatusOK, map[string]any{"invites": out})
}

// revokeInvite serves DELETE /v1/account/invites/{id} (admin).
func (s *Server) revokeInvite(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok || !requireRole(w, p, domain.RoleAdmin) {
		return
	}
	inv, found, err := s.st.RevokeInvite(r.Context(), p.TenantID, r.PathValue("id"))
	if err != nil {
		serverError(w, err)
		return
	}
	if !found {
		writeErr(w, http.StatusNotFound, "not_found", "no such pending invitation")
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: p.TenantID, ActorID: p.UserID, Action: "invite.revoke", TargetType: "invite", TargetID: inv.ID,
		Before: map[string]any{"email": inv.Email, "roles": inv.Roles}, IsPaper: p.IsPaper,
	})
	w.WriteHeader(http.StatusNoContent)
}

// lookupInvite serves GET /v1/account/invites/lookup?token=: what the accept screen shows. The token
// is the credential; an unknown, used or expired one is 404.
func (s *Server) lookupInvite(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("token")
	if tok == "" || len(tok) > 256 {
		writeErr(w, http.StatusNotFound, "invite_invalid", "this invitation is invalid, used, revoked or expired")
		return
	}
	inv, err := s.st.LookupInvite(r.Context(), domain.HashAPIKey(tok))
	if err != nil {
		writeTeamErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"email": inv.Email, "tenant_name": inv.TenantName, "roles": inv.Roles,
		"expires_at": inv.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

// acceptInvite serves POST /v1/account/invites/accept {token, password}: creates the invited user
// (email verified) and signs them in.
func (s *Server) acceptInvite(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !decodeBody(w, r, &b) {
		return
	}
	if len(b.Password) < 8 || len(b.Password) > 72 {
		writeErr(w, http.StatusUnprocessableEntity, "weak_password", "the password must be 8 to 72 characters")
		return
	}
	if b.Token == "" || len(b.Token) > 256 {
		writeErr(w, http.StatusNotFound, "invite_invalid", "this invitation is invalid, used, revoked or expired")
		return
	}
	hash, err := domain.HashPassword(b.Password)
	if err != nil {
		serverError(w, err)
		return
	}
	u, inv, err := s.st.AcceptInvite(r.Context(), domain.HashAPIKey(b.Token), hash)
	if err != nil {
		writeTeamErr(w, err)
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: u.TenantID, ActorID: u.UserID, Action: "invite.accept", TargetType: "user", TargetID: u.UserID,
		After: map[string]any{"email": inv.Email, "roles": inv.Roles, "invite_id": inv.ID}, IsPaper: u.IsPaper,
	})
	s.issue(w, http.StatusCreated, u.UserID, domain.Claims{
		TenantID: u.TenantID, SubAccountID: u.SubAccountID, Roles: u.Roles, IsPaper: u.IsPaper,
	})
}

// subAccountJSON renders a sub-account.
func subAccountJSON(sa store.SubAccount) map[string]any {
	return map[string]any{"id": sa.ID, "name": sa.Name, "members": sa.Members, "created_at": sa.CreatedAt.UTC().Format(time.RFC3339)}
}

// createSubAccount serves POST /v1/account/sub-accounts (admin).
func (s *Server) createSubAccount(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok || !requireRole(w, p, domain.RoleAdmin) {
		return
	}
	var b struct {
		Name string `json:"name"`
	}
	if !decodeBody(w, r, &b) {
		return
	}
	b.Name = strings.TrimSpace(b.Name)
	if b.Name == "" || len(b.Name) > 80 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "name (1-80 characters) is required")
		return
	}
	sa, err := s.st.CreateSubAccount(r.Context(), p.TenantID, b.Name)
	if err != nil {
		writeTeamErr(w, err)
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: p.TenantID, ActorID: p.UserID, Action: "sub_account.create", TargetType: "sub_account", TargetID: sa.ID,
		After: map[string]any{"name": sa.Name}, IsPaper: p.IsPaper,
	})
	writeJSON(w, http.StatusCreated, subAccountJSON(sa))
}

// listSubAccounts serves GET /v1/account/sub-accounts.
func (s *Server) listSubAccounts(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	list, err := s.st.ListSubAccounts(r.Context(), p.TenantID)
	if err != nil {
		serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, sa := range list {
		out = append(out, subAccountJSON(sa))
	}
	writeJSON(w, http.StatusOK, map[string]any{"sub_accounts": out})
}

// transferSubAccount serves POST /v1/account/sub-accounts/{id}/transfer (admin or billing): fund the
// sub-account from the tenant's main balance, or return credits to it. Idempotent on the key.
func (s *Server) transferSubAccount(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok || !requireRole(w, p, domain.RoleBilling) {
		return
	}
	if s.transfer == nil {
		writeErr(w, http.StatusServiceUnavailable, "ledger_unavailable", "the credit ledger is not configured")
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 128 {
		writeErr(w, http.StatusBadRequest, "bad_request", "an Idempotency-Key (at most 128 chars) is required")
		return
	}
	var b struct {
		CreditType string `json:"credit_type"`
		Amount     string `json:"amount"`
		Direction  string `json:"direction"` // fund | return
	}
	if !decodeBody(w, r, &b) {
		return
	}
	if b.Direction != "fund" && b.Direction != "return" {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "direction must be fund or return")
		return
	}
	if !domain.ValidAmount(b.Amount) || !domain.ValidCreditType(b.CreditType) {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "a known credit_type and a positive amount (at most 6 decimals) are required")
		return
	}
	sa, err := s.st.GetSubAccount(r.Context(), p.TenantID, r.PathValue("id"))
	if err != nil {
		writeTeamErr(w, err)
		return
	}
	t := billing.Transfer{TenantID: p.TenantID, CreditType: b.CreditType, Amount: b.Amount, IsPaper: p.IsPaper}
	if b.Direction == "fund" {
		t.ToSub = &sa.ID
	} else {
		t.FromSub = &sa.ID
	}
	// The key is scoped to the sub-account and direction so a reused key cannot move credits elsewhere.
	err = s.transfer.Transfer(r.Context(), t, "sub:"+sa.ID+":"+b.Direction+":"+key)
	if errors.Is(err, billing.ErrInsufficientCredit) {
		writeErr(w, http.StatusPaymentRequired, "insufficient_credit", "not enough credits in the balance you are moving from")
		return
	}
	if err != nil {
		slog.Error("sub-account transfer", "err", err)
		writeErr(w, http.StatusBadGateway, "ledger_error", "the credit ledger did not complete the transfer; retry with the same key")
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: p.TenantID, ActorID: p.UserID, Action: "sub_account." + b.Direction, TargetType: "sub_account", TargetID: sa.ID,
		After: map[string]any{"credit_type": b.CreditType, "amount": b.Amount}, IsPaper: p.IsPaper,
	})
	writeJSON(w, http.StatusOK, map[string]any{"sub_account_id": sa.ID, "direction": b.Direction, "credit_type": b.CreditType, "amount": b.Amount})
}
