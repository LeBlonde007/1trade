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

// SSO errors.
var (
	ErrNoSSO          = errors.New("store: single sign-on is not configured")
	ErrDomainTaken    = errors.New("store: an email domain already routes to another tenant")
	ErrSSORequest     = errors.New("store: unknown, used or expired sign-on request")
	ErrSSOOtherTenant = errors.New("store: that email belongs to another tenant")
	ErrSSONoAccount   = errors.New("store: no account and just-in-time creation is off")
)

// ssoRequestTTL is how long a sign-on request may take at the IdP.
const ssoRequestTTL = 10 * time.Minute

// SSOConfig is a tenant's IdP configuration.
type SSOConfig struct {
	TenantID    string
	IDPMetadata string
	IDPEntityID string
	DefaultRole string
	JIT         bool
	Enforce     bool
	Domains     []string
	UpdatedAt   time.Time
}

// GetSSO returns a tenant's configuration, or ErrNoSSO.
func (s *Store) GetSSO(ctx context.Context, tenantID string) (SSOConfig, error) {
	c := SSOConfig{TenantID: tenantID}
	err := s.pool.QueryRow(ctx, `SELECT idp_metadata, idp_entity_id, default_role, jit, enforce, updated_at,
		coalesce((SELECT array_agg(domain ORDER BY domain) FROM sso_domains WHERE tenant_id=$1), '{}')
		FROM sso_configs WHERE tenant_id=$1`, tenantID).
		Scan(&c.IDPMetadata, &c.IDPEntityID, &c.DefaultRole, &c.JIT, &c.Enforce, &c.UpdatedAt, &c.Domains)
	if errors.Is(err, pgx.ErrNoRows) {
		return SSOConfig{}, ErrNoSSO
	}
	return c, err
}

// PutSSO replaces a tenant's configuration and the email domains that route to it. A domain that
// already routes to another tenant is ErrDomainTaken (nothing changes).
func (s *Store) PutSSO(ctx context.Context, c SSOConfig) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	if _, err := tx.Exec(ctx, `INSERT INTO sso_configs (tenant_id, idp_metadata, idp_entity_id, default_role, jit, enforce)
		VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (tenant_id) DO UPDATE SET idp_metadata=excluded.idp_metadata,
		idp_entity_id=excluded.idp_entity_id, default_role=excluded.default_role, jit=excluded.jit, enforce=excluded.enforce,
		updated_at=now()`, c.TenantID, c.IDPMetadata, c.IDPEntityID, c.DefaultRole, c.JIT, c.Enforce); err != nil {
		return fmt.Errorf("put sso: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sso_domains WHERE tenant_id=$1`, c.TenantID); err != nil {
		return err
	}
	for _, d := range c.Domains {
		_, err := tx.Exec(ctx, `INSERT INTO sso_domains (domain, tenant_id) VALUES ($1,$2)`, d, c.TenantID)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDomainTaken
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// DeleteSSO removes a tenant's configuration and domains.
func (s *Store) DeleteSSO(ctx context.Context, tenantID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	if _, err := tx.Exec(ctx, `DELETE FROM sso_domains WHERE tenant_id=$1`, tenantID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sso_configs WHERE tenant_id=$1`, tenantID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SSOForDomain returns the configuration an email domain routes to, or ErrNoSSO.
func (s *Store) SSOForDomain(ctx context.Context, emailDomain string) (SSOConfig, error) {
	var tenant string
	err := s.pool.QueryRow(ctx, `SELECT tenant_id::text FROM sso_domains WHERE domain=$1`, emailDomain).Scan(&tenant)
	if errors.Is(err, pgx.ErrNoRows) {
		return SSOConfig{}, ErrNoSSO
	}
	if err != nil {
		return SSOConfig{}, err
	}
	return s.GetSSO(ctx, tenant)
}

// SaveSSORequest records an AuthnRequest id we sent for a tenant.
func (s *Store) SaveSSORequest(ctx context.Context, id, tenantID string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO sso_requests (id, tenant_id) VALUES ($1,$2)`, id, tenantID)
	return err
}

// ClaimSSORequest consumes a recent, unused request id and returns its tenant. Single use.
func (s *Store) ClaimSSORequest(ctx context.Context, id string) (string, error) {
	var tenant string
	err := s.pool.QueryRow(ctx, `UPDATE sso_requests SET used_at=now() WHERE id=$1 AND used_at IS NULL
		AND created_at > now() - $2::interval RETURNING tenant_id::text`, id, fmt.Sprintf("%d seconds", int(ssoRequestTTL.Seconds()))).Scan(&tenant)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrSSORequest
	}
	return tenant, err
}

// SSOUser finds the member an asserted email names in the tenant, creating one (email verified, the
// default role, no usable password) when just-in-time creation is on. An email that belongs to
// another tenant is refused: SSO never moves or merges accounts. created reports a new member.
func (s *Store) SSOUser(ctx context.Context, c SSOConfig, email string) (u AuthUser, created bool, err error) {
	email = domain.NormalizeEmail(email)
	au, found, err := s.GetUserByEmail(ctx, email)
	if err != nil {
		return AuthUser{}, false, err
	}
	if found {
		if au.TenantID != c.TenantID {
			return AuthUser{}, false, ErrSSOOtherTenant
		}
		return au, false, nil
	}
	if !c.JIT {
		return AuthUser{}, false, ErrSSONoAccount
	}
	id := uuid.NewString()
	// "!" is not a bcrypt hash, so password sign-in can never succeed for an SSO-created member.
	_, err = s.pool.Exec(ctx, `INSERT INTO users (id, tenant_id, email, password_hash, roles, email_verified)
		VALUES ($1,$2,$3,'!',$4,true)`, id, c.TenantID, email, []string{c.DefaultRole})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return s.SSOUser(ctx, c, email) // created concurrently: look it up again
	}
	if err != nil {
		return AuthUser{}, false, fmt.Errorf("sso user: %w", err)
	}
	au, _, err = s.GetUserByEmail(ctx, email)
	return au, true, err
}

// SSOEnforced reports whether a tenant requires single sign-on.
func (s *Store) SSOEnforced(ctx context.Context, tenantID string) (bool, error) {
	var on bool
	err := s.pool.QueryRow(ctx, `SELECT coalesce((SELECT enforce FROM sso_configs WHERE tenant_id=$1), false)`, tenantID).Scan(&on)
	return on, err
}
