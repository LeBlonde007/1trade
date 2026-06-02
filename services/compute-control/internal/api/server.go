// Package api is the HTTP surface of the compute control plane (docs/contracts/openapi/compute.yaml):
// a read-only GPU-type catalog + per-tenant quota for customers, and the internal scheduling surface
// (submit / get / cancel jobs) the inference gateway+runtime use to place gang-scheduled pods. Error
// bodies use the contract's ApiError shape ({code,message}); internals are logged, never leaked.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/exascale/compute-control/internal/auth"
	"github.com/exascale/compute-control/internal/config"
	"github.com/exascale/compute-control/internal/domain"
	"github.com/exascale/compute-control/internal/scheduler"
)

// Server wires config + the credential resolver + the scheduler backend behind one routed handler.
type Server struct {
	cfg   config.Config
	auth  *auth.Resolver
	sched scheduler.Scheduler
	mux   *http.ServeMux
}

// New builds the routed handler.
func New(cfg config.Config, resolver *auth.Resolver, sched scheduler.Scheduler) *Server {
	s := &Server{cfg: cfg, auth: resolver, sched: sched, mux: http.NewServeMux()}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// routes registers every endpoint from the contract.
func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	s.mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	s.mux.HandleFunc("GET /v1/compute/types", s.listTypes)
	s.mux.HandleFunc("GET /v1/compute/quota", s.getQuota)
	s.mux.HandleFunc("GET /v1/compute/jobs", s.listJobs)
	s.mux.HandleFunc("POST /v1/compute/jobs", s.submitJob)
	s.mux.HandleFunc("GET /v1/compute/jobs/{id}", s.getJob)
	s.mux.HandleFunc("DELETE /v1/compute/jobs/{id}", s.cancelJob)
	s.mux.HandleFunc("GET /v1/compute/instances", s.listInstances)
}

// listTypes serves the GPU-tier catalog with live availability. Public read (no auth) — it leaks no
// tenant data and the catalog drives the marketing/console pricing view.
func (s *Server) listTypes(w http.ResponseWriter, _ *http.Request) {
	h100, h200 := s.sched.Capacity()
	types := make([]map[string]any, 0)
	for _, t := range domain.Catalog(h100, h200) {
		types = append(types, map[string]any{
			"id": t.ID, "name": t.Name, "gpu": t.GPU,
			"credit_type": t.CreditType, "price_per_hour": t.PricePerHour, "available": t.Available,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"types": types})
}

// getQuota serves the authenticated tenant's per-tier GPU quota + usage (tenant JWT required).
func (s *Server) getQuota(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireTenant(w, r)
	if !ok {
		return
	}
	quotas := make([]map[string]any, 0)
	for _, q := range s.sched.Quotas(p.TenantID) {
		quotas = append(quotas, map[string]any{
			"credit_type":     q.GPUType,
			"limit_hours":     itoaDecimal(q.Capacity),
			"used_hours":      itoaDecimal(q.InUse),
			"reserved_hours":  itoaDecimal(0),
			"remaining_hours": itoaDecimal(q.RemainingPool),
			"is_paper":        p.IsPaper,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"quotas": quotas})
}

// listJobs lists the tenant's jobs, newest first (tenant JWT required).
func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireTenant(w, r)
	if !ok {
		return
	}
	jobs := make([]map[string]any, 0)
	for _, j := range s.sched.List(p.TenantID) {
		jobs = append(jobs, jobJSON(j))
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

// jobRequest mirrors the contract's JobRequest.
type jobRequest struct {
	WorkloadClass string  `json:"workload_class"`
	GPUType       string  `json:"gpu_type"`
	GPUs          int     `json:"gpus"`
	Pods          int     `json:"pods"`
	Reserved      bool    `json:"reserved"`
	SubAccountID  *string `json:"sub_account_id"`
	ReferenceID   string  `json:"reference_id"`
}

// submitJob places a workload (internal — service token, or a tenant JWT). The acting tenant +
// is_paper come from the principal: a JWT carries them directly; a service-token caller names the
// tenant via X-Tenant-Id (+ optional X-Is-Paper) — never the request body, so is_paper can't be
// spoofed by a customer. Idempotent on the required Idempotency-Key header.
func (s *Server) submitJob(w http.ResponseWriter, r *http.Request) {
	idem := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idem == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "Idempotency-Key header is required")
		return
	}
	tenantID, subAccount, isPaper, ok := s.resolveActor(w, r)
	if !ok {
		return
	}
	var req jobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Pods < 1 {
		req.Pods = 1
	}
	if req.SubAccountID != nil && *req.SubAccountID != "" {
		subAccount = *req.SubAccountID
	}
	spec := scheduler.JobSpec{
		TenantID: tenantID, SubAccountID: subAccount, IsPaper: isPaper,
		WorkloadClass: req.WorkloadClass, GPUType: req.GPUType,
		GPUs: req.GPUs, Pods: req.Pods, Reserved: req.Reserved, ReferenceID: req.ReferenceID,
	}
	job, err := s.sched.Submit(spec, idem)
	if err != nil {
		writeSchedErr(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, jobJSON(job))
}

// getJob serves one job, scoped to the tenant (tenant JWT required).
func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireTenant(w, r)
	if !ok {
		return
	}
	job, err := s.sched.Get(p.TenantID, r.PathValue("id"))
	if err != nil {
		writeSchedErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, jobJSON(job))
}

// cancelJob cancels/tears down one of the tenant's jobs (tenant JWT or service token).
func (s *Server) cancelJob(w http.ResponseWriter, r *http.Request) {
	tenantID, _, _, ok := s.resolveActor(w, r)
	if !ok {
		return
	}
	job, err := s.sched.Cancel(tenantID, r.PathValue("id"))
	if err != nil {
		writeSchedErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, jobJSON(job))
}

// listInstances serves the tenant's live placements as minimal instances (M3 expands this).
func (s *Server) listInstances(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requireTenant(w, r)
	if !ok {
		return
	}
	instances := make([]map[string]any, 0)
	for _, j := range s.sched.Instances(p.TenantID) {
		instances = append(instances, map[string]any{
			"id": j.ID, "gpu_type": j.GPUType, "status": "running",
			"created_at": j.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"instances": instances})
}

// requireTenant resolves a first-party tenant JWT; on failure it writes a 401 and returns ok=false.
func (s *Server) requireTenant(w http.ResponseWriter, r *http.Request) (auth.Principal, bool) {
	p, err := s.auth.ResolveJWT(bearer(r))
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "a valid tenant token is required")
		return auth.Principal{}, false
	}
	return p, true
}

// resolveActor determines the tenant a mutating call acts on. A service token names the tenant via
// X-Tenant-Id (+ optional X-Is-Paper, default config); a tenant JWT supplies its own claims. Writes a
// 401/400 and returns ok=false on failure.
func (s *Server) resolveActor(w http.ResponseWriter, r *http.Request) (tenantID, subAccount string, isPaper, ok bool) {
	b := bearer(r)
	if s.auth.IsService(b) {
		tid := strings.TrimSpace(r.Header.Get("X-Tenant-Id"))
		if tid == "" {
			writeErr(w, http.StatusBadRequest, "bad_request", "X-Tenant-Id is required for service calls")
			return "", "", false, false
		}
		paper := s.cfg.Paper
		if v := r.Header.Get("X-Is-Paper"); v != "" {
			paper = v != "false"
		}
		return tid, "", paper, true
	}
	p, err := s.auth.ResolveJWT(b)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized", "a valid tenant token or service token is required")
		return "", "", false, false
	}
	return p.TenantID, p.SubAccountID, p.IsPaper, true
}

// jobJSON renders a Job in the contract shape.
func jobJSON(j domain.Job) map[string]any {
	var placement any
	if j.Placement != "" {
		placement = j.Placement
	}
	return map[string]any{
		"id": j.ID, "status": j.Status, "workload_class": j.WorkloadClass,
		"gpu_type": j.GPUType, "gpus": j.GPUs, "pods": j.Pods, "reserved": j.Reserved,
		"supply_source_id": j.SupplySourceID, "placement": placement, "is_paper": j.IsPaper,
		"created_at": j.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// itoaDecimal renders an integer GPU count as a fixed-point 6dp decimal string (contract Decimal).
func itoaDecimal(n int) string { return strconv.Itoa(n) + ".000000" }

// bearer extracts the token from an Authorization: Bearer <token> header.
func bearer(r *http.Request) string {
	if after, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}

// writeSchedErr maps a scheduler error to the right HTTP status + ApiError code.
func writeSchedErr(w http.ResponseWriter, err error) {
	switch err {
	case scheduler.ErrNotFound, scheduler.ErrForbidden:
		writeErr(w, http.StatusNotFound, "not_found", "job not found")
	case scheduler.ErrCapacity:
		writeErr(w, http.StatusPaymentRequired, "quota_exhausted", "insufficient GPU capacity for this tier")
	case scheduler.ErrUnknownGPU, scheduler.ErrUnknownClass, scheduler.ErrBadRequest:
		writeErr(w, http.StatusUnprocessableEntity, "invalid_request", err.Error())
	case scheduler.ErrNotCancelable:
		writeErr(w, http.StatusConflict, "conflict", "job is not in a cancelable state")
	default:
		slog.Error("scheduler error", "err", err)
		writeErr(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
	}
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr writes the contract's ApiError shape ({code, message}).
func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]string{"code": code, "message": msg})
}
