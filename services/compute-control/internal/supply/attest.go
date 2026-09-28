package supply

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Layers are the five attestation layers (supply.yaml v1.2, F19), in the order they are shown.
var Layers = []string{"kyb", "hardware", "challenge", "telemetry", "bond"}

// Layer results.
const (
	Pass  = "pass"
	Fail  = "fail"
	Drift = "drift"
)

// Challenge kinds.
const (
	KindHardware  = "hardware"  // a nonce the agent binds into the signed GPU identity report
	KindChallenge = "challenge" // a nonce the agent answers by iterated SHA-256 (presence, responsiveness)
)

// Challenge limits.
const (
	HardwareTTL         = 5 * time.Minute
	ChallengeTTL        = 30 * time.Second
	ChallengeIterations = 100_000
	MaxOpenChallenges   = 10
	maxReportGPUs       = 4096
)

// actorAttestation is the audit actor for transitions attestation makes by itself.
const actorAttestation = "attestation"

// Attestation errors surfaced to the API.
var (
	ErrAttestationIncomplete = errors.New("supply: attestation incomplete")
	ErrChallengeClosed       = errors.New("supply: challenge already answered or expired")
	ErrTooManyChallenges     = errors.New("supply: too many open challenges")
)

// LayerResult is one layer's latest recorded result.
type LayerResult struct {
	State      string    `json:"state"`
	AttestedAt time.Time `json:"attested_at"`
	Detail     string    `json:"detail"`
}

// Status is a source's attestation: each layer's latest result and which still block activation.
type Status struct {
	SourceID string                 `json:"source_id"`
	Complete bool                   `json:"complete"`
	Missing  []string               `json:"missing"`
	Layers   map[string]LayerResult `json:"layers"`
}

// Challenge is a nonce issued to a source's agent.
type Challenge struct {
	ID         string    `json:"id"`
	Kind       string    `json:"kind"`
	Nonce      string    `json:"nonce"`
	Iterations int       `json:"iterations"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// Answer is an agent's reply to a challenge: Report + Signature for hardware, Result for challenge.
type Answer struct {
	Report    []byte
	Signature []byte
	Result    string
}

// HardwareVerifier checks that a GPU identity report was signed by a trusted attestation root. The
// production implementation verifies an NVIDIA attestation token; Ed25519Verifier stands in for it.
type HardwareVerifier interface {
	Verify(report, signature []byte) error
}

// Ed25519Verifier trusts reports signed by any of Keys (the stand-in attestation root).
type Ed25519Verifier struct {
	Keys []ed25519.PublicKey
}

// Verify accepts the report when any trusted key verifies the signature.
func (v Ed25519Verifier) Verify(report, signature []byte) error {
	for _, k := range v.Keys {
		if ed25519.Verify(k, report, signature) {
			return nil
		}
	}
	return errors.New("signature does not verify against a trusted attestation key")
}

// ParseTrustKeys reads comma-separated base64 Ed25519 public keys (ATTESTATION_TRUST_KEYS).
func ParseTrustKeys(s string) ([]ed25519.PublicKey, error) {
	parts := strings.Split(s, ",")
	out := make([]ed25519.PublicKey, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(part)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("supply: attestation trust key %q is not a base64 Ed25519 public key", part)
		}
		out = append(out, ed25519.PublicKey(raw))
	}
	return out, nil
}

// ChallengeAnswer is the correct answer to a challenge: SHA-256 applied iterations times to the
// nonce, in hex. The agent computes the same thing.
func ChallengeAnswer(nonce []byte, iterations int) string {
	sum := sha256.Sum256(nonce)
	for i := 1; i < iterations; i++ {
		sum = sha256.Sum256(sum[:])
	}
	return hex.EncodeToString(sum[:])
}

// querier is what reading attestation needs from a pool or a transaction.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// latest reads each layer's most recent result for a source.
func latest(ctx context.Context, q querier, id string) (Status, error) {
	rows, err := q.Query(ctx, `SELECT DISTINCT ON (layer) layer, state, detail, attested_at
		FROM attestation_records WHERE source_id=$1 ORDER BY layer, id DESC`, id)
	if err != nil {
		return Status{}, fmt.Errorf("supply: attestation: %w", err)
	}
	defer rows.Close()
	st := Status{SourceID: id, Layers: map[string]LayerResult{}, Missing: []string{}}
	for rows.Next() {
		var layer string
		var r LayerResult
		if err := rows.Scan(&layer, &r.State, &r.Detail, &r.AttestedAt); err != nil {
			return Status{}, err
		}
		st.Layers[layer] = r
	}
	if err := rows.Err(); err != nil {
		return Status{}, err
	}
	for _, l := range Layers {
		if st.Layers[l].State != Pass {
			st.Missing = append(st.Missing, l)
		}
	}
	st.Complete = len(st.Missing) == 0
	return st, nil
}

// Attestation returns a source's attestation status (partner "" = operations).
func (s *Store) Attestation(ctx context.Context, id, partner string) (Status, error) {
	src, err := s.Get(ctx, id, partner)
	if err != nil {
		return Status{}, err
	}
	return latest(ctx, s.pool, src.ID)
}

// lockSource locks a source row for the rest of tx and checks the caller may act on it; a retired
// source takes no attestation (ErrState).
func lockSource(ctx context.Context, tx pgx.Tx, id, partner string) (Source, error) {
	if _, err := uuid.Parse(id); err != nil {
		return Source{}, ErrNotFound
	}
	src, err := scanSource(tx.QueryRow(ctx, `SELECT `+columns+` FROM supply_sources WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return Source{}, err
	}
	if partner != "" && src.PartnerTenantID != partner {
		return Source{}, ErrNotFound
	}
	if src.State == Retired {
		return Source{}, ErrState
	}
	return src, nil
}

// applyLayer appends a layer result and makes the transitions it implies: a fail or drift suspends
// an active source; the pass that completes attestation activates a pending one. Caller holds the
// source lock in tx.
func applyLayer(ctx context.Context, tx pgx.Tx, src Source, layer, state, detail string, evidence map[string]any, actor string) (Status, error) {
	if evidence == nil {
		evidence = map[string]any{}
	}
	ev, err := json.Marshal(evidence)
	if err != nil {
		return Status{}, fmt.Errorf("supply: evidence: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO attestation_records (source_id, layer, state, detail, evidence, actor)
		VALUES ($1,$2,$3,$4,$5,$6)`, src.ID, layer, state, detail, ev, actor); err != nil {
		return Status{}, fmt.Errorf("supply: record %s: %w", layer, err)
	}
	st, err := latest(ctx, tx, src.ID)
	if err != nil {
		return Status{}, err
	}
	to, kind := "", ""
	switch {
	case state != Pass && src.State == Active:
		to, kind = Suspended, "suspended"
	case state == Pass && src.State == Pending && st.Complete:
		to, kind = Active, "activated"
	}
	if to != "" {
		if _, err := tx.Exec(ctx, `UPDATE supply_sources SET state=$2, updated_at=now() WHERE id=$1`, src.ID, to); err != nil {
			return Status{}, fmt.Errorf("supply: attestation %s: %w", kind, err)
		}
		if err := event(ctx, tx, src.ID, kind, actorAttestation); err != nil {
			return Status{}, err
		}
	}
	return st, nil
}

// RecordLayer records a manual layer (kyb or bond, by operations) and applies its consequences.
func (s *Store) RecordLayer(ctx context.Context, id, layer, state string, evidence map[string]any, actor string) (Status, error) {
	if (layer != "kyb" && layer != "bond") || (state != Pass && state != Fail) {
		return Status{}, fmt.Errorf("supply: %s/%s cannot be recorded by hand", layer, state)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Status{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	src, err := lockSource(ctx, tx, id, "")
	if err != nil {
		return Status{}, err
	}
	st, err := applyLayer(ctx, tx, src, layer, state, "", evidence, actor)
	if err != nil {
		return Status{}, err
	}
	return st, tx.Commit(ctx)
}

// telemetry records the telemetry layer from a heartbeat when its result changes: uncorrectable ECC
// errors are drift; a clean report with healthy GPUs is pass; no healthy GPUs records nothing
// (Schedulable already keeps work away). Caller holds the source lock in tx.
func telemetry(ctx context.Context, tx pgx.Tx, src Source, healthy int, ecc int64) error {
	state, detail := Pass, ""
	switch {
	case ecc > 0:
		state, detail = Drift, fmt.Sprintf("%d uncorrectable ECC errors reported", ecc)
	case healthy == 0:
		return nil
	}
	cur, err := latest(ctx, tx, src.ID)
	if err != nil {
		return err
	}
	if prev, ok := cur.Layers["telemetry"]; ok && prev.State == state && state == Pass {
		return nil // unchanged; every drift is recorded
	}
	_, err = applyLayer(ctx, tx, src, "telemetry", state, detail, map[string]any{"gpus_healthy": healthy, "ecc_errors": ecc}, actorAttestation)
	return err
}

// IssueChallenge hands a source's agent a fresh nonce of the given kind. At most MaxOpenChallenges
// may be open at once, so a caller cannot grow the table without bound.
func (s *Store) IssueChallenge(ctx context.Context, id, partner, kind string, now time.Time) (Challenge, error) {
	iterations, ttl := 0, HardwareTTL
	switch kind {
	case KindHardware:
	case KindChallenge:
		iterations, ttl = ChallengeIterations, ChallengeTTL
	default:
		return Challenge{}, fmt.Errorf("supply: unknown challenge kind %q", kind)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Challenge{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	src, err := lockSource(ctx, tx, id, partner)
	if err != nil {
		return Challenge{}, err
	}
	var open int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM attestation_challenges WHERE source_id=$1 AND answered_at IS NULL
		AND expires_at > $2`, src.ID, now).Scan(&open); err != nil {
		return Challenge{}, fmt.Errorf("supply: challenges: %w", err)
	}
	if open >= MaxOpenChallenges {
		return Challenge{}, ErrTooManyChallenges
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return Challenge{}, fmt.Errorf("supply: nonce: %w", err)
	}
	c := Challenge{ID: uuid.NewString(), Kind: kind, Nonce: hex.EncodeToString(nonce), Iterations: iterations, ExpiresAt: now.Add(ttl).UTC()}
	if _, err := tx.Exec(ctx, `INSERT INTO attestation_challenges (id, source_id, kind, nonce, iterations, issued_at, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, c.ID, src.ID, kind, nonce, iterations, now, c.ExpiresAt); err != nil {
		return Challenge{}, fmt.Errorf("supply: issue challenge: %w", err)
	}
	return c, tx.Commit(ctx)
}

// AnswerChallenge takes the single allowed answer to a challenge and records the layer it proves
// (pass or fail). A late or second answer is ErrChallengeClosed and records nothing.
func (s *Store) AnswerChallenge(ctx context.Context, id, partner, challengeID string, a Answer, v HardwareVerifier, now time.Time) (Status, error) {
	if _, err := uuid.Parse(challengeID); err != nil {
		return Status{}, ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Status{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	src, err := lockSource(ctx, tx, id, partner)
	if err != nil {
		return Status{}, err
	}
	var (
		kind       string
		nonce      []byte
		iterations int
		issued     time.Time
		expires    time.Time
		answered   *time.Time
	)
	err = tx.QueryRow(ctx, `SELECT kind, nonce, iterations, issued_at, expires_at, answered_at FROM attestation_challenges
		WHERE id=$1 AND source_id=$2 FOR UPDATE`, challengeID, src.ID).Scan(&kind, &nonce, &iterations, &issued, &expires, &answered)
	if errors.Is(err, pgx.ErrNoRows) {
		return Status{}, ErrNotFound
	}
	if err != nil {
		return Status{}, fmt.Errorf("supply: challenge: %w", err)
	}
	if answered != nil || !now.Before(expires) {
		return Status{}, ErrChallengeClosed
	}
	if _, err := tx.Exec(ctx, `UPDATE attestation_challenges SET answered_at=$2 WHERE id=$1`, challengeID, now); err != nil {
		return Status{}, fmt.Errorf("supply: answer: %w", err)
	}

	state, detail := Pass, ""
	evidence := map[string]any{"challenge_id": challengeID, "response_ms": now.Sub(issued).Milliseconds()}
	if kind == KindChallenge {
		want := ChallengeAnswer(nonce, iterations)
		if subtle.ConstantTimeCompare([]byte(strings.ToLower(strings.TrimSpace(a.Result))), []byte(want)) != 1 {
			state, detail = Fail, "wrong challenge answer"
		}
	} else {
		gpus, model, verr := verifyHardware(src, nonce, a, v)
		if verr == nil {
			var cerr error
			if verr, cerr = claimGPUs(ctx, tx, src.ID, gpus); cerr != nil {
				return Status{}, cerr
			}
		}
		if verr != nil {
			state, detail = Fail, verr.Error()
		} else {
			evidence["gpu_uuids"], evidence["model"] = gpus, model
		}
	}
	st, err := applyLayer(ctx, tx, src, kind, state, detail, evidence, actorAttestation)
	if err != nil {
		return Status{}, err
	}
	return st, tx.Commit(ctx)
}

// hardwareReport is the signed GPU identity report an agent submits.
type hardwareReport struct {
	SourceID string `json:"source_id"`
	Nonce    string `json:"nonce"`
	GPUs     []struct {
		UUID  string `json:"uuid"`
		Model string `json:"model"`
	} `json:"gpus"`
}

// modelFor is the GPU model string a report must name for each registered tier.
var modelFor = map[string]string{"gpu_h100": "H100", "gpu_h200": "H200"}

// verifyHardware checks a report: signed by a trusted root, for this source, bound to this
// challenge's nonce, and showing at least the registered number of distinct GPUs of the registered
// model. It returns the GPU UUIDs and the model.
func verifyHardware(src Source, nonce []byte, a Answer, v HardwareVerifier) ([]string, string, error) {
	if v == nil {
		return nil, "", errors.New("no trusted attestation root is configured")
	}
	if err := v.Verify(a.Report, a.Signature); err != nil {
		return nil, "", err
	}
	var rep hardwareReport
	dec := json.NewDecoder(bytes.NewReader(a.Report))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rep); err != nil {
		return nil, "", errors.New("report is not valid JSON")
	}
	if rep.SourceID != src.ID {
		return nil, "", errors.New("report is for another source")
	}
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(rep.Nonce)), []byte(hex.EncodeToString(nonce))) != 1 {
		return nil, "", errors.New("report is not bound to this challenge's nonce")
	}
	if len(rep.GPUs) > maxReportGPUs {
		return nil, "", errors.New("report lists too many GPUs")
	}
	want := modelFor[src.GPUType]
	seen := map[string]bool{}
	gpus := make([]string, 0, len(rep.GPUs))
	for _, g := range rep.GPUs {
		id := strings.TrimSpace(g.UUID)
		if id == "" || len(id) > 128 {
			return nil, "", errors.New("report has a GPU without a valid UUID")
		}
		if !strings.Contains(strings.ToUpper(g.Model), want) {
			return nil, "", fmt.Errorf("GPU model %q does not match the registered %s", g.Model, want)
		}
		if !seen[id] {
			seen[id] = true
			gpus = append(gpus, id)
		}
	}
	if len(gpus) < src.GPUCount {
		return nil, "", fmt.Errorf("report shows %d distinct GPUs, the registration says %d", len(gpus), src.GPUCount)
	}
	return gpus, want, nil
}

// claimGPUs binds the attested GPUs to the source, refusing GPUs already attested to another source
// that is not retired: the same hardware cannot back two live sources. Claims are serialised. refused
// is why the claim failed attestation; err is a database failure.
func claimGPUs(ctx context.Context, tx pgx.Tx, sourceID string, gpus []string) (refused, err error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('attested_gpus'))`); err != nil {
		return nil, fmt.Errorf("supply: lock gpus: %w", err)
	}
	var taken string
	err = tx.QueryRow(ctx, `SELECT a.gpu_uuid FROM attested_gpus a JOIN supply_sources s ON s.id = a.source_id
		WHERE a.gpu_uuid = ANY($1) AND a.source_id <> $2 AND s.state <> 'retired' LIMIT 1`, gpus, sourceID).Scan(&taken)
	switch {
	case err == nil:
		return fmt.Errorf("GPU %s is attested to another live source", taken), nil
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, fmt.Errorf("supply: gpus: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO attested_gpus (gpu_uuid, source_id) SELECT unnest($1::text[]), $2
		ON CONFLICT (gpu_uuid) DO UPDATE SET source_id = excluded.source_id, updated_at = now()`, gpus, sourceID); err != nil {
		return nil, fmt.Errorf("supply: claim gpus: %w", err)
	}
	return nil, nil
}
