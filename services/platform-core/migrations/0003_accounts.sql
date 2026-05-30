-- 0003_accounts.sql — F03 accounts/orgs/RBAC (M1). The audit log: every sensitive action appends a
-- tenant-scoped row; before/after capture the change for diffs. The matching admin.action.v1 event
-- is the real-time fan-out of the same row. (Sub-account hierarchy + per-team budgets are M4.)

CREATE TABLE IF NOT EXISTS audit_log (
    id          UUID PRIMARY KEY,
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    actor_id    UUID,                          -- null = system / self-serve
    action      TEXT NOT NULL,                 -- dotted verb, e.g. apikey.create
    target_type TEXT,                          -- api_key | org | user | tenant
    target_id   TEXT,
    before      JSONB,                         -- prior state (null on create)
    after       JSONB,                         -- new state (null on delete)
    is_paper    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_tenant ON audit_log (tenant_id, created_at DESC);
