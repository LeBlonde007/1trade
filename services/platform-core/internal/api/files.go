package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// presignReq is the body for a presigned-upload request.
type presignReq struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

// presignUpload signs a short-lived URL that the browser PUTs a file to directly (DigitalOcean Spaces),
// and returns the object's eventual public URL. Authed; keys are tenant-scoped so one tenant can't
// overwrite another's objects. 501 when storage isn't configured (the platform runs fine without it).
func (s *Server) presignUpload(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	if !s.storage.Enabled() {
		writeErr(w, http.StatusNotImplemented, "storage_disabled", "file storage is not configured")
		return
	}
	var b presignReq
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Filename == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "filename is required")
		return
	}
	out, err := s.storage.PresignUpload(r.Context(), p.TenantID, b.Filename, 10*time.Minute)
	if err != nil {
		serverError(w, err)
		return
	}
	slog.Info("audit: presigned upload", "tenant_id", p.TenantID, "actor_id", p.UserID, "key", out.Key)
	writeJSON(w, http.StatusOK, out)
}
