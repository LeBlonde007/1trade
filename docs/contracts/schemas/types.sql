-- Exascale — shared SQL types (v1.0). Owned by tech-lead. Consumed by every service.
--
-- These are the types/conventions that MUST be identical across services so two services can't
-- disagree on a shared shape. Per-service tables live in services/<name>/migrations/ — NOT here.
-- See docs/plans/CONTRACTS.md and docs/contracts/credit-types.md.
--
-- Rules encoded here:
--   * money/credit amounts are NUMERIC(20,6) — fixed-point, never float (ENGINEERING_STANDARDS §1).
--   * every tenant-scoped, money/credit, or order row carries is_paper.
--   * credit_type values come from the credit_types lookup table below (mirrors credit-types.md).

-- ============================================================================================
-- Shared domains / conventions (documented; applied per-service in each service's migrations)
-- ============================================================================================
-- tenant_id        UUID NOT NULL              -- the billing unit (platform-core owns the entity)
-- sub_account_id   UUID NULL                  -- optional team sub-account within an org
-- is_paper         BOOLEAN NOT NULL           -- paper vs real money; NEVER mixed. default true in dev/staging
-- amount/balance   NUMERIC(20,6) NOT NULL     -- fixed-point; serialized as decimal strings in JSON
-- created_at       TIMESTAMPTZ NOT NULL DEFAULT now()

-- A reusable fixed-point money/credit domain. Services MAY use this domain for clarity.
-- credit_amount is a NUMERIC(20,6); negative values are allowed only where a column explicitly
-- represents a signed delta (e.g. a transaction amount), never for a balance.
DO $$ BEGIN
  CREATE DOMAIN credit_amount AS NUMERIC(20,6);
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- ============================================================================================
-- credit_types — the canonical enum as data (mirrors docs/contracts/credit-types.md §1).
-- Services FK their credit_type columns to this table instead of using a hard-to-alter PG enum,
-- so adding a tier later is an INSERT (a tech-lead-approved migration), not a type rewrite.
-- ============================================================================================
CREATE TABLE IF NOT EXISTS credit_types (
    code            TEXT PRIMARY KEY,                 -- e.g. 'ai_index', 'text', 'gpu_h100'
    kind            TEXT NOT NULL CHECK (kind IN ('index','sub','gpu')),
    unit            TEXT NOT NULL,                    -- human-readable unit, e.g. 'H100-80GB-GPU-hour'
    tradeable       BOOLEAN NOT NULL DEFAULT FALSE,   -- Phase 2 only; dormant in Phase 1
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed the v1 enum. Idempotent: re-running leaves rows unchanged.
INSERT INTO credit_types (code, kind, unit, tradeable) VALUES
    ('ai_index',   'index', 'AI credit',            TRUE),
    ('text',       'sub',   'text credit',          TRUE),
    ('speech',     'sub',   'speech credit',        TRUE),
    ('image',      'sub',   'image credit',         TRUE),
    ('video',      'sub',   'video credit',         TRUE),
    ('embeddings', 'sub',   'embeddings credit',    TRUE),
    ('gpu_h100',   'gpu',   'H100-80GB-GPU-hour',   TRUE),
    ('gpu_h200',   'gpu',   'H200-GPU-hour',        TRUE)
ON CONFLICT (code) DO NOTHING;

-- ============================================================================================
-- set_updated_at() — shared trigger to maintain an updated_at column. Services attach it to
-- tables that have an updated_at. (Append-only tables like credit_transactions do NOT use it.)
-- ============================================================================================
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================================
-- Hash-chain convention (documented; implemented in Go, not SQL).
-- For append-only audited tables (credit_transactions, index_prints, trades):
--   chain_hash = sha256( prev_chain_hash || canonical_json(row_without_chain_hash) )
-- canonical_json = keys sorted, no insignificant whitespace, decimals as fixed-point strings.
-- The balance update and the transaction insert MUST commit in one DB transaction.
-- ============================================================================================
