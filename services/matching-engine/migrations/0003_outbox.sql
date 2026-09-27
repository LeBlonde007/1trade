-- 0003_outbox.sql — event relay progress (KW03). The journal is the outbox: the relay re-derives every
-- event from the journal and publishes it; this row records how far it got, as (seq, idx) = the next
-- event to publish is event idx of journal command seq. Mutable by design — it is a cursor, not a
-- record. Losing it is safe: every message carries a deterministic Nats-Msg-Id and consumers dedupe on
-- event_id / trade_id, so a republish is harmless.

CREATE TABLE IF NOT EXISTS engine_outbox_cursor (
    name       TEXT PRIMARY KEY,
    next_seq   BIGINT NOT NULL CHECK (next_seq >= 1),
    next_idx   INTEGER NOT NULL CHECK (next_idx >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
