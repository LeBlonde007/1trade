// Package reserve is reserved GPU capacity (compute.yaml v1.2, F14): a tenant prepays for N GPUs of
// one tier over a term, in that tier's GPU credits at the term discount, and 1Trade sets those GPUs
// aside in the pool (a hold) for the whole term. The tenant's instances draw from the hold first and
// their usage there bills zero; on-demand work can never take held GPUs.
//
// A reservation is only sold against free capacity (the hold is taken before payment), and it is
// only live once the ledger debit succeeded. The debit is idempotent on the reservation id, so a
// retry — by the caller with the same Idempotency-Key, or by Sync after a crash — never charges twice.
package reserve

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trade1/compute-control/internal/ledger"
	"github.com/trade1/compute-control/internal/pool"
)

// Term is one reservation term: its length in hours and its discount against on-demand.
type Term struct {
	Hours       int
	DiscountPct int
}

// Terms are the terms on sale (F14: 17 / 27 / 33 %). A month is 730 hours, as clouds bill it.
var Terms = map[string]Term{
	"1mo":  {Hours: 730, DiscountPct: 17},
	"6mo":  {Hours: 4380, DiscountPct: 27},
	"12mo": {Hours: 8760, DiscountPct: 33},
}

// MaxGPUs is the largest single reservation.
const MaxGPUs = 256

// States a reservation moves through.
const (
	PendingPayment = "pending_payment"
	Active         = "active"
	Expired        = "expired"
	Failed         = "failed"
)

// Errors surfaced to the API.
var (
	ErrBadRequest     = errors.New("reserve: invalid reservation")
	ErrNoCapacity     = errors.New("reserve: not enough free capacity to set aside")
	ErrInsufficient   = errors.New("reserve: insufficient GPU credits")
	ErrPaymentPending = errors.New("reserve: payment not confirmed yet")
	ErrIdemConflict   = errors.New("reserve: idempotency key reused with a different body")
	ErrNotFound       = errors.New("reserve: no such reservation")
)

// Quote is the price of a reservation before it is bought. Amounts are GPU credits (GPU-hours),
// fixed-point.
type Quote struct {
	GPUType     string `json:"gpu_type"`
	GPUs        int    `json:"gpus"`
	Term        string `json:"term"`
	Hours       int    `json:"hours"`
	DiscountPct int    `json:"discount_pct"`
	GPUHours    string `json:"gpu_hours"`
	OnDemand    string `json:"on_demand_credits"`
	Price       string `json:"price_credits"`
	Saving      string `json:"saving_credits"`
}

// NewQuote validates a request and prices it exactly: gpu_hours = gpus × term hours, and the price is
// gpu_hours less the term discount (one GPU credit buys one on-demand GPU-hour).
func NewQuote(gpuType string, gpus int, term string) (Quote, error) {
	t, ok := Terms[term]
	switch {
	case gpuType != "gpu_h100" && gpuType != "gpu_h200":
		return Quote{}, fmt.Errorf("%w: gpu_type must be gpu_h100 or gpu_h200", ErrBadRequest)
	case gpus < 1 || gpus > MaxGPUs:
		return Quote{}, fmt.Errorf("%w: gpus must be 1-%d", ErrBadRequest, MaxGPUs)
	case !ok:
		return Quote{}, fmt.Errorf("%w: term must be 1mo, 6mo or 12mo", ErrBadRequest)
	}
	hours := big.NewRat(int64(gpus)*int64(t.Hours), 1)
	price := new(big.Rat).Mul(hours, big.NewRat(int64(100-t.DiscountPct), 100))
	return Quote{
		GPUType: gpuType, GPUs: gpus, Term: term, Hours: t.Hours, DiscountPct: t.DiscountPct,
		GPUHours: hours.FloatString(6), OnDemand: hours.FloatString(6), Price: price.FloatString(6),
		Saving: new(big.Rat).Sub(hours, price).FloatString(6),
	}, nil
}

// Reservation is one reservation.
type Reservation struct {
	ID           string     `json:"id"`
	TenantID     string     `json:"-"`
	SubAccountID *string    `json:"sub_account_id"`
	IsPaper      bool       `json:"is_paper"`
	GPUType      string     `json:"gpu_type"`
	GPUs         int        `json:"gpus"`
	Term         string     `json:"term"`
	DiscountPct  int        `json:"discount_pct"`
	GPUHours     string     `json:"gpu_hours"`
	Price        string     `json:"price_credits"`
	State        string     `json:"state"`
	StartsAt     *time.Time `json:"starts_at"`
	EndsAt       *time.Time `json:"ends_at"`
	LedgerTxID   *string    `json:"ledger_tx_id"`
	Failure      *string    `json:"failure"`
	CreatedAt    time.Time  `json:"created_at"`
}

// key is the pool hold this reservation contributes to.
func (r Reservation) key() pool.HoldKey {
	return pool.HoldKey{Tenant: r.TenantID, IsPaper: r.IsPaper, Tier: r.GPUType}
}

// Debiter books the payment (the ledger client).
type Debiter interface {
	Debit(ctx context.Context, d ledger.Debit) (string, error)
}

// Service sells and tracks reservations. Buy and Sync are serialised so a hold is never rebuilt from
// the records while a purchase is between taking its hold and recording it.
type Service struct {
	db     *pgxpool.Pool
	pool   *pool.Pool
	ledger Debiter
	mu     sync.Mutex
	Now    func() time.Time // nil = time.Now
}

// Open connects to the database.
func Open(ctx context.Context, dsn string, p *pool.Pool, l Debiter) (*Service, error) {
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("reserve: connect: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("reserve: ping: %w", err)
	}
	return &Service{db: db, pool: p, ledger: l}, nil
}

// Close releases the database pool.
func (s *Service) Close() { s.db.Close() }

// now is the service clock.
func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// columns is the select list scan reads.
const columns = `id::text, tenant_id, sub_account_id, is_paper, gpu_type, gpus, term, discount_pct, gpu_hours::text,
	price::text, state, starts_at, ends_at, ledger_tx_id, failure, created_at`

// scan reads one row in `columns` order.
func scan(row pgx.Row) (Reservation, error) {
	var r Reservation
	err := row.Scan(&r.ID, &r.TenantID, &r.SubAccountID, &r.IsPaper, &r.GPUType, &r.GPUs, &r.Term, &r.DiscountPct,
		&r.GPUHours, &r.Price, &r.State, &r.StartsAt, &r.EndsAt, &r.LedgerTxID, &r.Failure, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Reservation{}, ErrNotFound
	}
	return r, err
}

// requestHash fingerprints a purchase for idempotency.
func requestHash(q Quote, sub string) string {
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%d|%s|%s", q.GPUType, q.GPUs, q.Term, sub))
	return hex.EncodeToString(sum[:])
}

// Buy sells q to a tenant: it sets the GPUs aside, records the reservation, and debits the price.
// Idempotent on (tenant, key): a replay returns the original, retrying its payment if that had not
// gone through. ErrNoCapacity means nothing was recorded; ErrInsufficient means it was recorded as
// failed and the GPUs were given back; ErrPaymentPending means the ledger did not answer and the
// GPUs stay set aside until a retry or Sync settles it.
func (s *Service) Buy(ctx context.Context, tenant, sub string, isPaper bool, key string, q Quote) (Reservation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	hash := requestHash(q, sub)
	var prevHash string
	prev, err := scanHash(s.db.QueryRow(ctx, `SELECT `+columns+`, request_hash FROM compute_reservations
		WHERE tenant_id=$1 AND idempotency_key=$2`, tenant, key), &prevHash)
	switch {
	case err == nil:
		if prevHash != hash {
			return Reservation{}, ErrIdemConflict
		}
		if prev.State == PendingPayment {
			return s.pay(ctx, prev)
		}
		return prev, nil
	case !errors.Is(err, ErrNotFound):
		return Reservation{}, err
	}

	r := Reservation{ID: uuid.NewString(), TenantID: tenant, IsPaper: isPaper, GPUType: q.GPUType, GPUs: q.GPUs, Term: q.Term}
	if sub != "" {
		r.SubAccountID = &sub
	}
	if !s.pool.Hold(r.key(), q.GPUs) {
		return Reservation{}, ErrNoCapacity
	}
	r, err = s.insert(ctx, r, q, key, hash)
	if err != nil {
		s.pool.ReleaseHold(pool.HoldKey{Tenant: tenant, IsPaper: isPaper, Tier: q.GPUType}, q.GPUs)
		return Reservation{}, err
	}
	return s.pay(ctx, r)
}

// scanHash reads a reservation plus its request hash.
func scanHash(row pgx.Row, hash *string) (Reservation, error) {
	var r Reservation
	err := row.Scan(&r.ID, &r.TenantID, &r.SubAccountID, &r.IsPaper, &r.GPUType, &r.GPUs, &r.Term, &r.DiscountPct,
		&r.GPUHours, &r.Price, &r.State, &r.StartsAt, &r.EndsAt, &r.LedgerTxID, &r.Failure, &r.CreatedAt, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Reservation{}, ErrNotFound
	}
	return r, err
}

// insert records a new reservation awaiting payment, with its audit row.
func (s *Service) insert(ctx context.Context, r Reservation, q Quote, key, hash string) (Reservation, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Reservation{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	out, err := scan(tx.QueryRow(ctx, `INSERT INTO compute_reservations (id, tenant_id, sub_account_id, is_paper, gpu_type,
		gpus, term, discount_pct, gpu_hours, price, state, idempotency_key, request_hash)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::numeric,$10::numeric,'pending_payment',$11,$12) RETURNING `+columns,
		r.ID, r.TenantID, r.SubAccountID, r.IsPaper, q.GPUType, q.GPUs, q.Term, q.DiscountPct, q.GPUHours, q.Price, key, hash))
	if err != nil {
		return Reservation{}, fmt.Errorf("reserve: insert: %w", err)
	}
	if err := event(ctx, tx, out.ID, "requested", r.TenantID, ""); err != nil {
		return Reservation{}, err
	}
	return out, tx.Commit(ctx)
}

// execer is a pool or a transaction.
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// event appends one audit row.
func event(ctx context.Context, q execer, id, kind, actor, detail string) error {
	if _, err := q.Exec(ctx, `INSERT INTO compute_reservation_events (reservation_id, kind, actor, detail) VALUES ($1,$2,$3,$4)`,
		id, kind, actor, detail); err != nil {
		return fmt.Errorf("reserve: audit %s: %w", kind, err)
	}
	return nil
}

// pay debits a pending reservation's price and settles its state. Caller holds s.mu.
func (s *Service) pay(ctx context.Context, r Reservation) (Reservation, error) {
	txID, err := s.ledger.Debit(ctx, ledger.Debit{
		TenantID: r.TenantID, SubAccountID: r.SubAccountID, CreditType: r.GPUType, Amount: r.Price,
		UsageEventID: "reservation:" + r.ID, IsPaper: r.IsPaper,
	})
	switch {
	case errors.Is(err, ledger.ErrInsufficientCredit):
		out, uerr := s.settle(ctx, r, Failed, nil, "insufficient "+r.GPUType+" credits", "failed")
		if uerr != nil {
			return Reservation{}, uerr
		}
		s.pool.ReleaseHold(r.key(), r.GPUs)
		return out, ErrInsufficient
	case err != nil:
		return r, fmt.Errorf("%w: %w", ErrPaymentPending, err)
	}
	return s.settle(ctx, r, Active, &txID, "", "paid")
}

// settle moves a pending reservation to active (paid: its term starts now) or failed.
func (s *Service) settle(ctx context.Context, r Reservation, state string, txID *string, failure, kind string) (Reservation, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Reservation{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	now := s.now()
	var fail *string
	if failure != "" {
		fail = &failure
	}
	out, err := scan(tx.QueryRow(ctx, `UPDATE compute_reservations SET state=$2, ledger_tx_id=$3, failure=$4,
		starts_at = CASE WHEN $2 = 'active' THEN $5::timestamptz END,
		ends_at   = CASE WHEN $2 = 'active' THEN $5::timestamptz + make_interval(hours => $6::int) END,
		updated_at = now()
		WHERE id=$1 AND state='pending_payment' RETURNING `+columns, r.ID, state, txID, fail, now, Terms[r.Term].Hours))
	if err != nil {
		return Reservation{}, fmt.Errorf("reserve: settle: %w", err)
	}
	if err := event(ctx, tx, r.ID, kind, "system", failure); err != nil {
		return Reservation{}, err
	}
	return out, tx.Commit(ctx)
}

// List returns a tenant's reservations, newest first.
func (s *Service) List(ctx context.Context, tenant string) ([]Reservation, error) {
	rows, err := s.db.Query(ctx, `SELECT `+columns+` FROM compute_reservations WHERE tenant_id=$1 ORDER BY created_at DESC, id`, tenant)
	if err != nil {
		return nil, fmt.Errorf("reserve: list: %w", err)
	}
	defer rows.Close()
	out := []Reservation{}
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Get returns one of a tenant's reservations; another tenant's is ErrNotFound.
func (s *Service) Get(ctx context.Context, tenant, id string) (Reservation, error) {
	if _, err := uuid.Parse(id); err != nil {
		return Reservation{}, ErrNotFound
	}
	return scan(s.db.QueryRow(ctx, `SELECT `+columns+` FROM compute_reservations WHERE id=$1 AND tenant_id=$2`, id, tenant))
}

// Sync settles the reservations with the pool once: it retries payments left pending (a crash or a
// ledger outage), expires reservations whose term is over, and sets every hold to exactly what the
// live reservations add up to.
func (s *Service) Sync(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pending, err := s.byState(ctx, PendingPayment)
	if err != nil {
		return err
	}
	for _, r := range pending {
		if _, err := s.pay(ctx, r); err != nil && !errors.Is(err, ErrInsufficient) && !errors.Is(err, ErrPaymentPending) {
			return err
		}
	}
	rows, err := s.db.Query(ctx, `UPDATE compute_reservations SET state='expired', updated_at=now()
		WHERE state='active' AND ends_at <= $1 RETURNING id::text`, s.now())
	if err != nil {
		return fmt.Errorf("reserve: expire: %w", err)
	}
	var expired []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		expired = append(expired, id)
	}
	rows.Close()
	for _, id := range expired {
		if err := event(ctx, s.db, id, "expired", "system", ""); err != nil {
			return err
		}
	}
	return s.rebuildHolds(ctx)
}

// byState lists every reservation in a state.
func (s *Service) byState(ctx context.Context, state string) ([]Reservation, error) {
	rows, err := s.db.Query(ctx, `SELECT `+columns+` FROM compute_reservations WHERE state=$1 ORDER BY created_at`, state)
	if err != nil {
		return nil, fmt.Errorf("reserve: %s: %w", state, err)
	}
	defer rows.Close()
	var out []Reservation
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// rebuildHolds sets the pool's holds to the live reservations (pending payment or active).
func (s *Service) rebuildHolds(ctx context.Context) error {
	rows, err := s.db.Query(ctx, `SELECT tenant_id, is_paper, gpu_type, sum(gpus)::int FROM compute_reservations
		WHERE state IN ('pending_payment', 'active') GROUP BY tenant_id, is_paper, gpu_type`)
	if err != nil {
		return fmt.Errorf("reserve: holds: %w", err)
	}
	defer rows.Close()
	want := map[pool.HoldKey]int{}
	for rows.Next() {
		var k pool.HoldKey
		var n int
		if err := rows.Scan(&k.Tenant, &k.IsPaper, &k.Tier, &n); err != nil {
			return err
		}
		want[k] = n
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for k := range s.pool.Holds() {
		if _, ok := want[k]; !ok {
			s.pool.SetHold(k, 0)
		}
	}
	for k, n := range want {
		s.pool.SetHold(k, n)
	}
	return nil
}

// Run syncs on an interval until ctx ends.
func (s *Service) Run(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		if err := s.Sync(ctx); err != nil && ctx.Err() == nil {
			slog.Warn("reservation sync failed", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
