package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// conversationBody is the create/update payload from the playground (the transcript is opaque JSON).
type conversationBody struct {
	Title      string          `json:"title"`
	Model      string          `json:"model"`
	Transcript json.RawMessage `json:"transcript"`
}

// listConversations returns the caller's conversations (metadata only) for the history sidebar.
func (s *Server) listConversations(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	out, err := s.st.ListConversations(r.Context(), p.TenantID, 200)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": out})
}

// createConversation persists a new conversation and returns it (with its server-assigned id).
func (s *Server) createConversation(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	var b conversationBody
	_ = json.NewDecoder(r.Body).Decode(&b) // all fields optional — a new chat may be empty
	c, err := s.st.CreateConversation(r.Context(), p.TenantID, p.UserID, b.Title, b.Model, b.Transcript)
	if err != nil {
		serverError(w, err)
		return
	}
	slog.Info("audit: conversation created", "tenant_id", p.TenantID, "actor_id", p.UserID, "conversation_id", c.ID)
	writeJSON(w, http.StatusCreated, c)
}

// getConversation loads one full conversation (transcript included), scoped to the caller's tenant.
func (s *Server) getConversation(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	c, found, err := s.st.GetConversation(r.Context(), p.TenantID, r.PathValue("id"))
	if err != nil {
		serverError(w, err)
		return
	}
	if !found {
		writeErr(w, http.StatusNotFound, "not_found", "conversation not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// updateConversation replaces a conversation's transcript (and optionally title/model), tenant-scoped.
func (s *Server) updateConversation(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	var b conversationBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	c, found, err := s.st.UpdateConversation(r.Context(), p.TenantID, r.PathValue("id"), b.Title, b.Model, b.Transcript)
	if err != nil {
		serverError(w, err)
		return
	}
	if !found {
		writeErr(w, http.StatusNotFound, "not_found", "conversation not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// deleteConversation removes a conversation (tenant-scoped; a guessed id can't touch another tenant's).
func (s *Server) deleteConversation(w http.ResponseWriter, r *http.Request) {
	p, ok := s.authed(w, r)
	if !ok {
		return
	}
	found, err := s.st.DeleteConversation(r.Context(), p.TenantID, r.PathValue("id"))
	if err != nil {
		serverError(w, err)
		return
	}
	if !found {
		writeErr(w, http.StatusNotFound, "not_found", "conversation not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
