-- 0008_sso.sql — SAML single sign-on (platform-core.yaml v1.8). One IdP per tenant; each email
-- domain routes to at most one tenant; every AuthnRequest id is recorded and consumed once, so an
-- assertion can only answer a request we made (and only once). Rollback: forward-only in prod.
CREATE TABLE IF NOT EXISTS sso_configs (
    tenant_id     UUID PRIMARY KEY REFERENCES tenants(id),
    idp_metadata  TEXT NOT NULL,                 -- the IdP's SAML metadata XML (signing certificate, SSO URL)
    idp_entity_id TEXT NOT NULL,
    default_role  TEXT NOT NULL DEFAULT 'viewer',
    jit           BOOLEAN NOT NULL DEFAULT TRUE,  -- create members on first SSO sign-in
    enforce       BOOLEAN NOT NULL DEFAULT FALSE, -- password sign-in refused for non-admins
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sso_domains (
    domain    TEXT PRIMARY KEY CHECK (domain = lower(domain)),
    tenant_id UUID NOT NULL REFERENCES tenants(id)
);

CREATE TABLE IF NOT EXISTS sso_requests (
    id         TEXT PRIMARY KEY,
    tenant_id  UUID NOT NULL REFERENCES tenants(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    used_at    TIMESTAMPTZ
);
