-- 0005_kyc.sql — F22 KYC/AML gate (M3). KYC is per-tenant: a tenant verifies once, and the status
-- gates real-money (is_paper=false) credit purchases. Sandbox (is_paper=true) needs no KYC — paper
-- credits carry no real-money/AML exposure. PII is kept minimal (legal name · country · entity type);
-- documents/biometrics are handled by the third-party IDV vendor, not stored here.
--
-- Status machine: unverified → pending → verified | rejected (rejected may resubmit → pending).
-- 'verified' is the only status that permits a real-money purchase (domain.CanPurchaseRealMoney).

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS kyc_status TEXT NOT NULL DEFAULT 'unverified'
    CHECK (kyc_status IN ('unverified','pending','verified','rejected'));
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS kyc_legal_name   TEXT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS kyc_country      TEXT;   -- ISO 3166-1 alpha-2
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS kyc_entity_type  TEXT;   -- individual | business
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS kyc_submitted_at TIMESTAMPTZ;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS kyc_reviewed_at  TIMESTAMPTZ;
