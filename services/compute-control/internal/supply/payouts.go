package supply

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Payout errors surfaced to the API.
var (
	ErrNoAgreement   = errors.New("supply: no payout agreement for this partner")
	ErrBadAgreement  = errors.New("supply: invalid agreement")
	ErrPaperPayout   = errors.New("supply: a paper statement is a simulation and is never wired")
	ErrDisputeWindow = errors.New("supply: outside the dispute window")
)

// Payout states.
const (
	PayoutPending  = "pending"
	PayoutWired    = "wired"
	PayoutDisputed = "disputed"
	PayoutSettled  = "settled"
)

// Agreement is a partner's payout terms (supply.yaml v1.1). Amounts are fixed-point strings.
type Agreement struct {
	PartnerTenantID string            `json:"partner_tenant_id"`
	Rates           map[string]string `json:"rates"` // USD per GPU-hour, per tier
	FeePercent      string            `json:"fee_percent"`
	HoldbackPercent string            `json:"holdback_percent"`
	DisputeDays     int               `json:"dispute_days"`
}

// Line is one tier on a statement.
type Line struct {
	GPUType  string `json:"gpu_type"`
	GPUHours string `json:"gpu_hours"`
	Rate     string `json:"rate"`
	Gross    string `json:"gross"`
}

// Payout is one partner statement.
type Payout struct {
	ID              string
	PartnerTenantID string
	PeriodStart     time.Time
	PeriodEnd       time.Time
	IsPaper         bool
	GPUSeconds      string
	Gross           string
	Fee             string
	Amount          string // gross − fee
	Holdback        string
	Released        string // amount − holdback
	UsageRecords    int
	State           string
	WireReference   *string
	DisputeUntil    time.Time
	DisputeReason   *string
	Resolution      *string
	Lines           []Line
}

// rat parses a fixed-point decimal string.
func rat(s string) (*big.Rat, bool) {
	return new(big.Rat).SetString(strings.TrimSpace(s))
}

// fixed6 renders a non-negative amount truncated to 6 decimal places.
func fixed6(r *big.Rat) string {
	scaled := new(big.Int).Quo(new(big.Int).Mul(r.Num(), big.NewInt(1_000_000)), r.Denom())
	s := fmt.Sprintf("%07d", scaled)
	return s[:len(s)-6] + "." + s[len(s)-6:]
}

// percentOf is amount × pct / 100, truncated to 6 places.
func percentOf(amount *big.Rat, pct string) (*big.Rat, error) {
	p, ok := rat(pct)
	if !ok {
		return nil, ErrBadAgreement
	}
	v, _ := new(big.Rat).SetString(fixed6(new(big.Rat).Quo(new(big.Rat).Mul(amount, p), big.NewRat(100, 1))))
	return v, nil
}

// validAgreement checks tiers, rates (>= 0, at most 6 decimals), percents (0-100) and the window.
func validAgreement(a Agreement) error {
	if len(a.Rates) == 0 || a.DisputeDays < 1 || a.DisputeDays > 90 {
		return ErrBadAgreement
	}
	for tier, r := range a.Rates {
		v, ok := rat(r)
		if (tier != "gpu_h100" && tier != "gpu_h200") || !six(r) || !ok || v.Sign() < 0 {
			return ErrBadAgreement
		}
	}
	for _, p := range []string{a.FeePercent, a.HoldbackPercent} {
		v, ok := rat(p)
		if !ok || !six(p) || v.Sign() < 0 || v.Cmp(big.NewRat(100, 1)) > 0 {
			return ErrBadAgreement
		}
	}
	return nil
}

// six reports whether s is a plain decimal with at most 6 fractional digits.
func six(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, "eE/+-") {
		return false
	}
	if i := strings.IndexByte(s, '.'); i >= 0 {
		return len(s)-i-1 <= 6 && i > 0
	}
	return true
}

// SetAgreement stores a partner's terms (operations).
func (s *Store) SetAgreement(ctx context.Context, a Agreement) (Agreement, error) {
	if _, err := uuid.Parse(a.PartnerTenantID); err != nil {
		return Agreement{}, ErrBadAgreement
	}
	if err := validAgreement(a); err != nil {
		return Agreement{}, err
	}
	rates, _ := json.Marshal(a.Rates)
	if _, err := s.pool.Exec(ctx, `INSERT INTO partner_agreements (partner_tenant_id, rates, fee_percent, holdback_percent, dispute_days)
		VALUES ($1,$2,$3::numeric,$4::numeric,$5)
		ON CONFLICT (partner_tenant_id) DO UPDATE SET rates=EXCLUDED.rates, fee_percent=EXCLUDED.fee_percent,
		  holdback_percent=EXCLUDED.holdback_percent, dispute_days=EXCLUDED.dispute_days, updated_at=now()`,
		a.PartnerTenantID, rates, a.FeePercent, a.HoldbackPercent, a.DisputeDays); err != nil {
		return Agreement{}, fmt.Errorf("supply: set agreement: %w", err)
	}
	return s.GetAgreement(ctx, a.PartnerTenantID)
}

// GetAgreement reads a partner's terms.
func (s *Store) GetAgreement(ctx context.Context, partner string) (Agreement, error) {
	if _, err := uuid.Parse(partner); err != nil {
		return Agreement{}, ErrNoAgreement
	}
	var a Agreement
	var rates []byte
	err := s.pool.QueryRow(ctx, `SELECT partner_tenant_id::text, rates, fee_percent::text, holdback_percent::text, dispute_days
		FROM partner_agreements WHERE partner_tenant_id=$1`, partner).Scan(&a.PartnerTenantID, &rates, &a.FeePercent, &a.HoldbackPercent, &a.DisputeDays)
	if errors.Is(err, pgx.ErrNoRows) {
		return Agreement{}, ErrNoAgreement
	}
	if err != nil {
		return Agreement{}, err
	}
	if err := json.Unmarshal(rates, &a.Rates); err != nil {
		return Agreement{}, err
	}
	return a, nil
}

// group is unpaid usage for one (partner, is_paper, tier).
type group struct {
	partner  string
	isPaper  bool
	gpuType  string
	seconds  string
	usageIDs []string
}

// CloseCycle turns unpaid usage created in [from, to) into statements: one per (partner, is_paper),
// for partners with an agreement, over tiers the agreement prices. It holds an exclusive lock on the
// usage-to-statement map for the transaction, so two closes can never pay the same usage; a repeat
// finds nothing unpaid and creates nothing.
func (s *Store) CloseCycle(ctx context.Context, from, to, now time.Time, actor string) ([]Payout, error) {
	if !from.Before(to) || to.After(now) {
		return nil, fmt.Errorf("%w: the period must end in the past", ErrBadAgreement)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	if _, err := tx.Exec(ctx, `LOCK TABLE payout_usage IN EXCLUSIVE MODE`); err != nil {
		return nil, fmt.Errorf("supply: lock: %w", err)
	}
	rows, err := tx.Query(ctx, `SELECT s.partner_tenant_id::text, u.is_paper, u.gpu_type, sum(u.gpu_seconds)::text, array_agg(u.usage_id ORDER BY u.usage_id)
		FROM supply_usage u
		JOIN supply_sources s ON s.id::text = u.source_id
		LEFT JOIN payout_usage pu ON pu.usage_id = u.usage_id
		WHERE pu.usage_id IS NULL AND u.created_at >= $1 AND u.created_at < $2
		GROUP BY 1, 2, 3 ORDER BY 1, 2, 3`, from, to)
	if err != nil {
		return nil, fmt.Errorf("supply: unpaid usage: %w", err)
	}
	var groups []group
	for rows.Next() {
		var g group
		if err := rows.Scan(&g.partner, &g.isPaper, &g.gpuType, &g.seconds, &g.usageIDs); err != nil {
			rows.Close()
			return nil, err
		}
		groups = append(groups, g)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	type key struct {
		partner string
		paper   bool
	}
	byKey := map[key][]group{}
	var keys []key
	for _, g := range groups {
		k := key{g.partner, g.isPaper}
		if _, ok := byKey[k]; !ok {
			keys = append(keys, k)
		}
		byKey[k] = append(byKey[k], g)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].partner < keys[j].partner || (keys[i].partner == keys[j].partner && !keys[i].paper)
	})
	agreements := map[string]*Agreement{}
	out := make([]Payout, 0, len(keys))
	for _, k := range keys {
		ag, seen := agreements[k.partner]
		if !seen {
			a, err := getAgreementTx(ctx, tx, k.partner)
			if err != nil && !errors.Is(err, ErrNoAgreement) {
				return nil, err
			}
			if err == nil {
				ag = &a
			}
			agreements[k.partner] = ag
		}
		if ag == nil {
			continue // no terms yet: the usage stays unpaid until there are
		}
		p, ids, err := statement(ag, k.partner, k.paper, byKey[k], from, to, now)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			continue
		}
		if err := insertPayout(ctx, tx, p, ids, actor); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, tx.Commit(ctx)
}

// statement computes one partner statement from its tier groups; tiers without a rate are left
// unpaid. It returns the usage ids it covers.
func statement(ag *Agreement, partner string, paper bool, gs []group, from, to, now time.Time) (Payout, []string, error) {
	p := Payout{ID: uuid.NewString(), PartnerTenantID: partner, PeriodStart: from, PeriodEnd: to, IsPaper: paper,
		State: PayoutPending, DisputeUntil: now.Add(time.Duration(ag.DisputeDays) * 24 * time.Hour)}
	gross, seconds := new(big.Rat), new(big.Rat)
	var ids []string
	for _, g := range gs {
		rate, ok := ag.Rates[g.gpuType]
		if !ok {
			continue
		}
		r, _ := rat(rate)
		secs, ok := rat(g.seconds)
		if !ok {
			return Payout{}, nil, fmt.Errorf("supply: bad usage sum %q", g.seconds)
		}
		hours := new(big.Rat).Quo(secs, big.NewRat(3600, 1))
		lineGross, _ := rat(fixed6(new(big.Rat).Mul(hours, r)))
		gross.Add(gross, lineGross)
		seconds.Add(seconds, secs)
		p.Lines = append(p.Lines, Line{GPUType: g.gpuType, GPUHours: fixed6(hours), Rate: fixed6(r), Gross: fixed6(lineGross)})
		ids = append(ids, g.usageIDs...)
	}
	fee, err := percentOf(gross, ag.FeePercent)
	if err != nil {
		return Payout{}, nil, err
	}
	amount := new(big.Rat).Sub(gross, fee)
	holdback, err := percentOf(amount, ag.HoldbackPercent)
	if err != nil {
		return Payout{}, nil, err
	}
	p.GPUSeconds, p.Gross, p.Fee, p.Amount = fixed6(seconds), fixed6(gross), fixed6(fee), fixed6(amount)
	p.Holdback, p.Released = fixed6(holdback), fixed6(new(big.Rat).Sub(amount, holdback))
	p.UsageRecords = len(ids)
	return p, ids, nil
}

// getAgreementTx reads terms inside the cycle's transaction.
func getAgreementTx(ctx context.Context, tx pgx.Tx, partner string) (Agreement, error) {
	var a Agreement
	var rates []byte
	err := tx.QueryRow(ctx, `SELECT partner_tenant_id::text, rates, fee_percent::text, holdback_percent::text, dispute_days
		FROM partner_agreements WHERE partner_tenant_id=$1`, partner).Scan(&a.PartnerTenantID, &rates, &a.FeePercent, &a.HoldbackPercent, &a.DisputeDays)
	if errors.Is(err, pgx.ErrNoRows) {
		return Agreement{}, ErrNoAgreement
	}
	if err != nil {
		return Agreement{}, err
	}
	return a, json.Unmarshal(rates, &a.Rates)
}

// insertPayout writes a statement, its lines, the usage it covers, and its audit row.
func insertPayout(ctx context.Context, tx pgx.Tx, p Payout, ids []string, actor string) error {
	if _, err := tx.Exec(ctx, `INSERT INTO partner_payouts (id, partner_tenant_id, period_start, period_end, is_paper, gpu_seconds,
		gross, fee, payout, holdback, released, usage_records, state, dispute_until)
		VALUES ($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8::numeric,$9::numeric,$10::numeric,$11::numeric,$12,'pending',$13)`,
		p.ID, p.PartnerTenantID, p.PeriodStart, p.PeriodEnd, p.IsPaper, p.GPUSeconds, p.Gross, p.Fee, p.Amount, p.Holdback,
		p.Released, p.UsageRecords, p.DisputeUntil); err != nil {
		return fmt.Errorf("supply: insert statement: %w", err)
	}
	for _, l := range p.Lines {
		secs := new(big.Rat)
		h, _ := rat(l.GPUHours)
		secs.Mul(h, big.NewRat(3600, 1))
		if _, err := tx.Exec(ctx, `INSERT INTO partner_payout_lines (payout_id, gpu_type, gpu_seconds, rate, gross)
			VALUES ($1,$2,$3::numeric,$4::numeric,$5::numeric)`, p.ID, l.GPUType, fixed6(secs), l.Rate, l.Gross); err != nil {
			return fmt.Errorf("supply: insert line: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO payout_usage (usage_id, payout_id) SELECT unnest($1::text[]), $2`, ids, p.ID); err != nil {
		return fmt.Errorf("supply: map usage: %w", err)
	}
	return payoutEvent(ctx, tx, p.ID, "created", actor, "")
}

// payoutEvent appends one payout audit row.
func payoutEvent(ctx context.Context, tx pgx.Tx, id, kind, actor, note string) error {
	if _, err := tx.Exec(ctx, `INSERT INTO payout_events (payout_id, kind, actor, note) VALUES ($1,$2,$3,NULLIF($4,''))`, id, kind, actor, note); err != nil {
		return fmt.Errorf("supply: payout audit: %w", err)
	}
	return nil
}

// payoutCols is the select list scanPayout reads.
const payoutCols = `id::text, partner_tenant_id::text, period_start, period_end, is_paper, gpu_seconds::text, gross::text, fee::text,
	payout::text, holdback::text, released::text, usage_records, state, wire_reference, dispute_until, dispute_reason, resolution`

// scanPayout reads one statement row.
func scanPayout(row pgx.Row) (Payout, error) {
	var p Payout
	err := row.Scan(&p.ID, &p.PartnerTenantID, &p.PeriodStart, &p.PeriodEnd, &p.IsPaper, &p.GPUSeconds, &p.Gross, &p.Fee,
		&p.Amount, &p.Holdback, &p.Released, &p.UsageRecords, &p.State, &p.WireReference, &p.DisputeUntil, &p.DisputeReason, &p.Resolution)
	if errors.Is(err, pgx.ErrNoRows) {
		return Payout{}, ErrNotFound
	}
	return p, err
}

// ListPayouts returns statements newest first: a partner's own, or all when partner is "" (optionally
// filtered by state — `disputed` is the review queue).
func (s *Store) ListPayouts(ctx context.Context, partner, state string) ([]Payout, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+payoutCols+` FROM partner_payouts
		WHERE ($1 = '' OR partner_tenant_id::text = $1) AND ($2 = '' OR state = $2) ORDER BY created_at DESC, id LIMIT 500`, partner, state)
	if err != nil {
		return nil, fmt.Errorf("supply: list payouts: %w", err)
	}
	defer rows.Close()
	var out []Payout
	for rows.Next() {
		p, err := scanPayout(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetPayout returns one statement with its lines; another partner's reads as ErrNotFound.
func (s *Store) GetPayout(ctx context.Context, id, partner string) (Payout, error) {
	if _, err := uuid.Parse(id); err != nil {
		return Payout{}, ErrNotFound
	}
	p, err := scanPayout(s.pool.QueryRow(ctx, `SELECT `+payoutCols+` FROM partner_payouts WHERE id=$1`, id))
	if err != nil {
		return Payout{}, err
	}
	if partner != "" && p.PartnerTenantID != partner {
		return Payout{}, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `SELECT gpu_type, (gpu_seconds/3600)::numeric(20,6)::text, rate::text, gross::text
		FROM partner_payout_lines WHERE payout_id=$1 ORDER BY gpu_type`, id)
	if err != nil {
		return Payout{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var l Line
		if err := rows.Scan(&l.GPUType, &l.GPUHours, &l.Rate, &l.Gross); err != nil {
			return Payout{}, err
		}
		p.Lines = append(p.Lines, l)
	}
	return p, rows.Err()
}

// payoutAction locks a statement, checks it, applies a change, and audits it.
func (s *Store) payoutAction(ctx context.Context, id, partner, kind, actor, note string,
	check func(p Payout) error, set string, args ...any) (Payout, error) {
	if _, err := uuid.Parse(id); err != nil {
		return Payout{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Payout{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	p, err := scanPayout(tx.QueryRow(ctx, `SELECT `+payoutCols+` FROM partner_payouts WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return Payout{}, err
	}
	if partner != "" && p.PartnerTenantID != partner {
		return Payout{}, ErrNotFound
	}
	if err := check(p); err != nil {
		return Payout{}, err
	}
	p, err = scanPayout(tx.QueryRow(ctx, `UPDATE partner_payouts SET `+set+`, updated_at=now() WHERE id=$1 RETURNING `+payoutCols,
		append([]any{id}, args...)...))
	if err != nil {
		return Payout{}, fmt.Errorf("supply: %s: %w", kind, err)
	}
	if err := payoutEvent(ctx, tx, id, kind, actor, note); err != nil {
		return Payout{}, err
	}
	return p, tx.Commit(ctx)
}

// Wire records that operations sent the released amount (pending → wired). Paper statements refuse.
func (s *Store) Wire(ctx context.Context, id, reference, actor string) (Payout, error) {
	return s.payoutAction(ctx, id, "", "wired", actor, reference, func(p Payout) error {
		if p.IsPaper {
			return ErrPaperPayout
		}
		if p.State != PayoutPending {
			return ErrState
		}
		return nil
	}, `state='wired', wire_reference=$2`, reference)
}

// Dispute freezes a statement for review, inside its dispute window (pending or wired → disputed).
func (s *Store) Dispute(ctx context.Context, id, partner, reason string, now time.Time) (Payout, error) {
	return s.payoutAction(ctx, id, partner, "disputed", partner, reason, func(p Payout) error {
		if p.State != PayoutPending && p.State != PayoutWired {
			return ErrState
		}
		if !now.Before(p.DisputeUntil) {
			return ErrDisputeWindow
		}
		return nil
	}, `state='disputed', dispute_reason=$2`, reason)
}

// ReleaseHoldback settles a wired statement once its dispute window has passed undisputed.
func (s *Store) ReleaseHoldback(ctx context.Context, id, actor string, now time.Time) (Payout, error) {
	return s.payoutAction(ctx, id, "", "settled", actor, "", func(p Payout) error {
		if p.State != PayoutWired {
			return ErrState
		}
		if now.Before(p.DisputeUntil) {
			return ErrDisputeWindow
		}
		return nil
	}, `state='settled', resolution='release'`)
}

// Resolve closes a dispute: release or withhold the holdback. A statement already wired settles;
// one not yet wired returns to pending (its released amount still has to be sent).
func (s *Store) Resolve(ctx context.Context, id, outcome, note, actor string) (Payout, error) {
	if outcome != "release" && outcome != "withhold" {
		return Payout{}, ErrBadAgreement
	}
	return s.payoutAction(ctx, id, "", "resolved", actor, outcome+": "+note, func(p Payout) error {
		if p.State != PayoutDisputed {
			return ErrState
		}
		return nil
	}, `state = CASE WHEN wire_reference IS NULL THEN 'pending' ELSE 'settled' END, resolution=$2, resolution_note=$3`, outcome, note)
}
