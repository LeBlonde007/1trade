-- 0002_payouts.sql — partner payouts (supply.yaml v1.1, F18). Money is NUMERIC(20,6), never float.
-- Every usage record lands in at most one statement (payout_usage.usage_id is the primary key), so
-- payouts reconcile to usage by construction. Rollback: forward-only in prod.

CREATE TABLE IF NOT EXISTS partner_agreements (
    partner_tenant_id UUID PRIMARY KEY,
    rates             JSONB NOT NULL,                 -- {"gpu_h100": "1.800000", ...} USD per GPU-hour
    fee_percent       NUMERIC(9,6) NOT NULL CHECK (fee_percent BETWEEN 0 AND 100),
    holdback_percent  NUMERIC(9,6) NOT NULL CHECK (holdback_percent BETWEEN 0 AND 100),
    dispute_days      INT NOT NULL CHECK (dispute_days BETWEEN 1 AND 90),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS partner_payouts (
    id                UUID PRIMARY KEY,
    partner_tenant_id UUID NOT NULL,
    period_start      TIMESTAMPTZ NOT NULL,
    period_end        TIMESTAMPTZ NOT NULL CHECK (period_end > period_start),
    currency          TEXT NOT NULL DEFAULT 'USD' CHECK (currency = 'USD'),
    is_paper          BOOLEAN NOT NULL,
    gpu_seconds       NUMERIC(24,6) NOT NULL CHECK (gpu_seconds >= 0),
    gross             NUMERIC(20,6) NOT NULL CHECK (gross >= 0),
    fee               NUMERIC(20,6) NOT NULL CHECK (fee >= 0),
    payout            NUMERIC(20,6) NOT NULL CHECK (payout >= 0),
    holdback          NUMERIC(20,6) NOT NULL CHECK (holdback >= 0),
    released          NUMERIC(20,6) NOT NULL CHECK (released >= 0),
    usage_records     INT NOT NULL CHECK (usage_records > 0),
    state             TEXT NOT NULL CHECK (state IN ('pending', 'wired', 'disputed', 'settled')),
    wire_reference    TEXT,
    dispute_until     TIMESTAMPTZ NOT NULL,
    dispute_reason    TEXT,
    resolution        TEXT CHECK (resolution IN ('release', 'withhold')),
    resolution_note   TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (payout = gross - fee),
    CHECK (released = payout - holdback),
    CHECK (NOT is_paper OR wire_reference IS NULL)       -- a simulation never moves money
);
CREATE INDEX IF NOT EXISTS idx_partner_payouts_partner ON partner_payouts (partner_tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_partner_payouts_state ON partner_payouts (state);

CREATE TABLE IF NOT EXISTS partner_payout_lines (
    payout_id   UUID NOT NULL REFERENCES partner_payouts(id),
    gpu_type    TEXT NOT NULL,
    gpu_seconds NUMERIC(24,6) NOT NULL,
    rate        NUMERIC(20,6) NOT NULL,
    gross       NUMERIC(20,6) NOT NULL,
    PRIMARY KEY (payout_id, gpu_type)
);

-- Which usage each statement paid for. The primary key is the reconciliation guarantee.
CREATE TABLE IF NOT EXISTS payout_usage (
    usage_id  TEXT PRIMARY KEY REFERENCES supply_usage(usage_id),
    payout_id UUID NOT NULL REFERENCES partner_payouts(id)
);
CREATE INDEX IF NOT EXISTS idx_payout_usage_payout ON payout_usage (payout_id);

-- Every payout state change, append-only.
CREATE TABLE IF NOT EXISTS payout_events (
    id         BIGSERIAL PRIMARY KEY,
    payout_id  UUID NOT NULL REFERENCES partner_payouts(id),
    kind       TEXT NOT NULL CHECK (kind IN ('created', 'wired', 'disputed', 'settled', 'resolved')),
    actor      TEXT NOT NULL,
    note       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
DROP TRIGGER IF EXISTS trg_payout_events_append_only ON payout_events;
CREATE TRIGGER trg_payout_events_append_only BEFORE UPDATE OR DELETE ON payout_events
    FOR EACH ROW EXECUTE FUNCTION supply_events_append_only();
DROP TRIGGER IF EXISTS trg_payout_usage_append_only ON payout_usage;
CREATE TRIGGER trg_payout_usage_append_only BEFORE UPDATE OR DELETE ON payout_usage
    FOR EACH ROW EXECUTE FUNCTION supply_events_append_only();
