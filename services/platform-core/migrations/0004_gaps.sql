-- 0004_gaps.sql — platform gap-closers: email verification + per-tenant billing budgets.

-- Email verification: a sha256(token) is stored at signup and cleared once verified. The raw token
-- is delivered out-of-band (email in prod; returned in dev). email_verified gates real-money flows.
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS verify_token TEXT;
CREATE INDEX IF NOT EXISTS idx_users_verify_token ON users (verify_token);

-- Billing budget: one monthly credit budget per tenant. Consumption alerts (50/80/100%) compare
-- month-to-date usage against this limit; auto-stop policy is M3.
CREATE TABLE IF NOT EXISTS budgets (
    tenant_id     UUID PRIMARY KEY REFERENCES tenants(id),
    credit_type   TEXT NOT NULL,
    monthly_limit NUMERIC(20,6) NOT NULL CHECK (monthly_limit >= 0),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
