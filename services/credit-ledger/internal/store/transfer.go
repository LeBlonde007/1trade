package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/trade1/credit-ledger/internal/domain"
)

// ErrSameAccount: a transfer must move credits between two different balances.
var ErrSameAccount = errors.New("transfer: from and to are the same balance")

// Transfer moves credits of one type between two balances of the same tenant and side (paper or
// real): its main balance ("" sub-account) and a sub-account, or two sub-accounts.
type Transfer struct {
	TenantID       string
	FromSub        string
	ToSub          string
	CreditType     domain.CreditType
	Amount         domain.Money
	IsPaper        bool
	IdempotencyKey string
	ReferenceID    string
}

// ApplyTransfer debits the source and credits the destination as two chained legs in one DB
// transaction, so credits are never created or lost: both land or neither does. Legs have distinct
// idempotency keys (key:from / key:to), so a replay returns the original pair. The source must have
// the credits available (not reserved for an open order): domain.ErrInsufficientCredit otherwise.
func (s *Store) ApplyTransfer(ctx context.Context, t Transfer) (debit, credit domain.Transaction, err error) {
	if t.FromSub == t.ToSub {
		return debit, credit, ErrSameAccount
	}
	dbtx, err := s.pool.Begin(ctx)
	if err != nil {
		return debit, credit, err
	}
	defer dbtx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	debit, err = applyLeg(ctx, dbtx, Movement{
		TenantID: t.TenantID, SubAccountID: t.FromSub, CreditType: t.CreditType, Operation: domain.OpTransfer,
		Amount: domain.Zero().Sub(t.Amount), ReferenceID: t.ReferenceID, IdempotencyKey: t.IdempotencyKey + ":from", IsPaper: t.IsPaper,
	})
	if err != nil {
		return debit, credit, err
	}
	credit, err = applyLeg(ctx, dbtx, Movement{
		TenantID: t.TenantID, SubAccountID: t.ToSub, CreditType: t.CreditType, Operation: domain.OpTransfer,
		Amount: t.Amount, ReferenceID: t.ReferenceID, IdempotencyKey: t.IdempotencyKey + ":to", IsPaper: t.IsPaper,
	})
	if err != nil {
		return debit, credit, err
	}
	if err := dbtx.Commit(ctx); err != nil {
		return debit, credit, fmt.Errorf("commit: %w", err)
	}
	return debit, credit, nil
}
