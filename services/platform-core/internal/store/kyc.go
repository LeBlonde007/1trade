package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/exascale/platform-core/internal/domain"
	"github.com/jackc/pgx/v5"
)

// KYCRecord is a tenant's identity-verification state (F22). PII is intentionally minimal.
type KYCRecord struct {
	Status      domain.KYCStatus
	LegalName   string
	Country     string
	EntityType  string
	SubmittedAt *time.Time
	ReviewedAt  *time.Time
}

// GetKYC loads a tenant's KYC record. ok=false if the tenant does not exist.
func (s *Store) GetKYC(ctx context.Context, tenantID string) (KYCRecord, bool, error) {
	var k KYCRecord
	var status string
	var legalName, country, entityType *string
	err := s.pool.QueryRow(ctx,
		`SELECT kyc_status, kyc_legal_name, kyc_country, kyc_entity_type, kyc_submitted_at, kyc_reviewed_at
		 FROM tenants WHERE id=$1`, tenantID).
		Scan(&status, &legalName, &country, &entityType, &k.SubmittedAt, &k.ReviewedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return KYCRecord{}, false, nil
	}
	if err != nil {
		return KYCRecord{}, false, err
	}
	k.Status = domain.KYCStatus(status)
	if legalName != nil {
		k.LegalName = *legalName
	}
	if country != nil {
		k.Country = *country
	}
	if entityType != nil {
		k.EntityType = *entityType
	}
	return k, true, nil
}

// KYCSubmission is the minimal PII captured at submission.
type KYCSubmission struct {
	LegalName  string
	Country    string
	EntityType string
}

// SubmitKYC records a submission and sets the resulting status: `verified` when autoApprove (dev/
// sandbox — no compliance back-office), else `pending`. The current status is part of the WHERE so a
// duplicate/concurrent submit on an already pending/verified record updates no rows (ok=false) rather
// than overwriting a decided record. Returns the new status.
func (s *Store) SubmitKYC(ctx context.Context, tenantID string, sub KYCSubmission, autoApprove bool) (newStatus domain.KYCStatus, ok bool, err error) {
	newStatus = domain.KYCPending
	var reviewedAt any // NULL until reviewed
	if autoApprove {
		newStatus = domain.KYCVerified
		reviewedAt = time.Now()
	}
	tag, err := s.pool.Exec(ctx,
		`UPDATE tenants
		 SET kyc_status=$2, kyc_legal_name=$3, kyc_country=$4, kyc_entity_type=$5,
		     kyc_submitted_at=now(), kyc_reviewed_at=$6
		 WHERE id=$1 AND kyc_status IN ('unverified','rejected')`,
		tenantID, string(newStatus), sub.LegalName, sub.Country, sub.EntityType, reviewedAt)
	if err != nil {
		return "", false, fmt.Errorf("submit kyc: %w", err)
	}
	return newStatus, tag.RowsAffected() == 1, nil
}

// ReviewKYC applies a compliance decision (verified|rejected) to a pending submission, guarded on
// kyc_status='pending' so only a pending record is decided. ok=false if nothing was pending.
func (s *Store) ReviewKYC(ctx context.Context, tenantID string, decision domain.KYCStatus) (ok bool, err error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE tenants SET kyc_status=$2, kyc_reviewed_at=now() WHERE id=$1 AND kyc_status='pending'`,
		tenantID, string(decision))
	if err != nil {
		return false, fmt.Errorf("review kyc: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
