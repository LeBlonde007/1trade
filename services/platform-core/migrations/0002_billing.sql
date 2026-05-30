-- 0002_billing.sql — credit purchases (F06). A purchase is an order created at checkout; it becomes
-- 'paid' when Stripe settles, and the webhook then books the credits to the ledger. The mint is
-- idempotent on stripe_event_id (a webhook replay must not double-credit). Money is fixed-point.

CREATE TABLE IF NOT EXISTS purchases (
    id                UUID PRIMARY KEY,
    tenant_id         UUID NOT NULL REFERENCES tenants(id),
    amount            NUMERIC(20,6) NOT NULL CHECK (amount > 0),  -- credits to buy
    credit_type       TEXT NOT NULL,
    currency          TEXT NOT NULL CHECK (currency IN ('usd','jpy')),
    status            TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','paid','failed')),
    stripe_session_id TEXT UNIQUE,                 -- checkout session; links the webhook back to the order
    stripe_event_id   TEXT UNIQUE,                 -- the settling event; idempotency anchor for the mint
    is_paper          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    paid_at           TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_purchases_tenant ON purchases (tenant_id, created_at DESC);
