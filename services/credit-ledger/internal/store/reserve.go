package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/trade1/credit-ledger/internal/domain"
)

// Reservation errors surfaced to the API.
var (
	// ErrReserveConflict: the order_id was already reserved with a different body (409).
	ErrReserveConflict = errors.New("order already reserved with a different body")
	// ErrNoReservation: release named an order with no reservation (404).
	ErrNoReservation = errors.New("no reservation for this order")
	// ErrBadAsset: the asset is not a known credit type / currency for its kind (422).
	ErrBadAsset = errors.New("unknown asset for this asset kind")
)

// Reservation is one order's hold on a balance (credit.yaml v1.2 Reservation).
type Reservation struct {
	OrderID      string
	TenantID     string
	SubAccountID string
	AssetKind    string // credit | cash
	Asset        string // credit_type or currency
	Amount       domain.Money
	Remaining    domain.Money
	State        string // open | released
}

// hash is the idempotency fingerprint of a reserve request (order_id is the key; this detects a
// different body under the same key).
func (r Reservation) hash() string {
	sum := sha256.Sum256([]byte(r.TenantID + "|" + r.SubAccountID + "|" + r.AssetKind + "|" + r.Asset + "|" + r.Amount.String()))
	return hex.EncodeToString(sum[:])
}

// balanceTable returns the balance table and its asset column for an asset kind. The values are
// constants chosen here, never caller input, so they are safe to place in SQL text.
func balanceTable(kind string) (table, col string, err error) {
	switch kind {
	case "credit":
		return "credit_balances", "credit_type", nil
	case "cash":
		return "cash_balances", "currency", nil
	}
	return "", "", ErrBadAsset
}

// lockBalance ensures the balance row exists and locks it FOR UPDATE, returning balance and locked.
// Lock order everywhere is balance row first, then reservation row, so reserve / release / settle
// can never deadlock on each other.
func lockBalance(ctx context.Context, dbtx pgx.Tx, kind, tenantID, subAccountID, asset string) (bal, locked domain.Money, err error) {
	table, col, err := balanceTable(kind)
	if err != nil {
		return
	}
	sub := nullable(subAccountID)
	if _, err = dbtx.Exec(ctx,
		`INSERT INTO `+table+` (balance_id, tenant_id, sub_account_id, `+col+`, balance, is_paper, last_chain_hash)
		 VALUES ($1,$2,$3,$4,0,true,'') ON CONFLICT DO NOTHING`,
		uuid.NewString(), tenantID, sub, asset); err != nil {
		if isForeignKeyViolation(err) || isCheckViolation(err) {
			err = ErrBadAsset
			return
		}
		err = fmt.Errorf("ensure %s: %w", table, err)
		return
	}
	var balStr, lockStr string
	if err = dbtx.QueryRow(ctx,
		`SELECT balance::text, locked_amount::text FROM `+table+`
		 WHERE tenant_id=$1 AND sub_account_id IS NOT DISTINCT FROM $2 AND `+col+`=$3 AND is_paper FOR UPDATE`,
		tenantID, sub, asset).Scan(&balStr, &lockStr); err != nil {
		err = fmt.Errorf("lock %s: %w", table, err)
		return
	}
	bal, _ = domain.ParseMoney(balStr)
	locked, _ = domain.ParseMoney(lockStr)
	return bal, locked, nil
}

// setLocked writes a balance row's locked_amount (the row must already be locked by lockBalance).
func setLocked(ctx context.Context, dbtx pgx.Tx, kind, tenantID, subAccountID, asset string, locked domain.Money) error {
	table, col, err := balanceTable(kind)
	if err != nil {
		return err
	}
	if _, err := dbtx.Exec(ctx,
		`UPDATE `+table+` SET locked_amount=$1::numeric, updated_at=now()
		 WHERE tenant_id=$2 AND sub_account_id IS NOT DISTINCT FROM $3 AND `+col+`=$4 AND is_paper`,
		locked.String(), tenantID, nullable(subAccountID), asset); err != nil {
		return fmt.Errorf("update %s locked: %w", table, err)
	}
	return nil
}

// reservationEvent appends one row to the reservation audit log.
func reservationEvent(ctx context.Context, dbtx pgx.Tx, orderID, kind string, amount, remaining domain.Money, ref string) error {
	if _, err := dbtx.Exec(ctx,
		`INSERT INTO reservation_events (event_id, order_id, kind, amount, remaining, reference)
		 VALUES ($1,$2,$3,$4::numeric,$5::numeric,$6)`,
		uuid.NewString(), orderID, kind, amount.String(), remaining.String(), nullable(ref)); err != nil {
		return fmt.Errorf("reservation event: %w", err)
	}
	return nil
}

// Reserve holds amount of an asset for an order (credit.yaml v1.2 /reserve). It must fit within
// available (balance − locked); then locked rises by amount. Idempotent on order_id: a replay returns
// the original (replayed=true); a different body is ErrReserveConflict.
func (s *Store) Reserve(ctx context.Context, r Reservation) (Reservation, bool, error) {
	dbtx, err := s.pool.Begin(ctx)
	if err != nil {
		return Reservation{}, false, err
	}
	defer dbtx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	bal, locked, err := lockBalance(ctx, dbtx, r.AssetKind, r.TenantID, r.SubAccountID, r.Asset)
	if err != nil {
		return Reservation{}, false, err
	}
	if prev, ok, err := getReservation(ctx, dbtx, r.OrderID, true); err != nil {
		return Reservation{}, false, err
	} else if ok {
		if prev.hash() != r.hash() {
			return Reservation{}, false, ErrReserveConflict
		}
		return prev, true, nil
	}
	if bal.Sub(locked).Cmp(r.Amount) < 0 {
		if r.AssetKind == "cash" {
			return Reservation{}, false, domain.ErrInsufficientCash
		}
		return Reservation{}, false, domain.ErrInsufficientCredit
	}
	r.Remaining, r.State = r.Amount, "open"
	if _, err := dbtx.Exec(ctx,
		`INSERT INTO reservations (order_id, tenant_id, sub_account_id, asset_kind, asset, amount, remaining, state, is_paper, request_hash)
		 VALUES ($1,$2,$3,$4,$5,$6::numeric,$6::numeric,'open',true,$7)`,
		r.OrderID, r.TenantID, nullable(r.SubAccountID), r.AssetKind, r.Asset, r.Amount.String(), r.hash()); err != nil {
		if IsUniqueViolation(err) {
			return Reservation{}, false, ErrReserveConflict // same order_id reserved concurrently on another asset
		}
		return Reservation{}, false, fmt.Errorf("insert reservation: %w", err)
	}
	if err := setLocked(ctx, dbtx, r.AssetKind, r.TenantID, r.SubAccountID, r.Asset, locked.Add(r.Amount)); err != nil {
		return Reservation{}, false, err
	}
	if err := reservationEvent(ctx, dbtx, r.OrderID, "reserve", r.Amount, r.Amount, ""); err != nil {
		return Reservation{}, false, err
	}
	if err := dbtx.Commit(ctx); err != nil {
		return Reservation{}, false, fmt.Errorf("commit: %w", err)
	}
	return r, false, nil
}

// Release unlocks whatever an order's reservation still holds (credit.yaml v1.2 /release).
// Idempotent: a released reservation is returned unchanged.
func (s *Store) Release(ctx context.Context, orderID string) (Reservation, error) {
	dbtx, err := s.pool.Begin(ctx)
	if err != nil {
		return Reservation{}, err
	}
	defer dbtx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	// Read without a lock to learn which balance to lock first (lock order: balance, then reservation).
	peek, ok, err := getReservation(ctx, dbtx, orderID, false)
	if err != nil {
		return Reservation{}, err
	}
	if !ok {
		return Reservation{}, ErrNoReservation
	}
	_, locked, err := lockBalance(ctx, dbtx, peek.AssetKind, peek.TenantID, peek.SubAccountID, peek.Asset)
	if err != nil {
		return Reservation{}, err
	}
	r, _, err := getReservation(ctx, dbtx, orderID, true)
	if err != nil {
		return Reservation{}, err
	}
	if r.State == "released" {
		return r, nil
	}
	freed := r.Remaining
	if _, err := dbtx.Exec(ctx,
		`UPDATE reservations SET remaining=0, state='released', updated_at=now() WHERE order_id=$1`, orderID); err != nil {
		return Reservation{}, fmt.Errorf("release reservation: %w", err)
	}
	if err := setLocked(ctx, dbtx, r.AssetKind, r.TenantID, r.SubAccountID, r.Asset, locked.Sub(freed)); err != nil {
		return Reservation{}, err
	}
	if err := reservationEvent(ctx, dbtx, orderID, "release", freed, domain.Zero(), ""); err != nil {
		return Reservation{}, err
	}
	if err := dbtx.Commit(ctx); err != nil {
		return Reservation{}, fmt.Errorf("commit: %w", err)
	}
	r.Remaining, r.State = domain.Zero(), "released"
	return r, nil
}

// consumeReservation is called by a settlement debit leg with its balance row already locked. If the
// paying order has an open reservation on this balance, the debit is taken from it: the reservation's
// remaining and the balance's locked both drop by the debit (the caller writes the new locked).
// A debit larger than the remaining reservation is insufficient. With no order or no open reservation,
// locked is returned unchanged and the caller's available check applies.
func consumeReservation(ctx context.Context, dbtx pgx.Tx, orderID, kind, tenantID, asset string,
	amount domain.Money, ref string, locked domain.Money, insufficient error) (domain.Money, error) {
	if orderID == "" || !amount.IsNegative() {
		return locked, nil
	}
	r, ok, err := getReservation(ctx, dbtx, orderID, true)
	if err != nil {
		return locked, err
	}
	if !ok || r.State != "open" || r.AssetKind != kind || r.TenantID != tenantID || r.Asset != asset {
		return locked, nil
	}
	spend := domain.Zero().Sub(amount)
	if spend.Cmp(r.Remaining) > 0 {
		return locked, insufficient
	}
	left := r.Remaining.Sub(spend)
	if _, err := dbtx.Exec(ctx,
		`UPDATE reservations SET remaining=$1::numeric, updated_at=now() WHERE order_id=$2`, left.String(), orderID); err != nil {
		return locked, fmt.Errorf("consume reservation: %w", err)
	}
	if err := reservationEvent(ctx, dbtx, orderID, "consume", spend, left, ref); err != nil {
		return locked, err
	}
	return locked.Sub(spend), nil
}

// getReservation reads one reservation, optionally FOR UPDATE.
func getReservation(ctx context.Context, dbtx pgx.Tx, orderID string, forUpdate bool) (Reservation, bool, error) {
	q := `SELECT order_id::text, tenant_id::text, coalesce(sub_account_id::text,''), asset_kind, asset,
	             amount::text, remaining::text, state FROM reservations WHERE order_id=$1`
	if forUpdate {
		q += ` FOR UPDATE`
	}
	var r Reservation
	var amt, rem string
	err := dbtx.QueryRow(ctx, q, orderID).Scan(&r.OrderID, &r.TenantID, &r.SubAccountID, &r.AssetKind, &r.Asset, &amt, &rem, &r.State)
	if errors.Is(err, pgx.ErrNoRows) {
		return Reservation{}, false, nil
	}
	if err != nil {
		return Reservation{}, false, fmt.Errorf("read reservation: %w", err)
	}
	r.Amount, _ = domain.ParseMoney(amt)
	r.Remaining, _ = domain.ParseMoney(rem)
	return r, true, nil
}
