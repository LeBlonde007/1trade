package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/exascale/platform-core/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Org is an organization under a tenant (multi-user accounts).
type Org struct {
	ID        string
	TenantID  string
	Name      string
	CreatedAt time.Time
}

// CreateOrg creates an org under a tenant.
func (s *Store) CreateOrg(ctx context.Context, tenantID, name string) (Org, error) {
	var o Org
	err := s.pool.QueryRow(ctx,
		`INSERT INTO orgs (id, tenant_id, name) VALUES ($1,$2,$3) RETURNING id, tenant_id, name, created_at`,
		uuid.NewString(), tenantID, name).Scan(&o.ID, &o.TenantID, &o.Name, &o.CreatedAt)
	if err != nil {
		return Org{}, fmt.Errorf("create org: %w", err)
	}
	return o, nil
}

// ListOrgs returns a tenant's orgs, newest first.
func (s *Store) ListOrgs(ctx context.Context, tenantID string) ([]Org, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, name, created_at FROM orgs WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Org
	for rows.Next() {
		var o Org
		if err := rows.Scan(&o.ID, &o.TenantID, &o.Name, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// AssignRoles sets a user's roles, scoped to the tenant (one tenant can't touch another's users).
// Returns the prior roles (for the audit diff) and ok=false if the user isn't in this tenant.
func (s *Store) AssignRoles(ctx context.Context, tenantID, userID string, roles []string) (prev []string, ok bool, err error) {
	err = s.pool.QueryRow(ctx, `SELECT roles FROM users WHERE id=$1 AND tenant_id=$2`, userID, tenantID).Scan(&prev)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if _, err := s.pool.Exec(ctx, `UPDATE users SET roles=$1 WHERE id=$2 AND tenant_id=$3`, roles, userID, tenantID); err != nil {
		return nil, false, err
	}
	return prev, true, nil
}

// TenantRow is a tenant's details.
type TenantRow struct {
	ID        string
	Name      string
	Kind      string
	IsPaper   bool
	CreatedAt time.Time
}

// GetTenant loads a tenant by id. ok=false if not found.
func (s *Store) GetTenant(ctx context.Context, id string) (TenantRow, bool, error) {
	var t TenantRow
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, kind, is_paper, created_at FROM tenants WHERE id=$1`, id).
		Scan(&t.ID, &t.Name, &t.Kind, &t.IsPaper, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return TenantRow{}, false, nil
	}
	if err != nil {
		return TenantRow{}, false, err
	}
	return t, true, nil
}

// OrgUser is a user in an org (with roles).
type OrgUser struct {
	ID    string
	Email string
	Roles []domain.Role
}

// ListOrgUsers returns the users in an org, scoped to a tenant.
func (s *Store) ListOrgUsers(ctx context.Context, tenantID, orgID string) ([]OrgUser, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, email, roles FROM users WHERE org_id=$1 AND tenant_id=$2 ORDER BY created_at`, orgID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OrgUser
	for rows.Next() {
		var u OrgUser
		var roles []string
		if err := rows.Scan(&u.ID, &u.Email, &roles); err != nil {
			return nil, err
		}
		u.Roles = toRoles(roles)
		out = append(out, u)
	}
	return out, rows.Err()
}
