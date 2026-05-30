package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/exascale/credit-ledger/internal/domain"
	"github.com/jackc/pgx/v5"
)

// ErrNoRate is returned when no conversion rate exists for a (from, to) pair.
var ErrNoRate = errors.New("no conversion rate for this pair")

// Rate is a published conversion rate (decimal strings).
type Rate struct {
	From   string
	To     string
	Rate   string
	Spread string
}

// ListRates returns all published conversion rates.
func (s *Store) ListRates(ctx context.Context) ([]Rate, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT from_type, to_type, rate::text, spread::text FROM conversion_rates ORDER BY from_type, to_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Rate
	for rows.Next() {
		var r Rate
		if err := rows.Scan(&r.From, &r.To, &r.Rate, &r.Spread); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Conversion is a request to convert Amount of From into To for a tenant.
type Conversion struct {
	TenantID       string
	SubAccountID   string
	From           domain.CreditType
	To             domain.CreditType
	Amount         domain.Money
	IsPaper        bool
	IdempotencyKey string
	ReferenceID    string
}

// ApplyConversion executes a conversion atomically: in ONE DB transaction it reads the pair's rate,
// computes the target (amount × rate × (1−spread), floored), then burns `from` and mints `to` as two
// chained legs (each extends its own balance's hash chain). Idempotency keys are distinct per leg
// (key:from / key:to) so the second leg doesn't replay the first. Returns both legs + the applied
// rate/spread. ErrNoRate for an unknown pair; domain.ErrInsufficientCredit when the source is short.
func (s *Store) ApplyConversion(ctx context.Context, c Conversion) (debit, credit domain.Transaction, rate, spread string, err error) {
	dbtx, err := s.pool.Begin(ctx)
	if err != nil {
		return
	}
	defer dbtx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if err = dbtx.QueryRow(ctx,
		`SELECT rate::text, spread::text FROM conversion_rates WHERE from_type=$1 AND to_type=$2`,
		string(c.From), string(c.To)).Scan(&rate, &spread); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = ErrNoRate
		}
		return
	}
	target, err := domain.ConvertedAmount(c.Amount, rate, spread)
	if err != nil {
		return
	}

	// Leg 1: burn `from` (negative delta). domain.Apply returns ErrInsufficientCredit if it would go
	// negative — that's the 402 path.
	debit, err = applyLeg(ctx, dbtx, Movement{
		TenantID: c.TenantID, SubAccountID: c.SubAccountID, CreditType: c.From, Operation: domain.OpConversion,
		Amount: domain.Zero().Sub(c.Amount), ReferenceID: c.ReferenceID, IdempotencyKey: c.IdempotencyKey + ":from", IsPaper: c.IsPaper,
	})
	if err != nil {
		return
	}

	// Leg 2: mint `to` (positive delta).
	credit, err = applyLeg(ctx, dbtx, Movement{
		TenantID: c.TenantID, SubAccountID: c.SubAccountID, CreditType: c.To, Operation: domain.OpConversion,
		Amount: target, ReferenceID: c.ReferenceID, IdempotencyKey: c.IdempotencyKey + ":to", IsPaper: c.IsPaper,
	})
	if err != nil {
		return
	}

	if err = dbtx.Commit(ctx); err != nil {
		err = fmt.Errorf("commit: %w", err)
	}
	return
}
