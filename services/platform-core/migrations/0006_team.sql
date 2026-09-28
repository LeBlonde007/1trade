-- 0006_team.sql — team management (platform-core.yaml v1.x, F02/F03 M4): sub-accounts (team budgets
-- with their own ledger balances), invitations, and membership changes. Rollback: forward-only in prod.

-- A sub-account is a team inside a tenant. Its members' usage is billed to its own ledger balances,
-- which the tenant funds from its main balance (credit.yaml v1.4 transfer).
CREATE TABLE IF NOT EXISTS sub_accounts (
    id         UUID PRIMARY KEY,
    tenant_id  UUID NOT NULL REFERENCES tenants(id),
    name       TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 80),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

-- The sub-account a member works in (null = the tenant's main balance). Carried into the JWT.
ALTER TABLE users ADD COLUMN IF NOT EXISTS sub_account_id UUID REFERENCES sub_accounts(id);

-- Invitations: the raw token goes out by email; only sha256(token) is stored. One pending invite per
-- (tenant, email). Accepting creates the user in the inviting tenant with the invited roles.
CREATE TABLE IF NOT EXISTS invites (
    id             UUID PRIMARY KEY,
    tenant_id      UUID NOT NULL REFERENCES tenants(id),
    email          TEXT NOT NULL,
    roles          TEXT[] NOT NULL,
    sub_account_id UUID REFERENCES sub_accounts(id),
    token_hash     TEXT NOT NULL UNIQUE,
    invited_by     UUID NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at     TIMESTAMPTZ NOT NULL,
    accepted_at    TIMESTAMPTZ,
    revoked_at     TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_invites_pending ON invites (tenant_id, email)
    WHERE accepted_at IS NULL AND revoked_at IS NULL;
