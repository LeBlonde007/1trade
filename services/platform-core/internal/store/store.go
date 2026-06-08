// Package store is the Postgres binding for platform-core: tenants, orgs, users, and API keys.
// Domain logic (hashing, JWT) stays in internal/domain; this layer is IO only.
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/exascale/platform-core/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrEmailTaken is returned when a signup email already exists.
var ErrEmailTaken = errors.New("email already registered")

// Store wraps a pgx connection pool.
type Store struct{ pool *pgxpool.Pool }

// New connects to Postgres and verifies the connection.
func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Ping reports whether Postgres is reachable (used by the readiness probe).
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// User is the result of a signup (the new account's identity).
type User struct {
	ID       string
	TenantID string
	Email    string
	Roles    []domain.Role
	IsPaper  bool
}

// Signup atomically creates an individual tenant + its first (admin) user. Returns ErrEmailTaken if
// the email is already registered.
func (s *Store) Signup(ctx context.Context, email, passwordHash, tenantName string) (User, error) {
	email = domain.NormalizeEmail(email) // one account per address regardless of case
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tenantID := uuid.NewString()
	if _, err := tx.Exec(ctx,
		`INSERT INTO tenants (id, name, kind, is_paper) VALUES ($1,$2,'individual',TRUE)`,
		tenantID, tenantName); err != nil {
		return User{}, fmt.Errorf("insert tenant: %w", err)
	}

	userID := uuid.NewString()
	roles := []string{string(domain.RoleAdmin)}
	if _, err := tx.Exec(ctx,
		`INSERT INTO users (id, tenant_id, email, password_hash, roles) VALUES ($1,$2,$3,$4,$5)`,
		userID, tenantID, email, passwordHash, roles); err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrEmailTaken
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit: %w", err)
	}
	return User{ID: userID, TenantID: tenantID, Email: email, Roles: []domain.Role{domain.RoleAdmin}, IsPaper: true}, nil
}

// AuthUser is everything needed to authenticate a login and mint a JWT.
type AuthUser struct {
	UserID        string
	TenantID      string
	OrgID         string
	PasswordHash  string
	Roles         []domain.Role
	IsPaper       bool
	EmailVerified bool
}

// GetUserByEmail loads the login record (joined with the tenant for is_paper). ok=false if no such user.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (AuthUser, bool, error) {
	email = domain.NormalizeEmail(email) // match Signup's canonical form (case-insensitive login)
	var u AuthUser
	var org *string
	var roles []string
	err := s.pool.QueryRow(ctx,
		`SELECT u.id, u.tenant_id, u.org_id, u.password_hash, u.roles, t.is_paper, u.email_verified
		 FROM users u JOIN tenants t ON t.id = u.tenant_id WHERE u.email = $1`, email).
		Scan(&u.UserID, &u.TenantID, &org, &u.PasswordHash, &roles, &u.IsPaper, &u.EmailVerified)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthUser{}, false, nil
	}
	if err != nil {
		return AuthUser{}, false, err
	}
	if org != nil {
		u.OrgID = *org
	}
	u.Roles = toRoles(roles)
	return u, true, nil
}

// Identity is the current-user view returned by /v1/auth/me. KYCStatus lets the web app render the
// real-money gate without a second round trip (the gate reads it off the /me user object).
type Identity struct {
	UserID, Email, TenantID, OrgID string
	Roles                          []domain.Role
	IsPaper                        bool
	KYCStatus                      domain.KYCStatus
}

// GetUserByID loads a user's identity by id (for /me). ok=false if not found.
func (s *Store) GetUserByID(ctx context.Context, id string) (Identity, bool, error) {
	var idn Identity
	var org *string
	var roles []string
	var kyc string
	err := s.pool.QueryRow(ctx,
		`SELECT u.id, u.email, u.tenant_id, u.org_id, u.roles, t.is_paper, t.kyc_status
		 FROM users u JOIN tenants t ON t.id = u.tenant_id WHERE u.id = $1`, id).
		Scan(&idn.UserID, &idn.Email, &idn.TenantID, &org, &roles, &idn.IsPaper, &kyc)
	if errors.Is(err, pgx.ErrNoRows) {
		return Identity{}, false, nil
	}
	if err != nil {
		return Identity{}, false, err
	}
	if org != nil {
		idn.OrgID = *org
	}
	idn.Roles = toRoles(roles)
	idn.KYCStatus = domain.KYCStatus(kyc)
	return idn, true, nil
}

// CreateAPIKey persists a key's metadata + hash (the secret itself is never stored).
func (s *Store) CreateAPIKey(ctx context.Context, tenantID, name, prefix, hash string, scopes []string) (string, error) {
	id := uuid.NewString()
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO api_keys (id, tenant_id, name, prefix, hash, scopes) VALUES ($1,$2,$3,$4,$5,$6)`,
		id, tenantID, name, prefix, hash, scopes); err != nil {
		return "", fmt.Errorf("insert api key: %w", err)
	}
	return id, nil
}

// APIKeyRow is API-key metadata for listing (never includes the secret or hash).
type APIKeyRow struct {
	ID, Name, Prefix string
	Scopes           []string
	CreatedAt        time.Time
	Revoked          bool
}

// ListAPIKeys returns a tenant's keys (metadata only).
func (s *Store) ListAPIKeys(ctx context.Context, tenantID string) ([]APIKeyRow, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, prefix, scopes, created_at, (revoked_at IS NOT NULL)
		 FROM api_keys WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKeyRow
	for rows.Next() {
		var k APIKeyRow
		if err := rows.Scan(&k.ID, &k.Name, &k.Prefix, &k.Scopes, &k.CreatedAt, &k.Revoked); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// RevokeAPIKey marks a tenant's key revoked (idempotent; scoped to the tenant so one tenant can't
// revoke another's key).
func (s *Store) RevokeAPIKey(ctx context.Context, tenantID, id string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE api_keys SET revoked_at=now() WHERE id=$1 AND tenant_id=$2 AND revoked_at IS NULL`,
		id, tenantID)
	return err
}

// LookupAPIKey resolves a presented key hash to its tenant + scopes + is_paper (only if not revoked)
// and stamps last_used_at, in one round trip. is_paper comes from the key's tenant so the inference
// gateway can propagate paper/real into usage events. ok=false if unknown/revoked.
func (s *Store) LookupAPIKey(ctx context.Context, hash string) (tenantID string, scopes []string, isPaper bool, ok bool, err error) {
	err = s.pool.QueryRow(ctx,
		`WITH k AS (
		     UPDATE api_keys SET last_used_at=now() WHERE hash=$1 AND revoked_at IS NULL
		     RETURNING tenant_id, scopes
		 )
		 SELECT k.tenant_id, k.scopes, t.is_paper
		 FROM k JOIN tenants t ON t.id = k.tenant_id`, hash).Scan(&tenantID, &scopes, &isPaper)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, false, false, nil
	}
	if err != nil {
		return "", nil, false, false, err
	}
	return tenantID, scopes, isPaper, true, nil
}

// toRoles converts stored role strings to domain.Role.
func toRoles(ss []string) []domain.Role {
	out := make([]domain.Role, 0, len(ss))
	for _, s := range ss {
		out = append(out, domain.Role(s))
	}
	return out
}

// isUniqueViolation reports a Postgres unique-constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Purchase is a credit purchase order (F06). Amount is a fixed-point decimal string.
type Purchase struct {
	ID         string
	TenantID   string
	Amount     string
	CreditType string
	Currency   string
	Status     string
	IsPaper    bool
	CreatedAt  time.Time
	PaidAt     *time.Time
}

// CreatePurchase inserts a pending purchase linked to its Stripe checkout session.
func (s *Store) CreatePurchase(ctx context.Context, p Purchase, sessionID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO purchases (id, tenant_id, amount, credit_type, currency, status, stripe_session_id, is_paper)
		 VALUES ($1,$2,$3::numeric,$4,$5,'pending',$6,$7)`,
		p.ID, p.TenantID, p.Amount, p.CreditType, p.Currency, sessionID, p.IsPaper)
	if err != nil {
		return fmt.Errorf("insert purchase: %w", err)
	}
	return nil
}

// GetPurchaseBySession loads the purchase for a Stripe checkout session. ok=false if unknown.
func (s *Store) GetPurchaseBySession(ctx context.Context, sessionID string) (Purchase, bool, error) {
	var p Purchase
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, amount::text, credit_type, currency, status, is_paper, created_at, paid_at
		 FROM purchases WHERE stripe_session_id=$1`, sessionID).
		Scan(&p.ID, &p.TenantID, &p.Amount, &p.CreditType, &p.Currency, &p.Status, &p.IsPaper, &p.CreatedAt, &p.PaidAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Purchase{}, false, nil
	}
	if err != nil {
		return Purchase{}, false, err
	}
	return p, true, nil
}

// MarkPurchasePaid settles a pending purchase: status→paid, stamps the Stripe event id + paid_at.
// Idempotent — a replay (already paid) updates no rows and is not an error.
func (s *Store) MarkPurchasePaid(ctx context.Context, sessionID, eventID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE purchases SET status='paid', stripe_event_id=$2, paid_at=now()
		 WHERE stripe_session_id=$1 AND status='pending'`, sessionID, eventID)
	return err
}

// ListPurchases returns a tenant's purchase history, newest first.
func (s *Store) ListPurchases(ctx context.Context, tenantID string, limit int) ([]Purchase, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, amount::text, credit_type, currency, status, is_paper, created_at, paid_at
		 FROM purchases WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Purchase
	for rows.Next() {
		var p Purchase
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Amount, &p.CreditType, &p.Currency, &p.Status, &p.IsPaper, &p.CreatedAt, &p.PaidAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
