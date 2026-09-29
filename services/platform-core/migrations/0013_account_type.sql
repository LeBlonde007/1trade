-- 0013_account_type.sql — what the account signed up as (trader | ai_company | enterprise | datacenter).
-- It chooses the product surface the web app shows (the sidebar and the landing page); it grants no
-- permission — roles and is_paper still decide what an account may do. Existing tenants default to
-- ai_company, the platform-first product. Rollback: forward-only in prod.
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS account_type TEXT NOT NULL DEFAULT 'ai_company';
DO $$ BEGIN
    ALTER TABLE tenants ADD CONSTRAINT tenants_account_type_check
        CHECK (account_type IN ('trader', 'ai_company', 'enterprise', 'datacenter'));
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
