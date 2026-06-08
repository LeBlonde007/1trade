package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Conversation is a stored chat transcript for the inference playground. Transcript is the raw JSON of
// the turns (with media referenced by Spaces URL), kept opaque to the platform.
type Conversation struct {
	ID         string          `json:"id"`
	TenantID   string          `json:"-"`
	UserID     string          `json:"-"`
	Title      string          `json:"title"`
	Model      string          `json:"model"`
	Transcript json.RawMessage `json:"transcript"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// ConversationMeta is the list view (no transcript body) so the sidebar query stays light.
type ConversationMeta struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// emptyTranscript normalizes a missing transcript to a JSON empty array.
func emptyTranscript(t json.RawMessage) json.RawMessage {
	if len(t) == 0 {
		return json.RawMessage("[]")
	}
	return t
}

// CreateConversation inserts a new conversation for a tenant and returns the stored row.
func (s *Store) CreateConversation(ctx context.Context, tenantID, userID, title, model string, transcript json.RawMessage) (Conversation, error) {
	if title == "" {
		title = "New chat"
	}
	var c Conversation
	var uid *string
	if userID != "" {
		uid = &userID
	}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO conversations (id, tenant_id, user_id, title, model, transcript)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 RETURNING id, title, model, transcript, created_at, updated_at`,
		uuid.NewString(), tenantID, uid, title, model, emptyTranscript(transcript)).
		Scan(&c.ID, &c.Title, &c.Model, &c.Transcript, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return Conversation{}, err
	}
	c.TenantID, c.UserID = tenantID, userID
	return c, nil
}

// ListConversations returns a tenant's conversations (metadata only), newest activity first.
func (s *Store) ListConversations(ctx context.Context, tenantID string, limit int) ([]ConversationMeta, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, title, model, created_at, updated_at
		 FROM conversations WHERE tenant_id=$1 ORDER BY updated_at DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ConversationMeta{}
	for rows.Next() {
		var m ConversationMeta
		if err := rows.Scan(&m.ID, &m.Title, &m.Model, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetConversation loads a single conversation (full transcript), scoped to the tenant so one tenant
// can't read another's. ok=false if not found / not owned.
func (s *Store) GetConversation(ctx context.Context, tenantID, id string) (Conversation, bool, error) {
	var c Conversation
	err := s.pool.QueryRow(ctx,
		`SELECT id, title, model, transcript, created_at, updated_at
		 FROM conversations WHERE id=$1 AND tenant_id=$2`, id, tenantID).
		Scan(&c.ID, &c.Title, &c.Model, &c.Transcript, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, false, nil
	}
	if err != nil {
		return Conversation{}, false, err
	}
	c.TenantID = tenantID
	return c, true, nil
}

// UpdateConversation replaces a conversation's transcript (and optionally title/model), scoped to the
// tenant. ok=false if the id isn't the tenant's (so it can't overwrite another tenant's row).
func (s *Store) UpdateConversation(ctx context.Context, tenantID, id, title, model string, transcript json.RawMessage) (Conversation, bool, error) {
	var c Conversation
	err := s.pool.QueryRow(ctx,
		`UPDATE conversations
		 SET title=COALESCE(NULLIF($3,''), title),
		     model=COALESCE(NULLIF($4,''), model),
		     transcript=$5, updated_at=now()
		 WHERE id=$1 AND tenant_id=$2
		 RETURNING id, title, model, transcript, created_at, updated_at`,
		id, tenantID, title, model, emptyTranscript(transcript)).
		Scan(&c.ID, &c.Title, &c.Model, &c.Transcript, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Conversation{}, false, nil
	}
	if err != nil {
		return Conversation{}, false, err
	}
	c.TenantID = tenantID
	return c, true, nil
}

// RenameConversation updates only the title (tenant-scoped). ok=false if not owned.
func (s *Store) RenameConversation(ctx context.Context, tenantID, id, title string) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE conversations SET title=$3, updated_at=now() WHERE id=$1 AND tenant_id=$2`, id, tenantID, title)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// DeleteConversation removes a tenant's conversation. ok=false if not owned (so a guessed id can't
// delete another tenant's row).
func (s *Store) DeleteConversation(ctx context.Context, tenantID, id string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM conversations WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
