// Package supply is the partner-capacity registry (supply.yaml v1.0, F16/F17): partner datacenters
// register GPU sources, heartbeat their health, and are drained by suspension or retirement. The
// registry is durable (Postgres); Syncer turns it into the scheduler's view of the pool. It also
// records every metered usage interval per source — the basis for partner payouts (F18).
package supply

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// States a source moves through.
const (
	Pending   = "pending"
	Active    = "active"
	Suspended = "suspended"
	Retired   = "retired"
)

// Errors surfaced to the API.
var (
	ErrNotFound     = errors.New("supply: no such source")
	ErrIdemConflict = errors.New("supply: idempotency key reused with a different body")
	ErrState        = errors.New("supply: not allowed in the source's current state")
)

// Source is one registered supply source.
type Source struct {
	ID              string
	PartnerTenantID string
	Name            string
	GPUType         string
	GPUCount        int
	Region          string
	SLATier         string
	State           string
	GPUsHealthy     *int
	UtilizationPct  *int
	LastHeartbeatAt *time.Time
	CreatedAt       time.Time
}

// Registration is what a partner submits.
type Registration struct {
	Name     string
	GPUType  string
	GPUCount int
	Region   string
	SLATier  string
}

// hash fingerprints a registration for idempotency.
func (r Registration) hash() string {
	sum := sha256.Sum256(fmt.Appendf(nil, "%s|%s|%d|%s|%s", r.Name, r.GPUType, r.GPUCount, r.Region, r.SLATier))
	return hex.EncodeToString(sum[:])
}

// Store is the Postgres-backed registry.
type Store struct {
	pool *pgxpool.Pool
}

// Open connects and verifies the connection.
func Open(ctx context.Context, dsn string) (*Store, error) {
	p, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("supply: connect: %w", err)
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, fmt.Errorf("supply: ping: %w", err)
	}
	return &Store{pool: p}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// columns is the select list scanSource reads.
const columns = `id::text, partner_tenant_id::text, name, gpu_type, gpu_count, region, sla_tier, state,
	gpus_healthy, utilization_pct, last_heartbeat_at, created_at`

// scanSource reads one row in `columns` order.
func scanSource(row pgx.Row) (Source, error) {
	var src Source
	err := row.Scan(&src.ID, &src.PartnerTenantID, &src.Name, &src.GPUType, &src.GPUCount, &src.Region, &src.SLATier,
		&src.State, &src.GPUsHealthy, &src.UtilizationPct, &src.LastHeartbeatAt, &src.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Source{}, ErrNotFound
	}
	return src, err
}

// Register records a new pending source for a partner. Idempotent on (partner, key): a replay
// returns the original (replayed=true); a different body under the same key is ErrIdemConflict.
func (s *Store) Register(ctx context.Context, partner, key string, r Registration) (src Source, replayed bool, err error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Source{}, false, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	var prevHash string
	prev, err := scanSourceHash(tx.QueryRow(ctx, `SELECT `+columns+`, request_hash FROM supply_sources
		WHERE partner_tenant_id=$1 AND idempotency_key=$2`, partner, key), &prevHash)
	switch {
	case err == nil:
		if prevHash != r.hash() {
			return Source{}, false, ErrIdemConflict
		}
		return prev, true, nil
	case !errors.Is(err, ErrNotFound):
		return Source{}, false, err
	}
	id := uuid.NewString()
	src, err = scanSource(tx.QueryRow(ctx, `INSERT INTO supply_sources
		(id, partner_tenant_id, name, gpu_type, gpu_count, region, sla_tier, state, idempotency_key, request_hash)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'pending',$8,$9) RETURNING `+columns,
		id, partner, r.Name, r.GPUType, r.GPUCount, r.Region, r.SLATier, key, r.hash()))
	if err != nil {
		return Source{}, false, fmt.Errorf("supply: register: %w", err)
	}
	if err := event(ctx, tx, id, "registered", partner); err != nil {
		return Source{}, false, err
	}
	return src, false, tx.Commit(ctx)
}

// scanSourceHash reads a source plus its request hash.
func scanSourceHash(row pgx.Row, hash *string) (Source, error) {
	var src Source
	err := row.Scan(&src.ID, &src.PartnerTenantID, &src.Name, &src.GPUType, &src.GPUCount, &src.Region, &src.SLATier,
		&src.State, &src.GPUsHealthy, &src.UtilizationPct, &src.LastHeartbeatAt, &src.CreatedAt, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Source{}, ErrNotFound
	}
	return src, err
}

// event appends one audit row.
func event(ctx context.Context, tx pgx.Tx, id, kind, actor string) error {
	if _, err := tx.Exec(ctx, `INSERT INTO supply_events (source_id, kind, actor) VALUES ($1,$2,$3)`, id, kind, actor); err != nil {
		return fmt.Errorf("supply: audit %s: %w", kind, err)
	}
	return nil
}

// Get returns a source. partner "" means any partner (operations); otherwise another partner's
// source is ErrNotFound, so ids leak nothing.
func (s *Store) Get(ctx context.Context, id, partner string) (Source, error) {
	if _, err := uuid.Parse(id); err != nil {
		return Source{}, ErrNotFound
	}
	src, err := scanSource(s.pool.QueryRow(ctx, `SELECT `+columns+` FROM supply_sources WHERE id=$1`, id))
	if err != nil {
		return Source{}, err
	}
	if partner != "" && src.PartnerTenantID != partner {
		return Source{}, ErrNotFound
	}
	return src, nil
}

// List returns a partner's sources, or every source when partner is "", oldest first.
func (s *Store) List(ctx context.Context, partner string) ([]Source, error) {
	q, args := `SELECT `+columns+` FROM supply_sources ORDER BY created_at, id`, []any{}
	if partner != "" {
		q, args = `SELECT `+columns+` FROM supply_sources WHERE partner_tenant_id=$1 ORDER BY created_at, id`, []any{partner}
	}
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("supply: list: %w", err)
	}
	defer rows.Close()
	var out []Source
	for rows.Next() {
		src, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, src)
	}
	return out, rows.Err()
}

// transitions is which states each action may leave from, and where it goes.
var transitions = map[string]struct {
	from []string
	to   string
	kind string
}{
	"activate": {[]string{Pending}, Active, "activated"},
	"suspend":  {[]string{Active, Suspended}, Suspended, "suspended"},
	"resume":   {[]string{Suspended, Active}, Active, "resumed"},
	"retire":   {[]string{Pending, Active, Suspended, Retired}, Retired, "retired"},
}

// Transition applies an action (activate | suspend | resume | retire) for actor. partner "" is
// operations. Repeating an action that already holds (suspend a suspended source) is a no-op
// success; an action from a state it does not leave is ErrState.
func (s *Store) Transition(ctx context.Context, id, partner, action, actor string) (Source, error) {
	t, ok := transitions[action]
	if !ok {
		return Source{}, fmt.Errorf("supply: unknown action %q", action)
	}
	if _, err := uuid.Parse(id); err != nil {
		return Source{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Source{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	src, err := scanSource(tx.QueryRow(ctx, `SELECT `+columns+` FROM supply_sources WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return Source{}, err
	}
	if partner != "" && src.PartnerTenantID != partner {
		return Source{}, ErrNotFound
	}
	allowed := false
	for _, f := range t.from {
		allowed = allowed || src.State == f
	}
	if !allowed {
		return Source{}, ErrState
	}
	if src.State == t.to {
		return src, nil // already there
	}
	src, err = scanSource(tx.QueryRow(ctx, `UPDATE supply_sources SET state=$2, updated_at=now() WHERE id=$1 RETURNING `+columns, id, t.to))
	if err != nil {
		return Source{}, fmt.Errorf("supply: %s: %w", action, err)
	}
	if err := event(ctx, tx, id, t.kind, actor); err != nil {
		return Source{}, err
	}
	return src, tx.Commit(ctx)
}

// Heartbeat records an agent report for a partner's source. A retired source refuses (ErrState).
func (s *Store) Heartbeat(ctx context.Context, id, partner string, healthy, util int, ecc int64) (Source, error) {
	src, err := s.Get(ctx, id, partner)
	if err != nil {
		return Source{}, err
	}
	if src.State == Retired {
		return Source{}, ErrState
	}
	return scanSource(s.pool.QueryRow(ctx, `UPDATE supply_sources SET gpus_healthy=$2, utilization_pct=$3,
		ecc_errors=ecc_errors+$4, last_heartbeat_at=now(), updated_at=now() WHERE id=$1 RETURNING `+columns,
		id, min(healthy, src.GPUCount), util, ecc))
}

// Usage is metered GPU time on a source for one tier.
type Usage struct {
	GPUType    string
	GPUSeconds string // fixed-point, as on compute.usage.v1
	Units      string
	Sessions   int
}

// RecordUsage stores one metered interval (idempotent on usageID).
func (s *Store) RecordUsage(ctx context.Context, usageID, sourceID, tenantID, gpuType, gpuSeconds, units string, isPaper bool) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO supply_usage (usage_id, source_id, tenant_id, gpu_type, gpu_seconds, units, is_paper)
		VALUES ($1,$2,$3,$4,$5::numeric,$6::numeric,$7) ON CONFLICT (usage_id) DO NOTHING`,
		usageID, sourceID, tenantID, gpuType, gpuSeconds, units, isPaper)
	if err != nil {
		return fmt.Errorf("supply: record usage: %w", err)
	}
	return nil
}

// UsageFor sums a source's metered time per tier in [from, to).
func (s *Store) UsageFor(ctx context.Context, sourceID string, from, to time.Time) ([]Usage, error) {
	rows, err := s.pool.Query(ctx, `SELECT gpu_type, coalesce(sum(gpu_seconds),0)::text, coalesce(sum(units),0)::text, count(*)
		FROM supply_usage WHERE source_id=$1 AND created_at >= $2 AND created_at < $3 GROUP BY gpu_type ORDER BY gpu_type`,
		sourceID, from, to)
	if err != nil {
		return nil, fmt.Errorf("supply: usage: %w", err)
	}
	defer rows.Close()
	var out []Usage
	for rows.Next() {
		var u Usage
		if err := rows.Scan(&u.GPUType, &u.GPUSeconds, &u.Units, &u.Sessions); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
