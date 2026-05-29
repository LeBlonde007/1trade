-- 0001_init.sql — credit-ledger schema (F05). Append-only, hash-chained, fixed-point.
-- Shared types (credit_type, conventions) live in docs/contracts/schemas/types.sql.
-- Rollback: drop the two tables (forward-only in prod — corrections are compensating entries).

-- Per-(tenant, sub_account, credit_type, is_paper) balance. locked_amount is reserved for Phase 2
-- resting orders (0 in Phase 1). Paper and real never share a row (is_paper in the unique key).
CREATE TABLE IF NOT EXISTS credit_balances (
    balance_id      UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    sub_account_id  UUID,
    credit_type     TEXT NOT NULL REFERENCES credit_types(code),
    balance         NUMERIC(20,6) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    locked_amount   NUMERIC(20,6) NOT NULL DEFAULT 0 CHECK (locked_amount >= 0),
    is_paper        BOOLEAN NOT NULL,
    last_chain_hash TEXT NOT NULL DEFAULT '',   -- tip of this balance's per-balance hash chain
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- NULLS NOT DISTINCT (PG15+): a NULL sub_account_id still dedupes. Without it Postgres treats
    -- NULLs as distinct, so every no-sub-account movement would create a NEW balance row.
    UNIQUE NULLS NOT DISTINCT (tenant_id, sub_account_id, credit_type, is_paper)
);

-- Append-only transaction log. amount is the SIGNED delta. chain_hash extends the audit chain.
-- (operation, idempotency_key) is unique so a retried mutating call never double-applies.
CREATE TABLE IF NOT EXISTS credit_transactions (
    tx_id           UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    sub_account_id  UUID,
    credit_type     TEXT NOT NULL REFERENCES credit_types(code),
    operation       TEXT NOT NULL CHECK (operation IN
                       ('purchase','consumption','conversion','mint','burn','refund','trade')),
    amount          NUMERIC(20,6) NOT NULL,
    reference_id    TEXT,
    idempotency_key TEXT,
    balance_before  NUMERIC(20,6) NOT NULL,
    balance_after   NUMERIC(20,6) NOT NULL,
    is_paper        BOOLEAN NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    chain_hash      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_credit_tx_tenant_time ON credit_transactions (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_credit_tx_ref         ON credit_transactions (reference_id);
-- Idempotency is PER TENANT: a tenant's key must not collide with another tenant's. (A global
-- (operation, key) unique would let tenant B receive tenant A's transaction on a shared key.)
CREATE UNIQUE INDEX IF NOT EXISTS uq_credit_tx_idem  ON credit_transactions (tenant_id, operation, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- Append-only guard: block UPDATE/DELETE on the transaction log at the DB layer.
CREATE OR REPLACE FUNCTION credit_tx_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'credit_transactions is append-only (no % allowed)', TG_OP;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_credit_tx_append_only ON credit_transactions;
CREATE TRIGGER trg_credit_tx_append_only
    BEFORE UPDATE OR DELETE ON credit_transactions
    FOR EACH ROW EXECUTE FUNCTION credit_tx_append_only();
