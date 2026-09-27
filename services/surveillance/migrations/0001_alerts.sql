-- 0001_alerts.sql — surveillance alerts (KW05). One row per surveillance.alert.v1 alert.
-- alert_id is deterministic, so inserting is idempotent: replaying the engine stream after a restart
-- re-derives the same alerts and ON CONFLICT DO NOTHING keeps exactly one row each.
-- Append-only: an alert is evidence. Review outcomes go in their own table (a later migration), so
-- the original detection can never be edited away.

CREATE TABLE IF NOT EXISTS surveillance_alerts (
    alert_id               TEXT PRIMARY KEY,
    rule                   TEXT NOT NULL CHECK (rule IN ('wash_trade','spoofing','layering','marking_the_close','cross_product','excessive_cancellation','position_limit')),
    severity               TEXT NOT NULL CHECK (severity IN ('low','medium','high')),
    action                 TEXT NOT NULL CHECK (action IN ('review','rate_limit','exclude_from_index','suspend_recommended')),
    tenant_id              TEXT NOT NULL,
    counterparty_tenant_id TEXT,
    product_id             TEXT NOT NULL,
    related_product_id     TEXT,
    is_paper               BOOLEAN NOT NULL,
    window_start           TIMESTAMPTZ NOT NULL,
    window_end             TIMESTAMPTZ NOT NULL,
    detected_at            TIMESTAMPTZ NOT NULL,
    evidence               JSONB NOT NULL,
    trade_ids              TEXT[] NOT NULL,
    order_ids              TEXT[] NOT NULL,
    stored_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_surv_alerts_detected ON surveillance_alerts (detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_surv_alerts_tenant   ON surveillance_alerts (tenant_id, detected_at DESC);

CREATE OR REPLACE FUNCTION surveillance_alerts_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'surveillance_alerts is append-only (no % allowed)', TG_OP;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_surv_alerts_append_only ON surveillance_alerts;
CREATE TRIGGER trg_surv_alerts_append_only
    BEFORE UPDATE OR DELETE ON surveillance_alerts
    FOR EACH ROW EXECUTE FUNCTION surveillance_alerts_append_only();
