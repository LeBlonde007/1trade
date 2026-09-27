// Package journal is the durable command journal for the matching engine: a hash-chained,
// append-only Postgres table that the engine writes ahead of every state change and replays on start.
//
// The engine stays pure. This package adapts it to Postgres through engine.Config.Persist, and
// Recover rebuilds an engine from the table — verifying the chain first, so a tampered or gapped
// journal refuses to start rather than silently producing different books.
package journal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trade1/matching-engine/internal/engine"
)

// ErrCorrupt means the stored journal failed verification: a gap in seq, a broken chain link, or a
// row whose hash does not recompute. The engine must not start from it.
var ErrCorrupt = errors.New("journal: verification failed")

// ErrOutOfOrder means an append was not for the next expected seq — usually a second writer.
var ErrOutOfOrder = errors.New("journal: append out of order")

// writeTimeout bounds one append. It runs under the engine lock, so it must never hang the book.
const writeTimeout = 2 * time.Second

// Store is the Postgres-backed journal. One Store is the single writer for its table.
type Store struct {
	pool *pgxpool.Pool

	mu   sync.Mutex
	next uint64 // seq the next append must carry
	head string // chain_hash of the last row
}

// Open connects to Postgres and verifies the connection. Call Load (or Recover) before appending, so
// the store knows where the chain ends.
func Open(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("journal: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("journal: ping: %w", err)
	}
	return &Store{pool: pool, next: 1}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// kindOf names a command for the kind column.
func kindOf(c engine.Command) string {
	switch {
	case c.Submit != nil:
		return "submit"
	case c.Cancel != nil:
		return "cancel"
	default:
		return "expire_day"
	}
}

// link computes chain_hash = SHA-256(prev || command_json).
func link(prev, commandJSON string) string {
	sum := sha256.Sum256([]byte(prev + commandJSON))
	return hex.EncodeToString(sum[:])
}

// Append durably writes command seq. It is the engine's Persist hook: it must be called with seq in
// order, and it returns only after the row is committed.
func (s *Store) Append(seq uint64, c engine.Command) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if seq != s.next {
		return fmt.Errorf("%w: got seq %d, want %d", ErrOutOfOrder, seq, s.next)
	}
	if seq > math.MaxInt64 {
		return fmt.Errorf("%w: seq %d exceeds BIGINT", ErrOutOfOrder, seq)
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("journal: encode seq %d: %w", seq, err)
	}
	hash := link(s.head, string(raw))
	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO engine_journal (seq, kind, command_json, prev_hash, chain_hash) VALUES ($1, $2, $3, $4, $5)`,
		int64(seq), kindOf(c), string(raw), s.head, hash); err != nil { //nolint:gosec // bounded above
		return fmt.Errorf("journal: append seq %d: %w", seq, err)
	}
	s.next, s.head = seq+1, hash
	return nil
}

// Load reads and verifies the whole journal in seq order: seq must run 1..n with no gaps, each row
// must link to its predecessor, and each hash must recompute from the stored text. On success the
// store is positioned to append seq n+1.
func (s *Store) Load(ctx context.Context) ([]engine.Command, error) {
	rows, err := s.pool.Query(ctx, `SELECT seq, command_json, prev_hash, chain_hash FROM engine_journal ORDER BY seq`)
	if err != nil {
		return nil, fmt.Errorf("journal: load: %w", err)
	}
	defer rows.Close()

	var (
		out  []engine.Command
		head string
		want int64 = 1
	)
	for rows.Next() {
		var (
			seq             int64
			raw, prev, hash string
		)
		if err := rows.Scan(&seq, &raw, &prev, &hash); err != nil {
			return nil, fmt.Errorf("journal: scan: %w", err)
		}
		if seq != want {
			return nil, fmt.Errorf("%w: expected seq %d, found %d", ErrCorrupt, want, seq)
		}
		if prev != head || link(prev, raw) != hash {
			return nil, fmt.Errorf("%w: chain broken at seq %d", ErrCorrupt, seq)
		}
		var c engine.Command
		if err := json.Unmarshal([]byte(raw), &c); err != nil {
			return nil, fmt.Errorf("%w: seq %d does not decode: %w", ErrCorrupt, seq, err)
		}
		out = append(out, c)
		head, want = hash, want+1
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("journal: load: %w", err)
	}
	s.mu.Lock()
	s.next, s.head = uint64(want), head // #nosec G115 -- want >= 1
	s.mu.Unlock()
	return out, nil
}

// Recover loads and verifies the journal, replays it into a fresh engine, and attaches the store as
// that engine's write-ahead hook. cfg.Persist is overwritten. Use it at process start, before the
// engine accepts any command.
func Recover(ctx context.Context, s *Store, cfg engine.Config) (*engine.Engine, error) {
	cmds, err := s.Load(ctx)
	if err != nil {
		return nil, err
	}
	cfg.Persist = s.Append
	e, _, err := engine.Replay(cfg, cmds)
	if err != nil {
		return nil, fmt.Errorf("journal: replay: %w", err)
	}
	return e, nil
}
