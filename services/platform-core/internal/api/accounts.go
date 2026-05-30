package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/exascale/platform-core/internal/domain"
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
