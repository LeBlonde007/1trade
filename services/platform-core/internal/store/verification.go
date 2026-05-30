package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// SetVerifyToken stores the hash of a fresh email-verification token for a user (resets verified).
func (s *Store) SetVerifyToken(ctx context.Context, userID, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET verify_token=$2, email_verified=false WHERE id=$1`, userID, tokenHash)
	return err
}

// VerifyEmail consumes a token hash: marks the user verified and clears the token. ok=false if the
// token is unknown or already used. Returns the user + tenant (for the audit row).
func (s *Store) VerifyEmail(ctx context.Context, tokenHash string) (userID, tenantID string, isPaper, ok bool, err error) {
	err = s.pool.QueryRow(ctx,
		`WITH v AS (
		     UPDATE users SET email_verified=true, verify_token=NULL
		     WHERE verify_token=$1 AND email_verified=false
		     RETURNING id, tenant_id
		 )
		 SELECT v.id, v.tenant_id, t.is_paper FROM v JOIN tenants t ON t.id = v.tenant_id`, tokenHash).
		Scan(&userID, &tenantID, &isPaper)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, false, nil
	}
	if err != nil {
		return "", "", false, false, err
	}
	return userID, tenantID, isPaper, true, nil
}

// Budget is a tenant's monthly credit budget.
type Budget struct {
	CreditType   string
	MonthlyLimit string
	UpdatedAt    time.Time
}

// GetBudget returns the tenant's budget. ok=false if none set.
func (s *Store) GetBudget(ctx context.Context, tenantID string) (Budget, bool, error) {
	var b Budget
	err := s.pool.QueryRow(ctx, `SELECT credit_type, monthly_limit::text, updated_at FROM budgets WHERE tenant_id=$1`, tenantID).
		Scan(&b.CreditType, &b.MonthlyLimit, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Budget{}, false, nil
	}
	if err != nil {
		return Budget{}, false, err
	}
	return b, true, nil
}

// SetBudget upserts the tenant's monthly budget.
func (s *Store) SetBudget(ctx context.Context, tenantID, creditType, limit string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO budgets (tenant_id, credit_type, monthly_limit) VALUES ($1,$2,$3::numeric)
		 ON CONFLICT (tenant_id) DO UPDATE SET credit_type=$2, monthly_limit=$3::numeric, updated_at=now()`,
		tenantID, creditType, limit)
	return err
}
