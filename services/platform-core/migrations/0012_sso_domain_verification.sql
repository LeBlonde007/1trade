-- 0012_sso_domain_verification.sql — an email domain routes sign-ins to a tenant only once that tenant
-- has proved it controls the domain (a DNS TXT record). Until then any number of tenants may hold a
-- pending claim, so claiming a domain first can neither hijack nor block its real owner.
-- Rollback: forward-only in prod.
ALTER TABLE sso_domains ADD COLUMN IF NOT EXISTS verify_token TEXT;
ALTER TABLE sso_domains ADD COLUMN IF NOT EXISTS verified_at  TIMESTAMPTZ;
UPDATE sso_domains SET verify_token = md5(random()::text || domain) WHERE verify_token IS NULL;
ALTER TABLE sso_domains ALTER COLUMN verify_token SET NOT NULL;
ALTER TABLE sso_domains DROP CONSTRAINT IF EXISTS sso_domains_pkey;
DO $$ BEGIN
    ALTER TABLE sso_domains ADD PRIMARY KEY (tenant_id, domain);
EXCEPTION WHEN invalid_table_definition THEN NULL;
END $$;
-- A domain routes to at most one tenant: the one that verified it.
CREATE UNIQUE INDEX IF NOT EXISTS uq_sso_domains_verified ON sso_domains (domain) WHERE verified_at IS NOT NULL;
