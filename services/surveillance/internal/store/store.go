// Package store persists surveillance alerts in Postgres (migrations/0001_alerts.sql).
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trade1/surveillance/internal/detect"
)

// Store wraps a pgx pool.
type Store struct{ pool *pgxpool.Pool }

// New connects and pings.
func New(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Ping checks the database is reachable.
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// SaveAlert stores an alert once; it reports false when the alert id was already stored.
func (s *Store) SaveAlert(ctx context.Context, a detect.Alert) (bool, error) {
	ev, err := json.Marshal(a.Evidence)
	if err != nil {
		return false, fmt.Errorf("encode evidence: %w", err)
	}
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO surveillance_alerts (alert_id, rule, severity, action, tenant_id, counterparty_tenant_id, product_id,
		   related_product_id, is_paper, window_start, window_end, detected_at, evidence, trade_ids, order_ids)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		 ON CONFLICT (alert_id) DO NOTHING`,
		a.AlertID, a.Rule, a.Severity, a.Action, a.TenantID, a.CounterpartyTenantID, a.ProductID, a.RelatedProductID,
		a.IsPaper, a.WindowStart, a.WindowEnd, a.DetectedAt, ev, a.TradeIDs, a.OrderIDs)
	if err != nil {
		return false, fmt.Errorf("insert alert: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// Filter narrows a listing. Empty fields do not filter; IsPaper nil means both ledgers.
type Filter struct {
	Rule, TenantID, ProductID string
	IsPaper                   *bool
	Limit                     int
}

// ListAlerts returns alerts newest first. The WHERE clause is built from fixed column names with
// numbered placeholders only — never from caller text.
func (s *Store) ListAlerts(ctx context.Context, f Filter) ([]detect.Alert, error) {
	var (
		where []string
		args  []any
	)
	add := func(col string, v any) {
		args = append(args, v)
		where = append(where, col+" = $"+strconv.Itoa(len(args)))
	}
	if f.Rule != "" {
		add("rule", f.Rule)
	}
	if f.TenantID != "" {
		add("tenant_id", f.TenantID)
	}
	if f.ProductID != "" {
		add("product_id", f.ProductID)
	}
	if f.IsPaper != nil {
		add("is_paper", *f.IsPaper)
	}
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 100
	}
	q := `SELECT alert_id, rule, severity, action, tenant_id, counterparty_tenant_id, product_id, related_product_id,
	             is_paper, window_start, window_end, detected_at, evidence, trade_ids, order_ids FROM surveillance_alerts`
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	args = append(args, f.Limit)
	q += " ORDER BY detected_at DESC, alert_id LIMIT $" + strconv.Itoa(len(args))
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()
	out := []detect.Alert{}
	for rows.Next() {
		var a detect.Alert
		var ev []byte
		if err := rows.Scan(&a.AlertID, &a.Rule, &a.Severity, &a.Action, &a.TenantID, &a.CounterpartyTenantID, &a.ProductID,
			&a.RelatedProductID, &a.IsPaper, &a.WindowStart, &a.WindowEnd, &a.DetectedAt, &ev, &a.TradeIDs, &a.OrderIDs); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		_ = json.Unmarshal(ev, &a.Evidence)
		out = append(out, a)
	}
	return out, rows.Err()
}
