-- 0003_cash.sql — paper cash + trade settlement (credit.yaml v1.1.0, ADR-0004).
-- The quote currency (USD) is NOT a credit type (credit-types.md §6): it gets its own balance and
-- transaction tables, with the same append-only + per-balance hash-chain rules as credits.
-- PAPER ONLY until counsel clears real-money custody: is_paper is CHECKed true on every cash row, so
-- a real-money cash row cannot exist even if application code is wrong.
-- Rollback: forward-only in prod (corrections are compensating entries).

CREATE TABLE IF NOT EXISTS cash_balances (
    balance_id      UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    sub_account_id  UUID,
    currency        TEXT NOT NULL CHECK (currency IN ('USD')),
    balance         NUMERIC(20,6) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    locked_amount   NUMERIC(20,6) NOT NULL DEFAULT 0 CHECK (locked_amount >= 0),
    is_paper        BOOLEAN NOT NULL CHECK (is_paper),
    last_chain_hash TEXT NOT NULL DEFAULT '',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (tenant_id, sub_account_id, currency, is_paper)
);

CREATE TABLE IF NOT EXISTS cash_transactions (
    tx_id           UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    sub_account_id  UUID,
    currency        TEXT NOT NULL CHECK (currency IN ('USD')),
    operation       TEXT NOT NULL CHECK (operation IN ('paper_grant', 'trade', 'fee')),
    amount          NUMERIC(20,6) NOT NULL,
    reference_id    TEXT,
    idempotency_key TEXT,
    balance_before  NUMERIC(20,6) NOT NULL,
    balance_after   NUMERIC(20,6) NOT NULL,
    is_paper        BOOLEAN NOT NULL CHECK (is_paper),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    chain_hash      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cash_tx_tenant_time ON cash_transactions (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_cash_tx_ref         ON cash_transactions (reference_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_cash_tx_idem  ON cash_transactions (tenant_id, operation, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- One row per settled trade: the idempotency record for /v1/credits/settle-trade. request_hash
-- detects a replayed trade_id with a different body (409); result_json is the original response,
-- returned verbatim on replay.
CREATE TABLE IF NOT EXISTS trade_settlements (
    trade_id          UUID PRIMARY KEY,
    request_hash      TEXT NOT NULL,
    engine_chain_hash TEXT NOT NULL,
    notional          NUMERIC(20,6) NOT NULL CHECK (notional >= 0),
    result_json       TEXT NOT NULL,
    is_paper          BOOLEAN NOT NULL CHECK (is_paper),
    settled_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Append-only guards (UPDATE / DELETE / TRUNCATE) for both new logs.
CREATE OR REPLACE FUNCTION ledger_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION '% is append-only (no % allowed)', TG_TABLE_NAME, TG_OP;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_cash_tx_append_only ON cash_transactions;
CREATE TRIGGER trg_cash_tx_append_only
    BEFORE UPDATE OR DELETE ON cash_transactions
    FOR EACH ROW EXECUTE FUNCTION ledger_append_only();
DROP TRIGGER IF EXISTS trg_cash_tx_no_truncate ON cash_transactions;
CREATE TRIGGER trg_cash_tx_no_truncate
    BEFORE TRUNCATE ON cash_transactions
    FOR EACH STATEMENT EXECUTE FUNCTION ledger_append_only();

DROP TRIGGER IF EXISTS trg_trade_settlements_append_only ON trade_settlements;
CREATE TRIGGER trg_trade_settlements_append_only
    BEFORE UPDATE OR DELETE ON trade_settlements
    FOR EACH ROW EXECUTE FUNCTION ledger_append_only();
DROP TRIGGER IF EXISTS trg_trade_settlements_no_truncate ON trade_settlements;
CREATE TRIGGER trg_trade_settlements_no_truncate
    BEFORE TRUNCATE ON trade_settlements
    FOR EACH STATEMENT EXECUTE FUNCTION ledger_append_only();
