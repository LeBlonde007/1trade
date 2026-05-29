-- 0001_init.sql — platform-core schema (F02 auth / F03 accounts). Tenants are the billing unit;
-- a tenant is an individual or an org; users + api_keys belong to a tenant. Shared conventions in
-- docs/contracts/schemas/types.sql. Rollback: drop the four tables (no prod data yet).

CREATE TABLE IF NOT EXISTS tenants (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    kind        TEXT NOT NULL CHECK (kind IN ('individual','org')),
    is_paper    BOOLEAN NOT NULL DEFAULT TRUE,   -- carried into the JWT; paper/real never mix
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS orgs (
    id          UUID PRIMARY KEY,
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY,
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    org_id        UUID REFERENCES orgs(id),
    email         TEXT NOT NULL UNIQUE,           -- login identity; one account per email
    password_hash TEXT NOT NULL,                  -- bcrypt; never plaintext
    roles         TEXT[] NOT NULL DEFAULT '{}',   -- RBAC roles (admin/billing/engineer/viewer/trader)
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users (tenant_id);

CREATE TABLE IF NOT EXISTS api_keys (
    id           UUID PRIMARY KEY,
    tenant_id    UUID NOT NULL REFERENCES tenants(id),
    name         TEXT NOT NULL,
    prefix       TEXT NOT NULL,                   -- shown in listings
    hash         TEXT NOT NULL UNIQUE,            -- sha256(secret); the secret itself is never stored
    scopes       TEXT[] NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ                      -- non-null = revoked
);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys (tenant_id);
