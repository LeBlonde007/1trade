package outbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

// Stream is the JetStream stream the engine publishes into. It captures both subjects so consumers
// see them in one order (surveillance creates the same stream if it starts first).
const Stream = "TRADING_EVENTS"

// DuplicateWindow is how long JetStream remembers message ids to drop republishes.
const DuplicateWindow = 10 * time.Minute

// JetStreamSink publishes with Nats-Msg-Id and waits for the stream's ack, so a returned nil means the
// event is durably in the stream.
type JetStreamSink struct{ js nats.JetStreamContext }

// NewJetStreamSink ensures the stream exists and returns a sink on it.
func NewJetStreamSink(nc *nats.Conn) (*JetStreamSink, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	if _, err := js.StreamInfo(Stream); err != nil {
		if _, err := js.AddStream(&nats.StreamConfig{
			Name: Stream, Subjects: []string{"orders.state.v1", "trades.executed.v1"},
			Storage: nats.FileStorage, Duplicates: DuplicateWindow,
		}); err != nil {
			return nil, fmt.Errorf("add stream: %w", err)
		}
	}
	return &JetStreamSink{js: js}, nil
}

// Publish sends one message and waits for the stream ack.
func (s *JetStreamSink) Publish(ctx context.Context, subject, msgID string, data []byte) error {
	m := nats.NewMsg(subject)
	m.Data = data
	m.Header.Set(nats.MsgIdHdr, msgID)
	if _, err := s.js.PublishMsg(m, nats.Context(ctx)); err != nil {
		return fmt.Errorf("publish %s: %w", subject, err)
	}
	return nil
}

// PGCursor stores the relay position in engine_outbox_cursor (migrations/0003_outbox.sql).
type PGCursor struct {
	pool *pgxpool.Pool
	name string
}

// NewPGCursor returns the named cursor.
func NewPGCursor(pool *pgxpool.Pool, name string) *PGCursor { return &PGCursor{pool: pool, name: name} }

// Load reads the position; a missing row is the start.
func (c *PGCursor) Load(ctx context.Context) (Position, error) {
	var seq int64
	var idx int
	err := c.pool.QueryRow(ctx, `SELECT next_seq, next_idx FROM engine_outbox_cursor WHERE name=$1`, c.name).Scan(&seq, &idx)
	if errors.Is(err, pgx.ErrNoRows) {
		return Position{Seq: 1}, nil
	}
	if err != nil {
		return Position{}, fmt.Errorf("load cursor: %w", err)
	}
	return Position{Seq: uint64(seq), Idx: idx}, nil //nolint:gosec // CHECK next_seq >= 1
}

// Save upserts the position.
func (c *PGCursor) Save(ctx context.Context, p Position) error {
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO engine_outbox_cursor (name, next_seq, next_idx) VALUES ($1,$2,$3)
		 ON CONFLICT (name) DO UPDATE SET next_seq=EXCLUDED.next_seq, next_idx=EXCLUDED.next_idx, updated_at=now()`,
		c.name, int64(p.Seq), p.Idx); err != nil { //nolint:gosec // journal seqs stay far below 2^63
		return fmt.Errorf("save cursor: %w", err)
	}
	return nil
}
