-- Conversations: server-side chat history for the inference playground. The transcript (turns + any
-- media URLs) is stored as JSONB; generated media itself lives in object storage (Spaces) and is
-- referenced by URL inside the transcript, so a conversation row stays small. Tenant-scoped and
-- cascade-deleted with the tenant.
CREATE TABLE IF NOT EXISTS conversations (
    id          UUID PRIMARY KEY,
    tenant_id   UUID NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    user_id     UUID,
    title       TEXT NOT NULL DEFAULT 'New chat',
    model       TEXT NOT NULL DEFAULT '',
    transcript  JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- List newest-first within a tenant (drives the conversation sidebar).
CREATE INDEX IF NOT EXISTS idx_conversations_tenant ON conversations (tenant_id, updated_at DESC);
