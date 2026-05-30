package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AuditEntry is one sensitive action to record. Before/After are JSON-serializable snapshots used
// for diffs (nil → SQL NULL). ActorID "" means a system / self-serve action.
type AuditEntry struct {
	TenantID   string
	ActorID    string
	Action     string
	TargetType string
	TargetID   string
	Before     any
	After      any
	IsPaper    bool
}

// WriteAudit appends an audit_log row and returns its id. Every sensitive action calls this.
func (s *Store) WriteAudit(ctx context.Context, e AuditEntry) (string, error) {
	id := uuid.NewString()
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit_log (id, tenant_id, actor_id, action, target_type, target_id, before, after, is_paper)
		 VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9)`,
		id, e.TenantID, strOrNil(e.ActorID), e.Action, strOrNil(e.TargetType), strOrNil(e.TargetID),
		jsonOrNil(e.Before), jsonOrNil(e.After), e.IsPaper)
	if err != nil {
		return "", fmt.Errorf("write audit: %w", err)
	}
	return id, nil
}

// AuditRow is an audit_log entry for listing.
type AuditRow struct {
	ID         string
	TenantID   string
	ActorID    *string
	Action     string
	TargetType string
	TargetID   string
	Before     json.RawMessage
	After      json.RawMessage
	IsPaper    bool
	CreatedAt  time.Time
}

// ListAudit returns a tenant's audit log, newest first.
func (s *Store) ListAudit(ctx context.Context, tenantID string, limit int) ([]AuditRow, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, actor_id, action, COALESCE(target_type,''), COALESCE(target_id,''),
		        COALESCE(before,'null'::jsonb), COALESCE(after,'null'::jsonb), is_paper, created_at
		 FROM audit_log WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditRow
	for rows.Next() {
		var r AuditRow
		if err := rows.Scan(&r.ID, &r.TenantID, &r.ActorID, &r.Action, &r.TargetType, &r.TargetID,
			&r.Before, &r.After, &r.IsPaper, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// strOrNil returns nil for "" (→ SQL NULL) else a pointer to s.
func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// jsonOrNil marshals v to JSON text, or returns nil (→ SQL NULL) when v is nil/unmarshalable.
func jsonOrNil(v any) *string {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	s := string(b)
	return &s
}
