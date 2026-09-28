-- 0001_supply.sql — partner supply sources (supply.yaml v1.0, F16/F17) and per-source usage.
-- compute-control's own tables in the shared database. Rollback: forward-only in prod.

CREATE TABLE IF NOT EXISTS supply_sources (
    id                UUID PRIMARY KEY,
    partner_tenant_id UUID NOT NULL,
    name              TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 80),
    gpu_type          TEXT NOT NULL CHECK (gpu_type IN ('gpu_h100', 'gpu_h200')),
    gpu_count         INT  NOT NULL CHECK (gpu_count BETWEEN 1 AND 4096),
    region            TEXT NOT NULL CHECK (length(region) BETWEEN 1 AND 64),
    sla_tier          TEXT NOT NULL CHECK (sla_tier IN ('bronze', 'silver', 'gold')),
    state             TEXT NOT NULL CHECK (state IN ('pending', 'active', 'suspended', 'retired')),
    gpus_healthy      INT CHECK (gpus_healthy >= 0),
    utilization_pct   INT CHECK (utilization_pct BETWEEN 0 AND 100),
    ecc_errors        BIGINT NOT NULL DEFAULT 0 CHECK (ecc_errors >= 0),
    last_heartbeat_at TIMESTAMPTZ,
    idempotency_key   TEXT NOT NULL,
    request_hash      TEXT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (partner_tenant_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_supply_sources_partner ON supply_sources (partner_tenant_id);

-- Every state change, append-only: who activated, suspended or retired a source, and when.
CREATE TABLE IF NOT EXISTS supply_events (
    id         BIGSERIAL PRIMARY KEY,
    source_id  UUID NOT NULL REFERENCES supply_sources(id),
    kind       TEXT NOT NULL CHECK (kind IN ('registered', 'activated', 'suspended', 'resumed', 'retired')),
    actor      TEXT NOT NULL,             -- tenant id, or 'service' for operations
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE OR REPLACE FUNCTION supply_events_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'supply_events is append-only (no % allowed)', TG_OP;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trg_supply_events_append_only ON supply_events;
CREATE TRIGGER trg_supply_events_append_only BEFORE UPDATE OR DELETE ON supply_events
    FOR EACH ROW EXECUTE FUNCTION supply_events_append_only();

-- One row per compute.usage.v1 event compute-control emits, keyed on usage_id (a replay is a no-op).
-- source_id is TEXT: 1Trade's own capacity has a configured id (e.g. dc-owned-1), not a UUID.
-- This is what payouts (F18) reconcile against, so it is append-only too.
CREATE TABLE IF NOT EXISTS supply_usage (
    usage_id    TEXT PRIMARY KEY,
    source_id   TEXT NOT NULL,
    tenant_id   TEXT NOT NULL,
    gpu_type    TEXT NOT NULL,
    gpu_seconds NUMERIC(20,6) NOT NULL CHECK (gpu_seconds >= 0),  -- as on compute.usage.v1
    units       NUMERIC(20,6) NOT NULL CHECK (units >= 0),
    is_paper    BOOLEAN NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_supply_usage_source ON supply_usage (source_id, created_at);
DROP TRIGGER IF EXISTS trg_supply_usage_append_only ON supply_usage;
CREATE TRIGGER trg_supply_usage_append_only BEFORE UPDATE OR DELETE ON supply_usage
    FOR EACH ROW EXECUTE FUNCTION supply_events_append_only();
