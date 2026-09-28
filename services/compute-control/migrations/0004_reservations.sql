-- 0004_reservations.sql — reserved GPU capacity (compute.yaml v1.2, F14). A reservation sets aside
-- `gpus` GPUs of one tier for one tenant (paper or real) for its term, prepaid in that tier's GPU
-- credits at the term discount. Rollback: forward-only in prod.

CREATE TABLE IF NOT EXISTS compute_reservations (
    id              UUID PRIMARY KEY,
    tenant_id       TEXT NOT NULL,
    sub_account_id  TEXT,
    is_paper        BOOLEAN NOT NULL,
    gpu_type        TEXT NOT NULL CHECK (gpu_type IN ('gpu_h100', 'gpu_h200')),
    gpus            INT  NOT NULL CHECK (gpus BETWEEN 1 AND 256),
    term            TEXT NOT NULL CHECK (term IN ('1mo', '6mo', '12mo')),
    discount_pct    INT  NOT NULL CHECK (discount_pct BETWEEN 0 AND 100),
    gpu_hours       NUMERIC(20,6) NOT NULL CHECK (gpu_hours > 0),   -- gpus × term hours
    price           NUMERIC(20,6) NOT NULL CHECK (price > 0),       -- GPU credits debited
    state           TEXT NOT NULL CHECK (state IN ('pending_payment', 'active', 'expired', 'failed')),
    starts_at       TIMESTAMPTZ,
    ends_at         TIMESTAMPTZ,
    ledger_tx_id    TEXT,
    failure         TEXT,
    idempotency_key TEXT NOT NULL,
    request_hash    TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key),
    -- the price is exactly the undiscounted GPU-hours less the discount
    CHECK (price = round(gpu_hours * (100 - discount_pct) / 100, 6)),
    CHECK (state <> 'active' OR (ledger_tx_id IS NOT NULL AND starts_at IS NOT NULL AND ends_at > starts_at))
);
CREATE INDEX IF NOT EXISTS idx_compute_reservations_tenant ON compute_reservations (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_compute_reservations_live ON compute_reservations (state) WHERE state IN ('pending_payment', 'active');

-- Every state change, append-only.
CREATE TABLE IF NOT EXISTS compute_reservation_events (
    id             BIGSERIAL PRIMARY KEY,
    reservation_id UUID NOT NULL REFERENCES compute_reservations(id),
    kind           TEXT NOT NULL CHECK (kind IN ('requested', 'paid', 'failed', 'expired')),
    actor          TEXT NOT NULL,
    detail         TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
DROP TRIGGER IF EXISTS trg_compute_reservation_events_append_only ON compute_reservation_events;
CREATE TRIGGER trg_compute_reservation_events_append_only BEFORE UPDATE OR DELETE ON compute_reservation_events
    FOR EACH ROW EXECUTE FUNCTION supply_events_append_only();
