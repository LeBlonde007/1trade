-- 0001_journal.sql — matching-engine command journal (KW03). The engine's source of truth: its books
-- are rebuilt on start by replaying this table in seq order (SPEC.md §5).
-- Append-only and hash-chained: chain_hash = SHA-256(prev_hash || command_json), over the exact stored
-- text, so any edit, deletion or reordering breaks verification on the next load.
-- Rollback: forward-only. A bad command is never removed; it is superseded by later commands.

CREATE TABLE IF NOT EXISTS engine_journal (
    seq          BIGINT PRIMARY KEY CHECK (seq > 0),        -- 1-based, contiguous; the PK makes a second writer fail loudly
    kind         TEXT NOT NULL CHECK (kind IN ('submit', 'cancel', 'expire_day')),
    command_json TEXT NOT NULL,                             -- exact bytes that were hashed (TEXT, not JSONB: JSONB would re-serialise them)
    prev_hash    TEXT NOT NULL,                             -- '' for seq 1
    chain_hash   TEXT NOT NULL UNIQUE,
    recorded_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Append-only guard: block UPDATE / DELETE / TRUNCATE at the DB layer.
CREATE OR REPLACE FUNCTION engine_journal_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'engine_journal is append-only (no % allowed)', TG_OP;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_engine_journal_append_only ON engine_journal;
CREATE TRIGGER trg_engine_journal_append_only
    BEFORE UPDATE OR DELETE ON engine_journal
    FOR EACH ROW EXECUTE FUNCTION engine_journal_append_only();

DROP TRIGGER IF EXISTS trg_engine_journal_no_truncate ON engine_journal;
CREATE TRIGGER trg_engine_journal_no_truncate
    BEFORE TRUNCATE ON engine_journal
    FOR EACH STATEMENT EXECUTE FUNCTION engine_journal_append_only();
