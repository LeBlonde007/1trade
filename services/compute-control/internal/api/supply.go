package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/trade1/compute-control/internal/pool"
	"github.com/trade1/compute-control/internal/supply"
)

// SupplyDeps is what the supply endpoints need (supply.yaml v1.0). Without it (no DATABASE_URL) the
// endpoints answer 503: partner capacity cannot live in memory.
type SupplyDeps struct {
	Store *supply.Store
	Pool  *pool.Pool
	Sync  *supply.Syncer
}

// EnableSupply turns the supply endpoints on.
func (s *Server) EnableSupply(d *SupplyDeps) { s.supply = d }

// supplyRoutes registers the supply endpoints.
func (s *Server) supplyRoutes() {
	s.mux.HandleFunc("POST /v1/supply/sources", s.registerSource)
	s.mux.HandleFunc("GET /v1/supply/sources", s.listSources)
	s.mux.HandleFunc("GET /v1/supply/sources/{id}", s.getSource)
	s.mux.HandleFunc("DELETE /v1/supply/sources/{id}", s.sourceAction("retire"))
	s.mux.HandleFunc("POST /v1/supply/sources/{id}/suspend", s.sourceAction("suspend"))
	s.mux.HandleFunc("POST /v1/supply/sources/{id}/resume", s.sourceAction("resume"))
	s.mux.HandleFunc("POST /v1/supply/sources/{id}/activate", s.sourceAction("activate"))
	s.mux.HandleFunc("POST /v1/supply/sources/{id}/heartbeat", s.heartbeat)
	s.mux.HandleFunc("GET /v1/supply/sources/{id}/usage", s.sourceUsage)
}

// supplyCaller authorises a supply call. The service token is operations (partner ""); a tenant JWT
// needs the admin or engineer role and acts on its own sources only. ok=false means a response was
// written.
func (s *Server) supplyCaller(w http.ResponseWriter, r *http.Request) (partner, actor string, ok bool) {
	if s.supply == nil {
		writeErr(w, http.StatusServiceUnavailable, "supply_unavailable", "partner supply needs the registry database")
		return "", "", false
	}
	if b := bearer(r); s.auth.IsService(b) {
		return "", "service", true
	}
	p, err := s.auth.ResolveJWT(bearer(r))
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "a valid tenant token is required")
		return "", "", false
	}
	if !p.HasRole("engineer") {
		writeErr(w, http.StatusForbidden, "forbidden", "managing supply needs the admin or engineer role")
		return "", "", false
	}
	return p.TenantID, p.TenantID, true
}

// resync mirrors the registry into the pool now, so a suspension or activation takes effect at once
// rather than at the next tick. Best effort: the periodic sync catches up.
func (s *Server) resync(ctx context.Context) {
	if err := s.supply.Sync.Sync(ctx); err != nil {
		slog.Warn("supply resync failed", "err", err)
	}
}

// inUse maps source id → GPUs reserved on it, from the live pool.
func (s *Server) inUse() map[string]int {
	out := map[string]int{}
	for _, st := range s.supply.Pool.Sources() {
		for _, n := range st.InUse {
			out[st.ID] += n
		}
	}
	return out
}

// sourceJSON renders a registry source in the contract shape.
func sourceJSON(src supply.Source, inUse int, now time.Time) map[string]any {
	return map[string]any{
		"id": src.ID, "partner_tenant_id": src.PartnerTenantID, "name": src.Name, "gpu_type": src.GPUType,
		"gpu_count": src.GPUCount, "region": src.Region, "sla_tier": src.SLATier, "state": src.State,
		"schedulable": supply.Schedulable(src, now), "gpus_healthy": src.GPUsHealthy, "gpus_in_use": inUse,
		"utilization_pct": src.UtilizationPct, "last_heartbeat_at": src.LastHeartbeatAt, "created_at": src.CreatedAt,
	}
}

// writeSupplyErr maps registry errors onto the contract's statuses.
func writeSupplyErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, supply.ErrNotFound):
		writeErr(w, http.StatusNotFound, "not_found", "no such source")
	case errors.Is(err, supply.ErrState):
		writeErr(w, http.StatusConflict, "invalid_state", "not allowed in the source's current state")
	case errors.Is(err, supply.ErrIdemConflict):
		writeErr(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Idempotency-Key reused with a different body")
	default:
		slog.Error("supply request failed", "err", err)
		writeErr(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
	}
}

// registerSource serves POST /v1/supply/sources (a partner registers capacity; it starts pending).
func (s *Server) registerSource(w http.ResponseWriter, r *http.Request) {
	partner, _, ok := s.supplyCaller(w, r)
	if !ok {
		return
	}
	if partner == "" {
		writeErr(w, http.StatusForbidden, "forbidden", "sources are registered by a partner tenant")
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 255 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "an Idempotency-Key (at most 255 chars) is required")
		return
	}
	var b struct {
		Name     string `json:"name"`
		GPUType  string `json:"gpu_type"`
		GPUCount int    `json:"gpu_count"`
		Region   string `json:"region"`
		SLATier  string `json:"sla_tier"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&b); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "invalid JSON body")
		return
	}
	b.Name, b.Region = strings.TrimSpace(b.Name), strings.TrimSpace(b.Region)
	switch {
	case b.Name == "" || len(b.Name) > 80, b.Region == "" || len(b.Region) > 64:
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "name (1-80) and region (1-64) are required")
		return
	case b.GPUType != "gpu_h100" && b.GPUType != "gpu_h200":
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "gpu_type must be gpu_h100 or gpu_h200")
		return
	case b.GPUCount < 1 || b.GPUCount > 4096:
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "gpu_count must be 1-4096")
		return
	case b.SLATier != "bronze" && b.SLATier != "silver" && b.SLATier != "gold":
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "sla_tier must be bronze, silver or gold")
		return
	}
	src, _, err := s.supply.Store.Register(r.Context(), partner, key, supply.Registration{
		Name: b.Name, GPUType: b.GPUType, GPUCount: b.GPUCount, Region: b.Region, SLATier: b.SLATier})
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sourceJSON(src, 0, time.Now()))
}

// listSources serves GET /v1/supply/sources: the caller's own, or (operations) every registry source
// plus 1Trade's own capacity, which lives in config rather than the registry.
func (s *Server) listSources(w http.ResponseWriter, r *http.Request) {
	partner, _, ok := s.supplyCaller(w, r)
	if !ok {
		return
	}
	srcs, err := s.supply.Store.List(r.Context(), partner)
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	now, used := time.Now(), s.inUse()
	data := make([]map[string]any, 0, len(srcs)+1)
	known := map[string]bool{}
	for _, src := range srcs {
		known[src.ID] = true
		data = append(data, sourceJSON(src, used[src.ID], now))
	}
	if partner == "" {
		for _, st := range s.supply.Pool.Sources() {
			if known[st.ID] {
				continue
			}
			for tier, n := range st.Capacity {
				data = append(data, map[string]any{"id": st.ID, "partner_tenant_id": nil, "name": "1Trade owned",
					"gpu_type": tier, "gpu_count": n, "state": supply.Active, "schedulable": st.Accepting,
					"gpus_in_use": used[st.ID]})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// getSource serves GET /v1/supply/sources/{id}.
func (s *Server) getSource(w http.ResponseWriter, r *http.Request) {
	partner, _, ok := s.supplyCaller(w, r)
	if !ok {
		return
	}
	src, err := s.supply.Store.Get(r.Context(), r.PathValue("id"), partner)
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sourceJSON(src, s.inUse()[src.ID], time.Now()))
}

// sourceAction serves suspend / resume / retire (the partner or operations) and activate (operations
// only), then resyncs the pool.
func (s *Server) sourceAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partner, actor, ok := s.supplyCaller(w, r)
		if !ok {
			return
		}
		if action == "activate" && partner != "" {
			writeErr(w, http.StatusForbidden, "forbidden", "activation is done by 1Trade operations")
			return
		}
		src, err := s.supply.Store.Transition(r.Context(), r.PathValue("id"), partner, action, actor)
		if err != nil {
			writeSupplyErr(w, err)
			return
		}
		s.resync(r.Context())
		writeJSON(w, http.StatusOK, sourceJSON(src, s.inUse()[src.ID], time.Now()))
	}
}

// heartbeat serves POST /v1/supply/sources/{id}/heartbeat (the partner's agent).
func (s *Server) heartbeat(w http.ResponseWriter, r *http.Request) {
	partner, _, ok := s.supplyCaller(w, r)
	if !ok {
		return
	}
	var b struct {
		GPUsHealthy    *int  `json:"gpus_healthy"`
		UtilizationPct int   `json:"utilization_pct"`
		ECCErrors      int64 `json:"ecc_errors"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&b); err != nil || b.GPUsHealthy == nil ||
		*b.GPUsHealthy < 0 || b.UtilizationPct < 0 || b.UtilizationPct > 100 || b.ECCErrors < 0 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "gpus_healthy (>=0) is required; utilization_pct 0-100; ecc_errors >= 0")
		return
	}
	src, err := s.supply.Store.Heartbeat(r.Context(), r.PathValue("id"), partner, *b.GPUsHealthy, b.UtilizationPct, b.ECCErrors)
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	s.resync(r.Context())
	writeJSON(w, http.StatusOK, sourceJSON(src, s.inUse()[src.ID], time.Now()))
}

// sourceUsage serves GET /v1/supply/sources/{id}/usage: metered GPU time per tier in [from, to).
func (s *Server) sourceUsage(w http.ResponseWriter, r *http.Request) {
	partner, _, ok := s.supplyCaller(w, r)
	if !ok {
		return
	}
	src, err := s.supply.Store.Get(r.Context(), r.PathValue("id"), partner)
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	to, from := time.Now().UTC(), time.Now().UTC().Add(-30*24*time.Hour)
	for _, q := range []struct {
		name string
		dst  *time.Time
	}{{"from", &from}, {"to", &to}} {
		if v := r.URL.Query().Get(q.name); v != "" {
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				writeErr(w, http.StatusUnprocessableEntity, "bad_request", q.name+" must be RFC 3339")
				return
			}
			*q.dst = t
		}
	}
	if !from.Before(to) || to.Sub(from) > 366*24*time.Hour {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "from must be before to, at most 366 days apart")
		return
	}
	rows, err := s.supply.Store.UsageFor(r.Context(), src.ID, from, to)
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	tiers := make([]map[string]any, 0, len(rows))
	for _, u := range rows {
		tiers = append(tiers, map[string]any{"gpu_type": u.GPUType, "gpu_seconds": u.GPUSeconds, "units": u.Units, "sessions": u.Sessions})
	}
	writeJSON(w, http.StatusOK, map[string]any{"source_id": src.ID, "from": from, "to": to, "by_tier": tiers})
}
