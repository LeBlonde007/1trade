package api

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/trade1/compute-control/internal/supply"
)

// clock is the server's time source.
func (s *Server) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// operator allows only the service token (operations). ok=false means a response was written.
func (s *Server) operator(w http.ResponseWriter, r *http.Request) bool {
	partner, _, ok := s.supplyCaller(w, r)
	if !ok {
		return false
	}
	if partner != "" {
		writeErr(w, http.StatusForbidden, "forbidden", "this is done by 1Trade operations")
		return false
	}
	return true
}

// payoutJSON renders a statement in the contract shape.
func payoutJSON(p supply.Payout) map[string]any {
	lines := p.Lines
	if lines == nil {
		lines = []supply.Line{}
	}
	secs, _ := newRat(p.GPUSeconds)
	return map[string]any{
		"id": p.ID, "partner_tenant_id": p.PartnerTenantID, "period_start": p.PeriodStart, "period_end": p.PeriodEnd,
		"currency": "USD", "is_paper": p.IsPaper, "gpu_hours": hours(secs), "gross": p.Gross, "fee": p.Fee,
		"payout": p.Amount, "holdback": p.Holdback, "released": p.Released, "usage_records": p.UsageRecords,
		"state": p.State, "wire_reference": p.WireReference, "dispute_until": p.DisputeUntil,
		"dispute_reason": p.DisputeReason, "resolution": p.Resolution, "lines": lines,
	}
}

// decodeStrict reads a small JSON body into v, refusing unknown fields.
func decodeStrict(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "invalid JSON body")
		return false
	}
	return true
}

// setAgreement serves PUT /v1/supply/partners/{tenant_id}/agreement (operations).
func (s *Server) setAgreement(w http.ResponseWriter, r *http.Request) {
	if !s.operator(w, r) {
		return
	}
	var a supply.Agreement
	if !decodeStrict(w, r, &a) {
		return
	}
	a.PartnerTenantID = r.PathValue("tenant_id")
	out, err := s.supply.Store.SetAgreement(r.Context(), a)
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// getAgreement serves GET …/agreement: the partner reads its own terms, operations any.
func (s *Server) getAgreement(w http.ResponseWriter, r *http.Request) {
	partner, _, ok := s.supplyCaller(w, r)
	if !ok {
		return
	}
	id := r.PathValue("tenant_id")
	if partner != "" && partner != id {
		writeErr(w, http.StatusNotFound, "not_found", "no payout agreement")
		return
	}
	a, err := s.supply.Store.GetAgreement(r.Context(), id)
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// closeCycle serves POST /v1/supply/payouts/cycles (operations).
func (s *Server) closeCycle(w http.ResponseWriter, r *http.Request) {
	if !s.operator(w, r) {
		return
	}
	var b struct {
		PeriodStart time.Time `json:"period_start"`
		PeriodEnd   time.Time `json:"period_end"`
	}
	if !decodeStrict(w, r, &b) {
		return
	}
	ps, err := s.supply.Store.CloseCycle(r.Context(), b.PeriodStart, b.PeriodEnd, s.clock(), "service")
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	data := make([]map[string]any, 0, len(ps))
	for _, p := range ps {
		data = append(data, payoutJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// listPayouts serves GET /v1/supply/payouts (own, or all for operations; ?state= filters).
func (s *Server) listPayouts(w http.ResponseWriter, r *http.Request) {
	partner, _, ok := s.supplyCaller(w, r)
	if !ok {
		return
	}
	state := r.URL.Query().Get("state")
	switch state {
	case "", supply.PayoutPending, supply.PayoutWired, supply.PayoutDisputed, supply.PayoutSettled:
	default:
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "unknown state")
		return
	}
	ps, err := s.supply.Store.ListPayouts(r.Context(), partner, state)
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	data := make([]map[string]any, 0, len(ps))
	for _, p := range ps {
		data = append(data, payoutJSON(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// getPayout serves GET /v1/supply/payouts/{payout_id}.
func (s *Server) getPayout(w http.ResponseWriter, r *http.Request) {
	partner, _, ok := s.supplyCaller(w, r)
	if !ok {
		return
	}
	p, err := s.supply.Store.GetPayout(r.Context(), r.PathValue("payout_id"), partner)
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payoutJSON(p))
}

// wirePayout serves POST …/wire (operations records the transfer).
func (s *Server) wirePayout(w http.ResponseWriter, r *http.Request) {
	if !s.operator(w, r) {
		return
	}
	var b struct {
		Reference string `json:"reference"`
	}
	if !decodeStrict(w, r, &b) {
		return
	}
	if b.Reference = strings.TrimSpace(b.Reference); b.Reference == "" || len(b.Reference) > 128 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "reference (1-128 chars) is required")
		return
	}
	p, err := s.supply.Store.Wire(r.Context(), r.PathValue("payout_id"), b.Reference, "service")
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payoutJSON(p))
}

// disputePayout serves POST …/dispute (the partner, inside the window).
func (s *Server) disputePayout(w http.ResponseWriter, r *http.Request) {
	partner, _, ok := s.supplyCaller(w, r)
	if !ok {
		return
	}
	if partner == "" {
		writeErr(w, http.StatusForbidden, "forbidden", "disputes are raised by the partner")
		return
	}
	var b struct {
		Reason string `json:"reason"`
	}
	if !decodeStrict(w, r, &b) {
		return
	}
	if b.Reason = strings.TrimSpace(b.Reason); b.Reason == "" || len(b.Reason) > 2000 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "reason (1-2000 chars) is required")
		return
	}
	p, err := s.supply.Store.Dispute(r.Context(), r.PathValue("payout_id"), partner, b.Reason, s.clock())
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payoutJSON(p))
}

// releaseHoldback serves POST …/release-holdback (operations, after the window).
func (s *Server) releaseHoldback(w http.ResponseWriter, r *http.Request) {
	if !s.operator(w, r) {
		return
	}
	p, err := s.supply.Store.ReleaseHoldback(r.Context(), r.PathValue("payout_id"), "service", s.clock())
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payoutJSON(p))
}

// resolvePayout serves POST …/resolve (operations close a dispute).
func (s *Server) resolvePayout(w http.ResponseWriter, r *http.Request) {
	if !s.operator(w, r) {
		return
	}
	var b struct {
		Outcome string `json:"outcome"`
		Note    string `json:"note"`
	}
	if !decodeStrict(w, r, &b) {
		return
	}
	if b.Note = strings.TrimSpace(b.Note); b.Note == "" || len(b.Note) > 2000 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "note (1-2000 chars) is required")
		return
	}
	p, err := s.supply.Store.Resolve(r.Context(), r.PathValue("payout_id"), b.Outcome, b.Note, "service")
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, payoutJSON(p))
}

// newRat parses a fixed-point decimal string.
func newRat(s string) (*big.Rat, bool) { return new(big.Rat).SetString(s) }

// hours renders GPU-seconds as GPU-hours, truncated to 6 places.
func hours(secs *big.Rat) string {
	if secs == nil {
		return "0.000000"
	}
	h := new(big.Rat).Quo(secs, big.NewRat(3600, 1))
	scaled := new(big.Int).Quo(new(big.Int).Mul(h.Num(), big.NewInt(1_000_000)), h.Denom())
	str := fmt.Sprintf("%07d", scaled)
	return str[:len(str)-6] + "." + str[len(str)-6:]
}
