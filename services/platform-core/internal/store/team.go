package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/trade1/platform-core/internal/domain"
)

// Team errors surfaced to the API.
var (
	ErrLastAdmin      = errors.New("store: a tenant must keep at least one admin")
	ErrInvitePending  = errors.New("store: an invitation to that email is already pending")
	ErrInviteInvalid  = errors.New("store: invitation is invalid, used, revoked or expired")
	ErrNoSubAccount   = errors.New("store: no such sub-account")
	ErrSubAccountName = errors.New("store: a sub-account with that name exists")
	ErrNotMember      = errors.New("store: no such member")
)

// querier is a pool or a transaction (what the read helpers need).
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Member is one user of a tenant, as the team screen shows it.
type Member struct {
	ID            string
	Email         string
	Roles         []domain.Role
	SubAccountID  *string
	EmailVerified bool
	CreatedAt     time.Time
}

// ListMembers returns a tenant's users, oldest first.
func (s *Store) ListMembers(ctx context.Context, tenantID string) ([]Member, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text, email, roles, sub_account_id::text, email_verified, created_at
		FROM users WHERE tenant_id=$1 ORDER BY created_at, id`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()
	out := []Member{}
	for rows.Next() {
		var m Member
		var roles []string
		if err := rows.Scan(&m.ID, &m.Email, &roles, &m.SubAccountID, &m.EmailVerified, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Roles = toRoles(roles)
		out = append(out, m)
	}
	return out, rows.Err()
}

// RemoveMember deletes a user from a tenant. The last admin cannot be removed (the tenant would be
// unmanageable). Returns the removed member for the audit.
func (s *Store) RemoveMember(ctx context.Context, tenantID, userID string) (Member, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return Member{}, ErrNotMember
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Member{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	// Lock every user of the tenant so two concurrent removals cannot each leave "another" admin.
	rows, err := tx.Query(ctx, `SELECT id::text, email, roles FROM users WHERE tenant_id=$1 FOR UPDATE`, tenantID)
	if err != nil {
		return Member{}, err
	}
	var target *Member
	admins := 0
	for rows.Next() {
		var m Member
		var roles []string
		if err := rows.Scan(&m.ID, &m.Email, &roles); err != nil {
			rows.Close()
			return Member{}, err
		}
		m.Roles = toRoles(roles)
		if hasExact(m.Roles, domain.RoleAdmin) {
			admins++
		}
		if m.ID == userID {
			cp := m
			target = &cp
		}
	}
	rows.Close()
	if target == nil {
		return Member{}, ErrNotMember
	}
	if hasExact(target.Roles, domain.RoleAdmin) && admins <= 1 {
		return Member{}, ErrLastAdmin
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id=$1 AND tenant_id=$2`, userID, tenantID); err != nil {
		return Member{}, fmt.Errorf("remove member: %w", err)
	}
	return *target, tx.Commit(ctx)
}

// hasExact reports whether roles contains role itself (not via admin).
func hasExact(roles []domain.Role, role domain.Role) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// Invite is a pending (or finished) invitation.
type Invite struct {
	ID           string
	TenantID     string
	TenantName   string
	Email        string
	Roles        []domain.Role
	SubAccountID *string
	InvitedBy    string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// CreateInvite records an invitation. An email that already has an account (anywhere: one account
// per email) is ErrEmailTaken; a second pending invite to the same email is ErrInvitePending.
func (s *Store) CreateInvite(ctx context.Context, tenantID, email string, roles []string, subAccountID *string, tokenHash, invitedBy string, ttl time.Duration) (Invite, error) {
	email = domain.NormalizeEmail(email)
	if subAccountID != nil {
		if err := s.checkSubAccount(ctx, s.pool, tenantID, *subAccountID); err != nil {
			return Invite{}, err
		}
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email=$1)`, email).Scan(&exists); err != nil {
		return Invite{}, err
	}
	if exists {
		return Invite{}, ErrEmailTaken
	}
	inv := Invite{ID: uuid.NewString(), TenantID: tenantID, Email: email, Roles: toRoles(roles), SubAccountID: subAccountID, InvitedBy: invitedBy}
	err := s.pool.QueryRow(ctx, `INSERT INTO invites (id, tenant_id, email, roles, sub_account_id, token_hash, invited_by, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7, now() + $8::interval) RETURNING created_at, expires_at`,
		inv.ID, tenantID, email, roles, subAccountID, tokenHash, invitedBy, fmt.Sprintf("%d seconds", int(ttl.Seconds()))).
		Scan(&inv.CreatedAt, &inv.ExpiresAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Invite{}, ErrInvitePending
	}
	if err != nil {
		return Invite{}, fmt.Errorf("create invite: %w", err)
	}
	return inv, nil
}

// ListInvites returns a tenant's pending, unexpired invitations, newest first.
func (s *Store) ListInvites(ctx context.Context, tenantID string) ([]Invite, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text, email, roles, sub_account_id::text, invited_by::text, created_at, expires_at
		FROM invites WHERE tenant_id=$1 AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()
	out := []Invite{}
	for rows.Next() {
		inv := Invite{TenantID: tenantID}
		var roles []string
		if err := rows.Scan(&inv.ID, &inv.Email, &roles, &inv.SubAccountID, &inv.InvitedBy, &inv.CreatedAt, &inv.ExpiresAt); err != nil {
			return nil, err
		}
		inv.Roles = toRoles(roles)
		out = append(out, inv)
	}
	return out, rows.Err()
}

// RevokeInvite cancels a pending invitation of the tenant. ok=false when there is none.
func (s *Store) RevokeInvite(ctx context.Context, tenantID, id string) (Invite, bool, error) {
	if !isUUID(id) {
		return Invite{}, false, nil // not an id we could have issued
	}
	var inv Invite
	var roles []string
	err := s.pool.QueryRow(ctx, `UPDATE invites SET revoked_at=now() WHERE id=$1 AND tenant_id=$2 AND accepted_at IS NULL
		AND revoked_at IS NULL RETURNING id::text, email, roles`, id, tenantID).Scan(&inv.ID, &inv.Email, &roles)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invite{}, false, nil
	}
	inv.Roles = toRoles(roles)
	return inv, err == nil, err
}

// LookupInvite returns a pending, unexpired invitation by its token hash (for the accept screen).
func (s *Store) LookupInvite(ctx context.Context, tokenHash string) (Invite, error) {
	return lookupInvite(ctx, s.pool, tokenHash, false)
}

// lookupInvite reads a pending invitation, optionally locking it.
func lookupInvite(ctx context.Context, q querier, tokenHash string, lock bool) (Invite, error) {
	sql := `SELECT i.id::text, i.tenant_id::text, t.name, i.email, i.roles, i.sub_account_id::text, i.invited_by::text, i.created_at, i.expires_at
		FROM invites i JOIN tenants t ON t.id = i.tenant_id
		WHERE i.token_hash=$1 AND i.accepted_at IS NULL AND i.revoked_at IS NULL AND i.expires_at > now()`
	if lock {
		sql += ` FOR UPDATE OF i`
	}
	var inv Invite
	var roles []string
	err := q.QueryRow(ctx, sql, tokenHash).Scan(&inv.ID, &inv.TenantID, &inv.TenantName, &inv.Email, &roles, &inv.SubAccountID,
		&inv.InvitedBy, &inv.CreatedAt, &inv.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invite{}, ErrInviteInvalid
	}
	inv.Roles = toRoles(roles)
	return inv, err
}

// AcceptInvite turns a pending invitation into a user of the inviting tenant, with the invited roles
// and sub-account, its email verified (the token proves the mailbox). Single use: the invitation is
// locked and marked accepted in the same transaction.
func (s *Store) AcceptInvite(ctx context.Context, tokenHash, passwordHash string) (AuthUser, Invite, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AuthUser{}, Invite{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	inv, err := lookupInvite(ctx, tx, tokenHash, true)
	if err != nil {
		return AuthUser{}, Invite{}, err
	}
	u := AuthUser{UserID: uuid.NewString(), TenantID: inv.TenantID, Roles: inv.Roles, EmailVerified: true}
	if inv.SubAccountID != nil {
		u.SubAccountID = *inv.SubAccountID
	}
	roles := make([]string, len(inv.Roles))
	for i, r := range inv.Roles {
		roles[i] = string(r)
	}
	_, err = tx.Exec(ctx, `INSERT INTO users (id, tenant_id, email, password_hash, roles, email_verified, sub_account_id)
		VALUES ($1,$2,$3,$4,$5,true,$6)`, u.UserID, inv.TenantID, inv.Email, passwordHash, roles, inv.SubAccountID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return AuthUser{}, Invite{}, ErrEmailTaken
	}
	if err != nil {
		return AuthUser{}, Invite{}, fmt.Errorf("accept invite: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE invites SET accepted_at=now() WHERE id=$1`, inv.ID); err != nil {
		return AuthUser{}, Invite{}, err
	}
	if err := tx.QueryRow(ctx, `SELECT is_paper FROM tenants WHERE id=$1`, inv.TenantID).Scan(&u.IsPaper); err != nil {
		return AuthUser{}, Invite{}, err
	}
	return u, inv, tx.Commit(ctx)
}

// SubAccount is a team inside a tenant.
type SubAccount struct {
	ID        string
	Name      string
	Members   int
	CreatedAt time.Time
}

// CreateSubAccount adds a sub-account to a tenant.
func (s *Store) CreateSubAccount(ctx context.Context, tenantID, name string) (SubAccount, error) {
	sa := SubAccount{ID: uuid.NewString(), Name: name}
	err := s.pool.QueryRow(ctx, `INSERT INTO sub_accounts (id, tenant_id, name) VALUES ($1,$2,$3) RETURNING created_at`,
		sa.ID, tenantID, name).Scan(&sa.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return SubAccount{}, ErrSubAccountName
	}
	if err != nil {
		return SubAccount{}, fmt.Errorf("create sub-account: %w", err)
	}
	return sa, nil
}

// ListSubAccounts returns a tenant's sub-accounts with their member counts, by name.
func (s *Store) ListSubAccounts(ctx context.Context, tenantID string) ([]SubAccount, error) {
	rows, err := s.pool.Query(ctx, `SELECT s.id::text, s.name, s.created_at, count(u.id)::int FROM sub_accounts s
		LEFT JOIN users u ON u.sub_account_id = s.id WHERE s.tenant_id=$1 GROUP BY s.id ORDER BY s.name`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list sub-accounts: %w", err)
	}
	defer rows.Close()
	out := []SubAccount{}
	for rows.Next() {
		var sa SubAccount
		if err := rows.Scan(&sa.ID, &sa.Name, &sa.CreatedAt, &sa.Members); err != nil {
			return nil, err
		}
		out = append(out, sa)
	}
	return out, rows.Err()
}

// GetSubAccount returns one of a tenant's sub-accounts.
func (s *Store) GetSubAccount(ctx context.Context, tenantID, id string) (SubAccount, error) {
	if _, err := uuid.Parse(id); err != nil {
		return SubAccount{}, ErrNoSubAccount
	}
	var sa SubAccount
	err := s.pool.QueryRow(ctx, `SELECT id::text, name, created_at FROM sub_accounts WHERE id=$1 AND tenant_id=$2`, id, tenantID).
		Scan(&sa.ID, &sa.Name, &sa.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SubAccount{}, ErrNoSubAccount
	}
	return sa, err
}

// checkSubAccount verifies a sub-account belongs to the tenant.
func (s *Store) checkSubAccount(ctx context.Context, q querier, tenantID, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrNoSubAccount
	}
	var ok bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM sub_accounts WHERE id=$1 AND tenant_id=$2)`, id, tenantID).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return ErrNoSubAccount
	}
	return nil
}

// SetMemberSubAccount moves a member into a sub-account (nil = the main balance). Returns the previous
// one for the audit.
func (s *Store) SetMemberSubAccount(ctx context.Context, tenantID, userID string, subAccountID *string) (prev *string, err error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, ErrNotMember
	}
	if subAccountID != nil {
		if err := s.checkSubAccount(ctx, s.pool, tenantID, *subAccountID); err != nil {
			return nil, err
		}
	}
	err = s.pool.QueryRow(ctx, `UPDATE users u SET sub_account_id=$3 FROM (SELECT sub_account_id AS old FROM users
		WHERE id=$1 AND tenant_id=$2 FOR UPDATE) o WHERE u.id=$1 AND u.tenant_id=$2 RETURNING o.old::text`,
		userID, tenantID, subAccountID).Scan(&prev)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotMember
	}
	return prev, err
}

// isUUID reports whether id is a UUID.
func isUUID(id string) bool { return uuid.Validate(id) == nil }
