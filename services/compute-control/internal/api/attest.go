package api

import (
	"encoding/base64"
	"net/http"

	"github.com/trade1/compute-control/internal/supply"
)

// attestRoutes registers the attestation endpoints (supply.yaml v1.2, F19).
func (s *Server) attestRoutes() {
	s.mux.HandleFunc("GET /v1/supply/sources/{id}/attestation", s.getAttestation)
	s.mux.HandleFunc("POST /v1/supply/sources/{id}/attestation/{layer}", s.recordManualLayer)
	s.mux.HandleFunc("POST /v1/supply/sources/{id}/challenges", s.issueChallenge)
	s.mux.HandleFunc("POST /v1/supply/sources/{id}/challenges/{challenge_id}", s.answerChallenge)
}

// getAttestation serves GET …/attestation: each layer's latest result (the partner's own, or any for
// operations).
func (s *Server) getAttestation(w http.ResponseWriter, r *http.Request) {
	partner, _, ok := s.supplyCaller(w, r)
	if !ok {
		return
	}
	st, err := s.supply.Store.Attestation(r.Context(), r.PathValue("id"), partner)
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// recordManualLayer serves POST …/attestation/{kyb|bond}: operations record a KYB review or a bond,
// with the evidence behind it. A pass that completes attestation activates the source.
func (s *Server) recordManualLayer(w http.ResponseWriter, r *http.Request) {
	if !s.operator(w, r) {
		return
	}
	layer := r.PathValue("layer")
	if layer != "kyb" && layer != "bond" {
		writeErr(w, http.StatusNotFound, "not_found", "only kyb and bond are recorded by operations")
		return
	}
	var b struct {
		State    string         `json:"state"`
		Evidence map[string]any `json:"evidence"`
	}
	if !decodeStrict(w, r, &b) {
		return
	}
	if (b.State != supply.Pass && b.State != supply.Fail) || len(b.Evidence) == 0 {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "state (pass|fail) and non-empty evidence are required")
		return
	}
	st, err := s.supply.Store.RecordLayer(r.Context(), r.PathValue("id"), layer, b.State, b.Evidence, "service")
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	s.resync(r.Context())
	writeJSON(w, http.StatusOK, st)
}

// issueChallenge serves POST …/challenges: a fresh nonce for the source's agent.
func (s *Server) issueChallenge(w http.ResponseWriter, r *http.Request) {
	partner, _, ok := s.supplyCaller(w, r)
	if !ok {
		return
	}
	var b struct {
		Kind string `json:"kind"`
	}
	if !decodeStrict(w, r, &b) {
		return
	}
	if b.Kind != supply.KindHardware && b.Kind != supply.KindChallenge {
		writeErr(w, http.StatusUnprocessableEntity, "bad_request", "kind must be hardware or challenge")
		return
	}
	c, err := s.supply.Store.IssueChallenge(r.Context(), r.PathValue("id"), partner, b.Kind, s.clock())
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// answerChallenge serves POST …/challenges/{challenge_id}: the single answer to a challenge. The
// layer it proves is recorded pass or fail; the source activates or suspends accordingly.
func (s *Server) answerChallenge(w http.ResponseWriter, r *http.Request) {
	partner, _, ok := s.supplyCaller(w, r)
	if !ok {
		return
	}
	var b struct {
		Report    string `json:"report"`
		Signature string `json:"signature"`
		Result    string `json:"result"`
	}
	if !decodeStrictN(w, r, &b, 1<<20) { // a report can list thousands of GPUs
		return
	}
	a := supply.Answer{Result: b.Result}
	if b.Report != "" || b.Signature != "" {
		var err1, err2 error
		a.Report, err1 = base64.StdEncoding.DecodeString(b.Report)
		a.Signature, err2 = base64.StdEncoding.DecodeString(b.Signature)
		if err1 != nil || err2 != nil {
			writeErr(w, http.StatusUnprocessableEntity, "bad_request", "report and signature must be base64")
			return
		}
	}
	st, err := s.supply.Store.AnswerChallenge(r.Context(), r.PathValue("id"), partner, r.PathValue("challenge_id"), a, s.supply.Verifier, s.clock())
	if err != nil {
		writeSupplyErr(w, err)
		return
	}
	s.resync(r.Context())
	writeJSON(w, http.StatusOK, st)
}
