package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/trade1/compute-control/internal/pool"
	"github.com/trade1/compute-control/internal/reserve"
)

// EnableReservations turns on reserved capacity (F14). Without it (no DATABASE_URL or ledger) the
// endpoints answer 503: a prepaid commitment cannot live in memory.
func (s *Server) EnableReservations(svc *reserve.Service) { s.reserve = svc }

// reservationRoutes registers the reservation endpoints (compute.yaml v1.2).
func (s *Server) reservationRoutes() {
	s.mux.HandleFunc("GET /v1/compute/reservations/quote", s.quoteReservation)
	s.mux.HandleFunc("POST /v1/compute/reservations", s.buyReservation)
	s.mux.HandleFunc("GET /v1/compute/reservations", s.listReservations)
	s.mux.HandleFunc("GET /v1/compute/reservations/{id}", s.getReservation)
}

// reservationTenant authorises a reservation call with a tenant JWT. ok=false means a response was
// written.
func (s *Server) reservationTenant(w http.ResponseWriter, r *http.Request) (tenant, sub string, isPaper, billing, ok bool) {
	if s.reserve == nil {
		writeErr(w, http.StatusServiceUnavailable, "reservations_unavailable", "reserved capacity needs the database and the ledger")
		return "", "", false, false, false
	}
	p, ok := s.requireTenant(w, r)
	if !ok {
		return "", "", false, false, false
	}
	return p.TenantID, p.SubAccountID, p.IsPaper, p.HasRole("billing"), true
}

// writeReserveErr maps reservation errors onto the contract's statuses.
func writeReserveErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, reserve.ErrBadRequest):
		writeErr(w, http.StatusUnprocessableEntity, "invalid_request", strings.TrimPrefix(err.Error(), "reserve: invalid reservation: "))
	case errors.Is(err, reserve.ErrNoCapacity):
		writeErr(w, http.StatusConflict, "capacity_unavailable", "not enough free GPUs of this tier to set aside for the term")
	case errors.Is(err, reserve.ErrInsufficient):
		writeErr(w, http.StatusPaymentRequired, "insufficient_credit", "not enough GPU credits of this tier to prepay the reservation")
	case errors.Is(err, reserve.ErrPaymentPending):
		writeErr(w, http.StatusServiceUnavailable, "payment_pending", "payment is not confirmed yet; retry with the same Idempotency-Key")
	case errors.Is(err, reserve.ErrIdemConflict):
		writeErr(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Idempotency-Key reused with a different body")
	case errors.Is(err, reserve.ErrNotFound):
		writeErr(w, http.StatusNotFound, "not_found", "no such reservation")
	default:
		slog.Error("reservation request failed", "err", err)
		writeErr(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
	}
}

// quoteReservation serves GET /v1/compute/reservations/quote?gpu_type=&gpus=&term= — the price.
func (s *Server) quoteReservation(w http.ResponseWriter, r *http.Request) {
	if _, _, _, _, ok := s.reservationTenant(w, r); !ok {
		return
	}
	q := r.URL.Query()
	gpus, err := strconv.Atoi(q.Get("gpus"))
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "invalid_request", "gpus must be an integer")
		return
	}
	quote, err := reserve.NewQuote(q.Get("gpu_type"), gpus, q.Get("term"))
	if err != nil {
		writeReserveErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"quote": quote, "available": s.inst.Available(quote.GPUType)})
}

// buyReservation serves POST /v1/compute/reservations: prepay and set aside capacity (billing role).
func (s *Server) buyReservation(w http.ResponseWriter, r *http.Request) {
	tenant, sub, isPaper, billing, ok := s.reservationTenant(w, r)
	if !ok {
		return
	}
	if !billing {
		writeErr(w, http.StatusForbidden, "forbidden", "buying reserved capacity needs the admin or billing role")
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 255 {
		writeErr(w, http.StatusUnprocessableEntity, "invalid_request", "an Idempotency-Key (at most 255 chars) is required")
		return
	}
	var b struct {
		GPUType string `json:"gpu_type"`
		GPUs    int    `json:"gpus"`
		Term    string `json:"term"`
	}
	if !decodeStrict(w, r, &b) {
		return
	}
	quote, err := reserve.NewQuote(b.GPUType, b.GPUs, b.Term)
	if err != nil {
		writeReserveErr(w, err)
		return
	}
	res, err := s.reserve.Buy(r.Context(), tenant, sub, isPaper, key, quote)
	if err == nil && res.State == reserve.Failed {
		err = reserve.ErrInsufficient // a replay of a purchase that could not be paid
	}
	if err != nil {
		writeReserveErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

// listReservations serves GET /v1/compute/reservations: the tenant's reservations, and per tier how
// many reserved GPUs its running instances occupy right now.
func (s *Server) listReservations(w http.ResponseWriter, r *http.Request) {
	tenant, _, isPaper, _, ok := s.reservationTenant(w, r)
	if !ok {
		return
	}
	list, err := s.reserve.List(r.Context(), tenant)
	if err != nil {
		writeReserveErr(w, err)
		return
	}
	holds := s.inst.Holds()
	capacity := make([]map[string]any, 0, 2)
	for _, tier := range []string{"gpu_h100", "gpu_h200"} {
		h := holds[pool.HoldKey{Tenant: tenant, IsPaper: isPaper, Tier: tier}]
		if h[0] > 0 || h[1] > 0 {
			capacity = append(capacity, map[string]any{"gpu_type": tier, "is_paper": isPaper, "reserved_gpus": h[0], "in_use_gpus": h[1]})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": list, "capacity": capacity})
}

// getReservation serves GET /v1/compute/reservations/{id}.
func (s *Server) getReservation(w http.ResponseWriter, r *http.Request) {
	tenant, _, _, _, ok := s.reservationTenant(w, r)
	if !ok {
		return
	}
	res, err := s.reserve.Get(r.Context(), tenant, r.PathValue("id"))
	if err != nil {
		writeReserveErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
