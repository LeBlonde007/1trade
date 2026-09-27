package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/trade1/credit-ledger/internal/domain"
)

// Settlement errors surfaced to the API.
var (
	// ErrSettleConflict: the trade_id was already settled with a different body (409).
	ErrSettleConflict = errors.New("trade already settled with a different body")
	// ErrNotTradeable: the credit type is unknown or not tradeable (422).
	ErrNotTradeable = errors.New("credit type is not tradeable")
)

// SettleResult is the outcome of one settlement: the recomputed notional and every row written.
type SettleResult struct {
	TradeID  string
	Notional domain.Money
	Credit   []domain.Transaction
	Cash     []domain.CashTx
}

// SettleTrade settles one trade atomically (credit.yaml v1.1 settle-trade). In ONE DB transaction:
//
//  1. take a per-trade advisory lock, so two concurrent calls for the same trade_id serialise;
//  2. if the trade is already settled, return the stored result (replayed=true) — or
//     ErrSettleConflict if this body differs from the one that settled it;
//  3. check the credit type is tradeable;
//  4. apply every credit leg, then every cash leg, in PlanSettlement's global order;
//  5. record the settlement row (the idempotency record + the original result).
//
// Any failure — domain.ErrInsufficientCredit (seller short), domain.ErrInsufficientCash (buyer short),
// or anything else — rolls the whole transaction back: nothing is written.
func (s *Store) SettleTrade(ctx context.Context, req domain.SettleRequest) (SettleResult, bool, error) {
	plan, err := domain.PlanSettlement(req)
	if err != nil {
		return SettleResult{}, false, err
	}
	dbtx, err := s.pool.Begin(ctx)
	if err != nil {
		return SettleResult{}, false, err
	}
	defer dbtx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if _, err := dbtx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, req.TradeID); err != nil {
		return SettleResult{}, false, fmt.Errorf("lock trade: %w", err)
	}
	var prevHash, prevJSON string
	err = dbtx.QueryRow(ctx, `SELECT request_hash, result_json FROM trade_settlements WHERE trade_id=$1`, req.TradeID).
		Scan(&prevHash, &prevJSON)
	switch {
	case err == nil:
		if prevHash != req.Hash() {
			return SettleResult{}, false, ErrSettleConflict
		}
		var prev SettleResult
		if err := json.Unmarshal([]byte(prevJSON), &prev); err != nil {
			return SettleResult{}, false, fmt.Errorf("decode stored settlement %s: %w", req.TradeID, err)
		}
		return prev, true, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return SettleResult{}, false, fmt.Errorf("read settlement: %w", err)
	}

	var tradeable bool
	if err := dbtx.QueryRow(ctx, `SELECT tradeable FROM credit_types WHERE code=$1 AND active`, string(req.CreditType)).
		Scan(&tradeable); err != nil || !tradeable {
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return SettleResult{}, false, fmt.Errorf("read credit type: %w", err)
		}
		return SettleResult{}, false, ErrNotTradeable
	}

	res := SettleResult{TradeID: req.TradeID, Notional: plan.Notional}
	for _, l := range plan.Credit {
		tx, err := applyLeg(ctx, dbtx, Movement{
			TenantID: l.TenantID, SubAccountID: l.SubAccountID, CreditType: l.CreditType, Operation: domain.OpTrade,
			Amount: l.Amount, ReferenceID: req.TradeID, IdempotencyKey: l.Key, IsPaper: true,
			ConsumeOrderID: l.PayingOrderID,
		})
		if err != nil {
			return SettleResult{}, false, err
		}
		res.Credit = append(res.Credit, tx)
	}
	for _, l := range plan.Cash {
		tx, _, err := applyCashLeg(ctx, dbtx, CashMovement{
			TenantID: l.TenantID, SubAccountID: l.SubAccountID, Currency: l.Currency, Operation: l.Operation,
			Amount: l.Amount, ReferenceID: req.TradeID, IdempotencyKey: l.Key,
			ConsumeOrderID: l.PayingOrderID,
		})
		if err != nil {
			return SettleResult{}, false, err
		}
		res.Cash = append(res.Cash, tx)
	}

	raw, err := json.Marshal(res)
	if err != nil {
		return SettleResult{}, false, fmt.Errorf("encode settlement: %w", err)
	}
	if _, err := dbtx.Exec(ctx,
		`INSERT INTO trade_settlements (trade_id, request_hash, engine_chain_hash, notional, result_json, is_paper)
		 VALUES ($1,$2,$3,$4::numeric,$5,true)`,
		req.TradeID, req.Hash(), req.EngineChainHash, plan.Notional.String(), string(raw)); err != nil {
		return SettleResult{}, false, fmt.Errorf("record settlement: %w", err)
	}
	if err := dbtx.Commit(ctx); err != nil {
		return SettleResult{}, false, fmt.Errorf("commit: %w", err)
	}
	return res, false, nil
}
