package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/trade1/credit-ledger/internal/domain"
)

// CashMovement is a single-leg cash movement. Amount is the SIGNED delta. Always paper (ADR-0004);
// the SQL CHECK rejects anything else.
type CashMovement struct {
	TenantID       string
	SubAccountID   string
	Currency       domain.Currency
	Operation      domain.CashOperation
	Amount         domain.Money
	ReferenceID    string
	IdempotencyKey string
	ConsumeOrderID string // see Movement.ConsumeOrderID
}

// applyCashLeg is the cash twin of applyLeg, inside an existing DB transaction: ensure the balance row
// → lock it → idempotency check under the lock → domain.ApplyCash → insert row → update balance and
// chain tip. It returns replayed=true when the key already exists (nothing written).
func applyCashLeg(ctx context.Context, dbtx pgx.Tx, m CashMovement) (domain.CashTx, bool, error) {
	sub := nullable(m.SubAccountID)
	if _, err := dbtx.Exec(ctx,
		`INSERT INTO cash_balances (balance_id, tenant_id, sub_account_id, currency, balance, is_paper, last_chain_hash)
		 VALUES ($1,$2,$3,$4,0,true,'')
		 ON CONFLICT (tenant_id, sub_account_id, currency, is_paper) DO NOTHING`,
		uuid.NewString(), m.TenantID, sub, string(m.Currency)); err != nil {
		return domain.CashTx{}, false, fmt.Errorf("ensure cash balance: %w", err)
	}
	var balStr, lockStr, prevHash string
	if err := dbtx.QueryRow(ctx,
		`SELECT balance::text, locked_amount::text, last_chain_hash FROM cash_balances
		 WHERE tenant_id=$1 AND sub_account_id IS NOT DISTINCT FROM $2 AND currency=$3 AND is_paper
		 FOR UPDATE`,
		m.TenantID, sub, string(m.Currency)).Scan(&balStr, &lockStr, &prevHash); err != nil {
		return domain.CashTx{}, false, fmt.Errorf("lock cash balance: %w", err)
	}
	if m.IdempotencyKey != "" {
		if existing, ok, err := findCashByIdem(ctx, dbtx, m.TenantID, string(m.Operation), m.IdempotencyKey); err != nil {
			return domain.CashTx{}, false, err
		} else if ok {
			return existing, true, nil
		}
	}
	bal, err := domain.ParseMoney(balStr)
	if err != nil {
		return domain.CashTx{}, false, fmt.Errorf("parse cash balance %q: %w", balStr, err)
	}
	locked, err := domain.ParseMoney(lockStr)
	if err != nil {
		return domain.CashTx{}, false, fmt.Errorf("parse cash locked %q: %w", lockStr, err)
	}
	locked, err = consumeReservation(ctx, dbtx, m.ConsumeOrderID, "cash", m.TenantID, string(m.Currency), m.Amount, m.ReferenceID, locked, domain.ErrInsufficientCash)
	if err != nil {
		return domain.CashTx{}, false, err
	}
	applied, err := domain.ApplyCash(bal, prevHash, domain.CashTx{
		TxID: uuid.NewString(), TenantID: m.TenantID, SubAccountID: m.SubAccountID, Currency: m.Currency,
		Operation: m.Operation, Amount: m.Amount, ReferenceID: m.ReferenceID, IdempotencyKey: m.IdempotencyKey,
		IsPaper: true, CreatedAt: time.Now().UTC().Truncate(time.Microsecond), // Postgres stores micros
	})
	if err != nil {
		return domain.CashTx{}, false, err // ErrInsufficientCash
	}
	if m.Amount.IsNegative() && applied.BalanceAfter.Cmp(locked) < 0 {
		return domain.CashTx{}, false, domain.ErrInsufficientCash // would spend cash reserved for an open order
	}
	if _, err := dbtx.Exec(ctx,
		`INSERT INTO cash_transactions
		   (tx_id, tenant_id, sub_account_id, currency, operation, amount, reference_id, idempotency_key,
		    balance_before, balance_after, is_paper, created_at, chain_hash)
		 VALUES ($1,$2,$3,$4,$5,$6::numeric,$7,$8,$9::numeric,$10::numeric,true,$11,$12)`,
		applied.TxID, applied.TenantID, sub, string(applied.Currency), string(applied.Operation),
		applied.Amount.String(), nullable(applied.ReferenceID), nullable(applied.IdempotencyKey),
		applied.BalanceBefore.String(), applied.BalanceAfter.String(), applied.CreatedAt, applied.ChainHash); err != nil {
		return domain.CashTx{}, false, fmt.Errorf("insert cash tx: %w", err)
	}
	if _, err := dbtx.Exec(ctx,
		`UPDATE cash_balances SET balance=$1::numeric, locked_amount=$6::numeric, last_chain_hash=$2, updated_at=now()
		 WHERE tenant_id=$3 AND sub_account_id IS NOT DISTINCT FROM $4 AND currency=$5 AND is_paper`,
		applied.BalanceAfter.String(), applied.ChainHash, m.TenantID, sub, string(m.Currency), locked.String()); err != nil {
		return domain.CashTx{}, false, fmt.Errorf("update cash balance: %w", err)
	}
	return applied, false, nil
}

// findCashByIdem returns a prior cash row with the same (tenant, operation, key), if any.
func findCashByIdem(ctx context.Context, dbtx pgx.Tx, tenantID, op, key string) (domain.CashTx, bool, error) {
	var t domain.CashTx
	var cur, amtStr, afterStr string
	err := dbtx.QueryRow(ctx,
		`SELECT tx_id, tenant_id, coalesce(sub_account_id::text,''), currency, amount::text,
		        coalesce(reference_id,''), balance_after::text, created_at, chain_hash
		 FROM cash_transactions WHERE tenant_id=$1 AND operation=$2 AND idempotency_key=$3`, tenantID, op, key).
		Scan(&t.TxID, &t.TenantID, &t.SubAccountID, &cur, &amtStr, &t.ReferenceID, &afterStr, &t.CreatedAt, &t.ChainHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CashTx{}, false, nil
	}
	if err != nil {
		return domain.CashTx{}, false, err
	}
	t.Currency, t.Operation, t.IdempotencyKey, t.IsPaper = domain.Currency(cur), domain.CashOperation(op), key, true
	t.Amount, _ = domain.ParseMoney(amtStr)
	t.BalanceAfter, _ = domain.ParseMoney(afterStr)
	return t, true, nil
}

// GrantPaperCash credits a tenant's starting paper cash, once per idempotency key. It returns the
// original row on replay (replayed=true), so activation can retry freely.
func (s *Store) GrantPaperCash(ctx context.Context, tenantID string, cur domain.Currency, amount domain.Money, key string) (domain.CashTx, bool, error) {
	dbtx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.CashTx{}, false, err
	}
	defer dbtx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	tx, replayed, err := applyCashLeg(ctx, dbtx, CashMovement{
		TenantID: tenantID, Currency: cur, Operation: domain.CashPaperGrant, Amount: amount,
		ReferenceID: key, IdempotencyKey: key,
	})
	if err != nil {
		return domain.CashTx{}, false, err
	}
	if err := dbtx.Commit(ctx); err != nil {
		return domain.CashTx{}, false, fmt.Errorf("commit: %w", err)
	}
	return tx, replayed, nil
}

// CashBalance is a read model of one cash balance row.
type CashBalance struct {
	Currency     domain.Currency
	Balance      domain.Money
	Locked       domain.Money
	SubAccountID string
}

// GetCashBalances returns a tenant's paper cash balances.
func (s *Store) GetCashBalances(ctx context.Context, tenantID string) ([]CashBalance, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT currency, balance::text, locked_amount::text, coalesce(sub_account_id::text,'')
		 FROM cash_balances WHERE tenant_id=$1 AND is_paper ORDER BY currency, sub_account_id NULLS FIRST`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CashBalance
	for rows.Next() {
		var cur, balStr, lockStr, sub string
		if err := rows.Scan(&cur, &balStr, &lockStr, &sub); err != nil {
			return nil, err
		}
		bal, _ := domain.ParseMoney(balStr)
		lock, _ := domain.ParseMoney(lockStr)
		out = append(out, CashBalance{domain.Currency(cur), bal, lock, sub})
	}
	return out, rows.Err()
}

// ListCashTransactions returns a tenant's paper cash rows newest first, up to limit (1–200).
func (s *Store) ListCashTransactions(ctx context.Context, tenantID string, limit int) ([]domain.CashTx, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx,
		`SELECT tx_id, coalesce(sub_account_id::text,''), currency, operation, amount::text,
		        coalesce(reference_id,''), balance_after::text, created_at, chain_hash
		 FROM cash_transactions WHERE tenant_id=$1 AND is_paper
		 ORDER BY created_at DESC, tx_id DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CashTx
	for rows.Next() {
		var t domain.CashTx
		var cur, op, amtStr, balStr string
		if err := rows.Scan(&t.TxID, &t.SubAccountID, &cur, &op, &amtStr, &t.ReferenceID, &balStr, &t.CreatedAt, &t.ChainHash); err != nil {
			return nil, err
		}
		t.TenantID, t.Currency, t.Operation, t.IsPaper = tenantID, domain.Currency(cur), domain.CashOperation(op), true
		t.Amount, _ = domain.ParseMoney(amtStr)
		t.BalanceAfter, _ = domain.ParseMoney(balStr)
		out = append(out, t)
	}
	return out, rows.Err()
}

// VerifyCashChain re-derives every cash balance's chain for a tenant, like VerifyChain for credits.
func (s *Store) VerifyCashChain(ctx context.Context, tenantID string) (ok bool, checked int, firstBadTxID string, err error) {
	rows, err := s.pool.Query(ctx,
		`SELECT tx_id, coalesce(sub_account_id::text,''), currency, operation, amount::text,
		        coalesce(reference_id,''), balance_before::text, balance_after::text, created_at, chain_hash
		 FROM cash_transactions WHERE tenant_id=$1
		 ORDER BY currency, sub_account_id NULLS FIRST, created_at, tx_id`, tenantID)
	if err != nil {
		return false, 0, "", err
	}
	defer rows.Close()
	prev := map[string]string{}
	for rows.Next() {
		var t domain.CashTx
		var cur, op, amtStr, beforeStr, afterStr string
		if err := rows.Scan(&t.TxID, &t.SubAccountID, &cur, &op, &amtStr, &t.ReferenceID,
			&beforeStr, &afterStr, &t.CreatedAt, &t.ChainHash); err != nil {
			return false, checked, "", err
		}
		t.TenantID, t.Currency, t.Operation, t.IsPaper = tenantID, domain.Currency(cur), domain.CashOperation(op), true
		t.Amount, _ = domain.ParseMoney(amtStr)
		t.BalanceBefore, _ = domain.ParseMoney(beforeStr)
		t.BalanceAfter, _ = domain.ParseMoney(afterStr)
		scope := cur + "|" + t.SubAccountID
		if domain.VerifyCashChain(prev[scope], []domain.CashTx{t}) != -1 {
			return false, checked, t.TxID, nil
		}
		prev[scope] = t.ChainHash
		checked++
	}
	return true, checked, "", rows.Err()
}
