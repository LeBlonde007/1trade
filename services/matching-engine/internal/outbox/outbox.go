// Package outbox publishes the matching engine's events. The journal IS the outbox: a relay follows
// the journal command by command, re-derives each command's events on a shadow engine (replay is exact
// — see engine.Replay), and publishes them to NATS in emission order. Nothing in the order path waits
// on the network; a crash anywhere loses nothing, because the journal is durable and the relay resumes
// from its cursor.
//
// Delivery is at least once, in order. Every message carries a deterministic Nats-Msg-Id (the event's
// event_id or trade_id), so JetStream drops a republish inside its duplicate window, and consumers
// (surveillance, settlement) already dedupe on those ids beyond it.
package outbox

import (
	"context"
	"fmt"

	"github.com/trade1/matching-engine/internal/engine"
	"github.com/trade1/matching-engine/internal/events"
	"github.com/trade1/matching-engine/internal/journal"
)

// Source is the journal as the relay reads it.
type Source interface {
	ReadAfter(ctx context.Context, afterSeq uint64, prevHash string, limit int) ([]journal.Entry, error)
}

// Position is the next event to publish: event Idx of journal command Seq.
type Position struct {
	Seq uint64
	Idx int
}

// Cursor persists the relay's position.
type Cursor interface {
	Load(ctx context.Context) (Position, error)
	Save(ctx context.Context, p Position) error
}

// Sink publishes one message with a deduplication id.
type Sink interface {
	Publish(ctx context.Context, subject, msgID string, data []byte) error
}

// Relay follows the journal and publishes its events.
type Relay struct {
	src    Source
	cursor Cursor
	sink   Sink
	shadow *engine.Engine
	seq    uint64 // last journal seq applied to the shadow engine
	head   string // chain hash of that entry
	pos    Position
	loaded bool
}

// New builds a relay. cfg must carry the journal's epoch and the live engine's fee schedule, so the
// shadow engine derives byte-identical events; Risk and Persist are cleared (journaled commands carry
// their risk decision, and the shadow never writes).
func New(cfg engine.Config, src Source, cursor Cursor, sink Sink) *Relay {
	cfg.Risk, cfg.Persist = nil, nil
	return &Relay{src: src, cursor: cursor, sink: sink, shadow: engine.New(cfg)}
}

// Step applies every new journal entry to the shadow engine and publishes the events at or after the
// cursor, saving the cursor after each command. It returns how many messages it published. On a
// publish failure it saves the partial position and returns the error; the next Step resumes there.
func (r *Relay) Step(ctx context.Context) (int, error) {
	if !r.loaded {
		p, err := r.cursor.Load(ctx)
		if err != nil {
			return 0, fmt.Errorf("outbox: load cursor: %w", err)
		}
		if p.Seq == 0 {
			p = Position{Seq: 1}
		}
		r.pos, r.loaded = p, true
	}
	entries, err := r.src.ReadAfter(ctx, r.seq, r.head, 1000)
	if err != nil {
		return 0, fmt.Errorf("outbox: read journal: %w", err)
	}
	published := 0
	for _, e := range entries {
		evs, err := r.shadow.Apply(e.Command)
		if err != nil {
			// The live engine accepted this command, so the shadow must too; anything else means the
			// shadow's config (epoch, fees) differs from the engine's. Stop rather than publish lies.
			return published, fmt.Errorf("outbox: shadow rejected journal seq %d: %w", e.Seq, err)
		}
		r.seq, r.head = e.Seq, e.ChainHash
		if e.Seq < r.pos.Seq {
			continue // published before; the shadow only needed the state
		}
		start := 0
		if e.Seq == r.pos.Seq {
			start = r.pos.Idx
		}
		for i := start; i < len(evs); i++ {
			subject, data, err := events.Encode(evs[i])
			if err != nil {
				return published, fmt.Errorf("outbox: encode seq %d event %d: %w", e.Seq, i, err)
			}
			if err := r.sink.Publish(ctx, subject, msgID(evs[i]), data); err != nil {
				r.pos = Position{Seq: e.Seq, Idx: i}
				_ = r.cursor.Save(ctx, r.pos)
				return published, fmt.Errorf("outbox: publish seq %d event %d: %w", e.Seq, i, err)
			}
			published++
		}
		r.pos = Position{Seq: e.Seq + 1}
		if err := r.cursor.Save(ctx, r.pos); err != nil {
			return published, fmt.Errorf("outbox: save cursor: %w", err)
		}
	}
	return published, nil
}

// msgID is the event's deterministic id: trade_id for trades, event_id for order transitions.
func msgID(ev engine.Event) string {
	if ev.Trade != nil {
		return ev.Trade.TradeID
	}
	return ev.Order.EventID
}
