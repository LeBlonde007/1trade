-- 0007_mfa.sql — two-factor authentication (TOTP, platform-core.yaml v1.7). Secrets are sealed with
-- AES-256-GCM before they reach the database. mfa_last_step makes every code single use. Five wrong
-- codes lock the second step for 15 minutes. Rollback: forward-only in prod.
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_enabled        BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_secret         BYTEA;   -- sealed; set while enabled
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_pending_secret BYTEA;   -- sealed; set up but not yet confirmed
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_last_step      BIGINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_failures       INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_locked_until   TIMESTAMPTZ;

-- Single-use recovery codes, stored as sha256 only.
CREATE TABLE IF NOT EXISTS mfa_recovery_codes (
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used_at   TIMESTAMPTZ,
    PRIMARY KEY (user_id, code_hash)
);
