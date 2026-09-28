-- 0003_attestation.sql — GPU attestation (supply.yaml v1.2, F19). A source activates only when its
-- latest result on every layer is pass; a later fail or drift suspends it. Rollback: forward-only in prod.

-- Every layer result ever recorded, append-only: the attestation history is evidence.
CREATE TABLE IF NOT EXISTS attestation_records (
    id          BIGSERIAL PRIMARY KEY,
    source_id   UUID NOT NULL REFERENCES supply_sources(id),
    layer       TEXT NOT NULL CHECK (layer IN ('kyb', 'hardware', 'challenge', 'telemetry', 'bond')),
    state       TEXT NOT NULL CHECK (state IN ('pass', 'fail', 'drift')),
    detail      TEXT NOT NULL DEFAULT '',
    evidence    JSONB NOT NULL DEFAULT '{}'::jsonb,
    actor       TEXT NOT NULL,             -- tenant id, 'service' for operations, 'attestation' for automatic checks
    attested_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_attestation_records_latest ON attestation_records (source_id, layer, id DESC);
DROP TRIGGER IF EXISTS trg_attestation_records_append_only ON attestation_records;
CREATE TRIGGER trg_attestation_records_append_only BEFORE UPDATE OR DELETE ON attestation_records
    FOR EACH ROW EXECUTE FUNCTION supply_events_append_only();

-- Nonces handed to a source's agent. answered_at is set exactly once (the single allowed answer).
CREATE TABLE IF NOT EXISTS attestation_challenges (
    id          UUID PRIMARY KEY,
    source_id   UUID NOT NULL REFERENCES supply_sources(id),
    kind        TEXT NOT NULL CHECK (kind IN ('hardware', 'challenge')),
    nonce       BYTEA NOT NULL CHECK (length(nonce) = 32),
    iterations  INT NOT NULL CHECK (iterations >= 0),
    issued_at   TIMESTAMPTZ NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    answered_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_attestation_challenges_open ON attestation_challenges (source_id) WHERE answered_at IS NULL;

-- Which source each physical GPU (by its hardware UUID) was attested to, so the same GPUs cannot back
-- two live sources.
CREATE TABLE IF NOT EXISTS attested_gpus (
    gpu_uuid   TEXT PRIMARY KEY,
    source_id  UUID NOT NULL REFERENCES supply_sources(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
