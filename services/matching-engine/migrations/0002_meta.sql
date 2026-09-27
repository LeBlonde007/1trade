-- 0002_meta.sql — journal metadata (KW03). Holds the journal's epoch: a random id created once, when
-- the journal is born, and mixed into every derived trade / event id so two journals can never mint
-- the same trade_id (the ledger is idempotent on it and would refuse the second as a conflict).
-- Write-once: a row is inserted and never changed.

CREATE TABLE IF NOT EXISTS engine_meta (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

DROP TRIGGER IF EXISTS trg_engine_meta_write_once ON engine_meta;
CREATE TRIGGER trg_engine_meta_write_once
    BEFORE UPDATE OR DELETE ON engine_meta
    FOR EACH ROW EXECUTE FUNCTION engine_journal_append_only();
