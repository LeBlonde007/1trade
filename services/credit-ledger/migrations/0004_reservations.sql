-- 0004_reservations.sql — order reservations (credit.yaml v1.2.0, KW03 SPEC §8).
-- The engine reserves what an order could spend when it accepts it; settlement consumes it; the engine
-- releases the rest when the order closes. A reservation is reflected in the balance's locked_amount,
-- and from here on no debit anywhere may take a balance below its locked_amount.
-- Rollback: forward-only in prod.

CREATE TABLE IF NOT EXISTS reservations (
    order_id        UUID PRIMARY KEY,                 -- one reservation per order (idempotency anchor)
    tenant_id       UUID NOT NULL,
    sub_account_id  UUID,
    asset_kind      TEXT NOT NULL CHECK (asset_kind IN ('credit', 'cash')),
    asset           TEXT NOT NULL,                    -- a credit_type, or a currency
    amount          NUMERIC(20,6) NOT NULL CHECK (amount > 0),
    remaining       NUMERIC(20,6) NOT NULL CHECK (remaining >= 0 AND remaining <= amount),
    state           TEXT NOT NULL CHECK (state IN ('open', 'released')),
    is_paper        BOOLEAN NOT NULL CHECK (is_paper),  -- paper only until counsel clears real money
    request_hash    TEXT NOT NULL,                    -- 409 when an order_id is re-reserved differently
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (state = 'open' OR remaining = 0)
);
CREATE INDEX IF NOT EXISTS idx_reservations_open ON reservations (tenant_id, asset_kind, asset) WHERE state = 'open';

-- Every change to a reservation, append-only: the audit trail for locked value (reservations itself
-- is mutable state; this is its history).
CREATE TABLE IF NOT EXISTS reservation_events (
    event_id    UUID PRIMARY KEY,
    order_id    UUID NOT NULL REFERENCES reservations(order_id),
    kind        TEXT NOT NULL CHECK (kind IN ('reserve', 'consume', 'release')),
    amount      NUMERIC(20,6) NOT NULL CHECK (amount >= 0),
    remaining   NUMERIC(20,6) NOT NULL CHECK (remaining >= 0),
    reference   TEXT,                                 -- trade_id for consume
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_reservation_events_order ON reservation_events (order_id, created_at);

DROP TRIGGER IF EXISTS trg_reservation_events_append_only ON reservation_events;
CREATE TRIGGER trg_reservation_events_append_only
    BEFORE UPDATE OR DELETE ON reservation_events
    FOR EACH ROW EXECUTE FUNCTION ledger_append_only();
DROP TRIGGER IF EXISTS trg_reservation_events_no_truncate ON reservation_events;
CREATE TRIGGER trg_reservation_events_no_truncate
    BEFORE TRUNCATE ON reservation_events
    FOR EACH STATEMENT EXECUTE FUNCTION ledger_append_only();

-- The backstop for "reserved value cannot be spent elsewhere": whatever the application does, a
-- balance can never fall below what is locked against it.
DO $$ BEGIN
    ALTER TABLE credit_balances ADD CONSTRAINT credit_balance_covers_locked CHECK (balance >= locked_amount);
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
    ALTER TABLE cash_balances ADD CONSTRAINT cash_balance_covers_locked CHECK (balance >= locked_amount);
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
