package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// MFA errors.
var (
	ErrMFAEnabled    = errors.New("store: two-factor authentication is already on")
	ErrMFANotPending = errors.New("store: no two-factor setup in progress")
)

// MFA lockout policy: this many wrong codes lock the second step for mfaLockFor.
const (
	mfaMaxFailures = 5
	mfaLockFor     = 15 * time.Minute
)

// MFAState is a user's two-factor state.
type MFAState struct {
	Email        string
	Enabled      bool
	Secret       []byte // sealed
	Pending      []byte // sealed
	LastStep     int64
	LockedUntil  *time.Time
	RecoveryLeft int
}

// GetMFA loads a user's two-factor state.
func (s *Store) GetMFA(ctx context.Context, userID string) (MFAState, error) {
	var m MFAState
	err := s.pool.QueryRow(ctx, `SELECT email, mfa_enabled, mfa_secret, mfa_pending_secret, mfa_last_step, mfa_locked_until,
		(SELECT count(*) FROM mfa_recovery_codes WHERE user_id=u.id AND used_at IS NULL)::int
		FROM users u WHERE id=$1`, userID).Scan(&m.Email, &m.Enabled, &m.Secret, &m.Pending, &m.LastStep, &m.LockedUntil, &m.RecoveryLeft)
	if errors.Is(err, pgx.ErrNoRows) {
		return MFAState{}, ErrNotMember
	}
	return m, err
}

// SetPendingMFA stores a sealed secret awaiting confirmation. Refused while 2FA is on.
func (s *Store) SetPendingMFA(ctx context.Context, userID string, sealed []byte) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET mfa_pending_secret=$2 WHERE id=$1 AND NOT mfa_enabled`, userID, sealed)
	if err != nil {
		return fmt.Errorf("mfa setup: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMFAEnabled
	}
	return nil
}

// EnableMFA turns 2FA on with the pending secret, records the step that confirmed it (so that code
// cannot be reused), and replaces the recovery codes.
func (s *Store) EnableMFA(ctx context.Context, userID string, step int64, codeHashes []string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	tag, err := tx.Exec(ctx, `UPDATE users SET mfa_enabled=true, mfa_secret=mfa_pending_secret, mfa_pending_secret=NULL,
		mfa_last_step=$2, mfa_failures=0, mfa_locked_until=NULL WHERE id=$1 AND NOT mfa_enabled AND mfa_pending_secret IS NOT NULL`, userID, step)
	if err != nil {
		return fmt.Errorf("mfa enable: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMFANotPending
	}
	if _, err := tx.Exec(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id=$1`, userID); err != nil {
		return err
	}
	for _, h := range codeHashes {
		if _, err := tx.Exec(ctx, `INSERT INTO mfa_recovery_codes (user_id, code_hash) VALUES ($1,$2)`, userID, h); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// DisableMFA turns 2FA off and forgets the secret and recovery codes.
func (s *Store) DisableMFA(ctx context.Context, userID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	if _, err := tx.Exec(ctx, `UPDATE users SET mfa_enabled=false, mfa_secret=NULL, mfa_pending_secret=NULL,
		mfa_last_step=0, mfa_failures=0, mfa_locked_until=NULL WHERE id=$1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM mfa_recovery_codes WHERE user_id=$1`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ClaimMFAStep records a verified code's time step. It succeeds only if the step is newer than the
// last one used, atomically, so two concurrent submissions of the same code cannot both pass.
func (s *Store) ClaimMFAStep(ctx context.Context, userID string, step int64) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET mfa_last_step=$2, mfa_failures=0 WHERE id=$1 AND mfa_last_step < $2`, userID, step)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// UseRecoveryCode spends a recovery code (once).
func (s *Store) UseRecoveryCode(ctx context.Context, userID, codeHash string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE mfa_recovery_codes SET used_at=now() WHERE user_id=$1 AND code_hash=$2 AND used_at IS NULL`, userID, codeHash)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 1 {
		_, err = s.pool.Exec(ctx, `UPDATE users SET mfa_failures=0 WHERE id=$1`, userID)
	}
	return tag.RowsAffected() == 1, err
}

// RecordMFAFailure counts a wrong code; the fifth locks the second step for 15 minutes.
func (s *Store) RecordMFAFailure(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET
		mfa_locked_until = CASE WHEN mfa_failures + 1 >= $2 THEN now() + $3::interval ELSE mfa_locked_until END,
		mfa_failures     = CASE WHEN mfa_failures + 1 >= $2 THEN 0 ELSE mfa_failures + 1 END
		WHERE id=$1`, userID, mfaMaxFailures, fmt.Sprintf("%d seconds", int(mfaLockFor.Seconds())))
	return err
}
