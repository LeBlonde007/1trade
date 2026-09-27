// Package api is the surveillance review API: internal, service-token only. Alerts name tenants and
// describe suspected abuse, so nothing here is ever customer-facing.
package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/trade1/surveillance/internal/detect"
	"github.com/trade1/surveillance/internal/store"
)

// Lister is what the API needs from the store.
type Lister interface {
	ListAlerts(ctx context.Context, f store.Filter) ([]detect.Alert, error)
	Ping(ctx context.Context) error
}

// New returns the routed handler. An empty token refuses every alerts request.
func New(st Lister, serviceToken string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := st.Ping(r.Context()); err != nil {
			writeErr(w, http.StatusServiceUnavailable, "not_ready", "database unreachable")
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /v1/surveillance/alerts", func(w http.ResponseWriter, r *http.Request) {
		tok, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if serviceToken == "" || subtle.ConstantTimeCompare([]byte(tok), []byte(serviceToken)) != 1 {
			writeErr(w, http.StatusUnauthorized, "unauthorized", "service authorization required")
			return
		}
		q := r.URL.Query()
		f := store.Filter{Rule: q.Get("rule"), TenantID: q.Get("tenant_id"), ProductID: q.Get("product_id")}
		if v := q.Get("is_paper"); v != "" {
			b := v == "true"
			f.IsPaper = &b
		}
		if v := q.Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > 500 {
				writeErr(w, http.StatusUnprocessableEntity, "bad_request", "limit must be an integer from 1 to 500")
				return
			}
			f.Limit = n
		}
		alerts, err := st.ListAlerts(r.Context(), f)
		if err != nil {
			slog.Error("list alerts", "err", err)
			writeErr(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"alerts": alerts})
	})
	return mux
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr writes the platform's ApiError shape.
func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"code": code, "message": msg})
}
