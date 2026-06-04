// Package store is the Postgres binding for the credit ledger. It enforces the F05 invariants at
// the IO boundary: balance update + transaction insert happen in ONE DB transaction; the per-balance
// row lock serialises writes so the hash chain stays consistent; idempotency keys make retries safe.
// All money math + chain computation comes from internal/domain (pure).
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/exascale/credit-ledger/internal/domain"
	"github.com/exascale/credit-ledger/internal/metrics"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

// Movement is a single-leg credit movement. Amount is the SIGNED delta (negative for debits/burns).
type Movement struct {
	TenantID       string
	SubAccountID   string // "" when none
	CreditType     domain.CreditType
	Operation      domain.Operation
	Amount         domain.Money
	ReferenceID    string
	IdempotencyKey string
	IsPaper        bool
}

// ApplyMovement applies one movement atomically: lock the balance row, dedupe on the idempotency
// key, run the pure domain Apply, then insert the tx + update the balance in the same DB transaction.
// Returns the resulting Transaction (or the original on an idempotent replay).
func (s *Store) ApplyMovement(ctx context.Context, m Movement) (domain.Transaction, error) {
	dbtx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Transaction{}, err
	}
	defer dbtx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	out, err := applyLeg(ctx, dbtx, m)
	if err != nil {
		return domain.Transaction{}, err
	}
	if err := dbtx.Commit(ctx); err != nil {
		return domain.Transaction{}, fmt.Errorf("commit: %w", err)
	}
	return out, nil
}

// applyLeg performs one locked movement inside an existing DB transaction (reused by conversion in
// F07, which runs two legs in one tx). Order: ensure balance row → lock it FOR UPDATE → idempotency
// check (authoritative under the lock) → domain.Apply → insert tx → update balance + chain tip.
func applyLeg(ctx context.Context, dbtx pgx.Tx, m Movement) (domain.Transaction, error) {
	sub := nullable(m.SubAccountID)

	// Ensure the balance row exists (no-op if it already does).
	if _, err := dbtx.Exec(ctx,
		`INSERT INTO credit_balances (balance_id, tenant_id, sub_account_id, credit_type, balance, is_paper, last_chain_hash)
		 VALUES ($1,$2,$3,$4,0,$5,'')
		 ON CONFLICT (tenant_id, sub_account_id, credit_type, is_paper) DO NOTHING`,
		uuid.NewString(), m.TenantID, sub, string(m.CreditType), m.IsPaper); err != nil {
		return domain.Transaction{}, fmt.Errorf("ensure balance: %w", err)
	}

	// Lock the balance row; serialises writes to this balance so the chain tip is consistent.
	var balStr, prevHash string
	if err := dbtx.QueryRow(ctx,
		`SELECT balance::text, last_chain_hash FROM credit_balances
		 WHERE tenant_id=$1 AND sub_account_id IS NOT DISTINCT FROM $2 AND credit_type=$3 AND is_paper=$4
		 FOR UPDATE`,
		m.TenantID, sub, string(m.CreditType), m.IsPaper).Scan(&balStr, &prevHash); err != nil {
		return domain.Transaction{}, fmt.Errorf("lock balance: %w", err)
	}

	// Idempotency: under the lock, a prior tx with this (tenant, operation, key) is authoritative.
	if m.IdempotencyKey != "" {
		if existing, ok, err := findByIdem(ctx, dbtx, m.TenantID, string(m.Operation), m.IdempotencyKey); err != nil {
			return domain.Transaction{}, err
		} else if ok {
			return existing, nil // replay — do not apply again
		}
	}

	bal, err := domain.ParseMoney(balStr)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("parse balance %q: %w", balStr, err)
	}

	applied, err := domain.Apply(bal, prevHash, domain.Transaction{
		TxID: uuid.NewString(), TenantID: m.TenantID, SubAccountID: m.SubAccountID,
		CreditType: m.CreditType, Operation: m.Operation, Amount: m.Amount,
		ReferenceID: m.ReferenceID, IdempotencyKey: m.IdempotencyKey, IsPaper: m.IsPaper,
		// Truncate to microseconds — Postgres timestamptz stores micros, so hashing at nanosecond
		// precision would make the chain unverifiable after a reload (hash input must == stored).
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	})
	if err != nil {
		return domain.Transaction{}, err // e.g. ErrInsufficientCredit
	}

	if _, err := dbtx.Exec(ctx,
		`INSERT INTO credit_transactions
		   (tx_id, tenant_id, sub_account_id, credit_type, operation, amount, reference_id,
		    idempotency_key, balance_before, balance_after, is_paper, created_at, chain_hash)
		 VALUES ($1,$2,$3,$4,$5,$6::numeric,$7,$8,$9::numeric,$10::numeric,$11,$12,$13)`,
		applied.TxID, applied.TenantID, sub, string(applied.CreditType), string(applied.Operation),
		applied.Amount.String(), nullable(applied.ReferenceID), nullable(applied.IdempotencyKey),
		applied.BalanceBefore.String(), applied.BalanceAfter.String(), applied.IsPaper,
		applied.CreatedAt, applied.ChainHash); err != nil {
		return domain.Transaction{}, fmt.Errorf("insert tx: %w", err)
	}
	// Count the written transaction. Idempotent replays return before this insert, so they are not
	// counted; a rare post-insert commit failure may over-count by one (acceptable for an advisory
	// counter). Labels are bounded (operation × credit-type enum).
	metrics.TransactionsTotal.WithLabelValues(string(applied.Operation), string(applied.CreditType)).Inc()

	if _, err := dbtx.Exec(ctx,
		`UPDATE credit_balances SET balance=$1::numeric, last_chain_hash=$2, updated_at=now()
		 WHERE tenant_id=$3 AND sub_account_id IS NOT DISTINCT FROM $4 AND credit_type=$5 AND is_paper=$6`,
		applied.BalanceAfter.String(), applied.ChainHash,
		m.TenantID, sub, string(m.CreditType), m.IsPaper); err != nil {
		return domain.Transaction{}, fmt.Errorf("update balance: %w", err)
	}
	return applied, nil
}

// Balance is a read model of one balance row.
type Balance struct {
	CreditType   domain.CreditType
	Balance      domain.Money
	Locked       domain.Money
	IsPaper      bool
	SubAccountID string
}

// GetBalances returns all balances for a tenant on the given ledger (paper or real).
func (s *Store) GetBalances(ctx context.Context, tenantID string, isPaper bool) ([]Balance, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT credit_type, balance::text, locked_amount::text, coalesce(sub_account_id::text,'')
		 FROM credit_balances WHERE tenant_id=$1 AND is_paper=$2 ORDER BY credit_type`,
		tenantID, isPaper)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Balance
	for rows.Next() {
		var ct, balStr, lockStr, subAcct string
		if err := rows.Scan(&ct, &balStr, &lockStr, &subAcct); err != nil {
			return nil, err
		}
		bal, _ := domain.ParseMoney(balStr)
		lock, _ := domain.ParseMoney(lockStr)
		out = append(out, Balance{domain.CreditType(ct), bal, lock, isPaper, subAcct})
	}
	return out, rows.Err()
}

// ListTransactions returns a tenant's transactions newest-first, up to limit.
func (s *Store) ListTransactions(ctx context.Context, tenantID string, isPaper bool, limit int) ([]domain.Transaction, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx,
		`SELECT tx_id, coalesce(sub_account_id::text,''), credit_type, operation, amount::text,
		        coalesce(reference_id,''), balance_after::text, is_paper, created_at, chain_hash
		 FROM credit_transactions WHERE tenant_id=$1 AND is_paper=$2
		 ORDER BY created_at DESC, tx_id DESC LIMIT $3`,
		tenantID, isPaper, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		var ct, op, amtStr, balStr string
		if err := rows.Scan(&t.TxID, &t.SubAccountID, &ct, &op, &amtStr,
			&t.ReferenceID, &balStr, &t.IsPaper, &t.CreatedAt, &t.ChainHash); err != nil {
			return nil, err
		}
		t.TenantID = tenantID
		t.CreditType = domain.CreditType(ct)
		t.Operation = domain.Operation(op)
		t.Amount, _ = domain.ParseMoney(amtStr)
		t.BalanceAfter, _ = domain.ParseMoney(balStr)
		out = append(out, t)
	}
	return out, rows.Err()
}

// VerifyChain re-derives every balance's hash chain and reports the first broken tx, if any.
// Returns ok=true when all chains are intact. checked = transactions inspected.
func (s *Store) VerifyChain(ctx context.Context, tenantID string) (ok bool, checked int, firstBadTxID string, err error) {
	// Group by balance scope; each balance is its own chain (genesis "").
	rows, err := s.pool.Query(ctx,
		`SELECT tx_id, coalesce(sub_account_id::text,''), credit_type, operation, amount::text,
		        coalesce(reference_id,''), balance_before::text, balance_after::text, is_paper,
		        created_at, chain_hash
		 FROM credit_transactions WHERE tenant_id=$1
		 ORDER BY credit_type, sub_account_id NULLS FIRST, is_paper, created_at, tx_id`,
		tenantID)
	if err != nil {
		return false, 0, "", err
	}
	defer rows.Close()

	type scope struct {
		ct, sub string
		paper   bool
	}
	prev := map[scope]string{}
	for rows.Next() {
		var t domain.Transaction
		var ct, op, amtStr, beforeStr, afterStr string
		if err := rows.Scan(&t.TxID, &t.SubAccountID, &ct, &op, &amtStr, &t.ReferenceID,
			&beforeStr, &afterStr, &t.IsPaper, &t.CreatedAt, &t.ChainHash); err != nil {
			return false, checked, "", err
		}
		t.TenantID = tenantID
		t.CreditType = domain.CreditType(ct)
		t.Operation = domain.Operation(op)
		t.Amount, _ = domain.ParseMoney(amtStr)
		t.BalanceBefore, _ = domain.ParseMoney(beforeStr)
		t.BalanceAfter, _ = domain.ParseMoney(afterStr)

		sc := scope{ct, t.SubAccountID, t.IsPaper}
		p, seen := prev[sc]
		if !seen {
			p = "" // genesis per balance
		}
		if domain.VerifyChain(p, []domain.Transaction{t}) != -1 {
			return false, checked, t.TxID, nil
		}
		prev[sc] = t.ChainHash
		checked++
	}
	return true, checked, "", rows.Err()
}

// findByIdem returns a prior transaction with the same (tenant, operation, idempotency_key), if
// any. Scoped per tenant so one tenant's key can never return another tenant's transaction.
func findByIdem(ctx context.Context, dbtx pgx.Tx, tenantID, op, key string) (domain.Transaction, bool, error) {
	var t domain.Transaction
	var ct, amtStr, afterStr string
	err := dbtx.QueryRow(ctx,
		`SELECT tx_id, tenant_id, coalesce(sub_account_id::text,''), credit_type, amount::text,
		        coalesce(reference_id,''), balance_after::text, is_paper, created_at, chain_hash
		 FROM credit_transactions WHERE tenant_id=$1 AND operation=$2 AND idempotency_key=$3`, tenantID, op, key).
		Scan(&t.TxID, &t.TenantID, &t.SubAccountID, &ct, &amtStr, &t.ReferenceID,
			&afterStr, &t.IsPaper, &t.CreatedAt, &t.ChainHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Transaction{}, false, nil
	}
	if err != nil {
		return domain.Transaction{}, false, err
	}
	t.Operation = domain.Operation(op)
	t.IdempotencyKey = key
	t.CreditType = domain.CreditType(ct)
	t.Amount, _ = domain.ParseMoney(amtStr)
	t.BalanceAfter, _ = domain.ParseMoney(afterStr)
	return t, true, nil
}

// nullable maps "" to a SQL NULL (for sub_account_id, reference_id, idempotency_key).
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// IsUniqueViolation reports whether err is a Postgres unique-constraint violation (SQLSTATE 23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
