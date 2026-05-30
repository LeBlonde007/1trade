package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/exascale/platform-core/internal/domain"
	"github.com/exascale/platform-core/internal/store"
)

// listAudit returns the tenant's audit log, newest first (admin-only). The queryable audit trail is
// an F03 acceptance criterion and a SOC 2 control.
func (s *Server) listAudit(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	if !requireRole(w, p, domain.RoleAdmin) {
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 500 {
			limit = n
		}
	}
	rows, err := s.st.ListAudit(r.Context(), p.TenantID, limit)
	if err != nil {
		serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, a := range rows {
		row := map[string]any{
			"id": a.ID, "action": a.Action, "target_type": a.TargetType, "target_id": a.TargetID,
			"is_paper": a.IsPaper, "created_at": a.CreatedAt.UTC().Format(time.RFC3339),
		}
		if a.ActorID != nil {
			row["actor_id"] = *a.ActorID
		}
		if len(a.Before) > 0 && string(a.Before) != "null" {
			row["before"] = json.RawMessage(a.Before)
		}
		if len(a.After) > 0 && string(a.After) != "null" {
			row["after"] = json.RawMessage(a.After)
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": out})
}

// createOrg creates an org under the caller's tenant (admin-only); audited.
func (s *Server) createOrg(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	if !requireRole(w, p, domain.RoleAdmin) {
		return
	}
	var b struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Name == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "name is required")
		return
	}
	o, err := s.st.CreateOrg(r.Context(), p.TenantID, b.Name)
	if err != nil {
		serverError(w, err)
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: p.TenantID, ActorID: p.UserID, Action: "org.create",
		TargetType: "org", TargetID: o.ID, After: map[string]any{"name": o.Name}, IsPaper: p.IsPaper,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"id": o.ID, "name": o.Name, "created_at": o.CreatedAt.UTC().Format(time.RFC3339)})
}

// listOrgs lists the tenant's orgs.
func (s *Server) listOrgs(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	orgs, err := s.st.ListOrgs(r.Context(), p.TenantID)
	if err != nil {
		serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(orgs))
	for _, o := range orgs {
		out = append(out, map[string]any{"id": o.ID, "name": o.Name, "created_at": o.CreatedAt.UTC().Format(time.RFC3339)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"orgs": out})
}

// assignRoles sets a user's roles within the tenant (admin-only). Validates roles; audits the diff.
func (s *Server) assignRoles(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	if !requireRole(w, p, domain.RoleAdmin) {
		return
	}
	userID := r.PathValue("id")
	var b struct {
		Roles []string `json:"roles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	for _, rl := range b.Roles {
		if !domain.ValidRole(domain.Role(rl)) {
			writeErr(w, http.StatusUnprocessableEntity, "bad_role", "unknown role: "+rl)
			return
		}
	}
	prev, found, err := s.st.AssignRoles(r.Context(), p.TenantID, userID, b.Roles)
	if err != nil {
		serverError(w, err)
		return
	}
	if !found {
		writeErr(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	_, _ = s.st.WriteAudit(r.Context(), store.AuditEntry{
		TenantID: p.TenantID, ActorID: p.UserID, Action: "user.roles.assign",
		TargetType: "user", TargetID: userID,
		Before: map[string]any{"roles": prev}, After: map[string]any{"roles": b.Roles}, IsPaper: p.IsPaper,
	})
	writeJSON(w, http.StatusOK, map[string]any{"id": userID, "roles": b.Roles})
}

// getTenant returns the caller's own tenant. Cross-tenant reads are refused (404, no info leak).
func (s *Server) getTenant(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if id != p.TenantID {
		writeErr(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	t, found, err := s.st.GetTenant(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}
	if !found {
		writeErr(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": t.ID, "name": t.Name, "kind": t.Kind, "is_paper": t.IsPaper, "created_at": t.CreatedAt.UTC().Format(time.RFC3339),
	})
}

// listOrgUsers lists the users in an org within the caller's tenant.
func (s *Server) listOrgUsers(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	users, err := s.st.ListOrgUsers(r.Context(), p.TenantID, r.PathValue("id"))
	if err != nil {
		serverError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, map[string]any{"id": u.ID, "email": u.Email, "roles": u.Roles})
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": out})
}
