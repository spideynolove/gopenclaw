CREATE TABLE IF NOT EXISTS tenants (
    id       TEXT PRIMARY KEY,
    api_key  TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS agents (
    id            TEXT PRIMARY KEY,
    tenant_id     TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    system_prompt TEXT NOT NULL,
    model         TEXT NOT NULL DEFAULT 'gpt-4',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE sessions ADD COLUMN IF NOT EXISTS tenant_id TEXT;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS agent_id TEXT;
ALTER TABLE sessions ADD CONSTRAINT IF NOT EXISTS fk_sessions_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
ALTER TABLE sessions ADD CONSTRAINT IF NOT EXISTS fk_sessions_agent
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_sessions_tenant_id ON sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_agents_tenant_id ON agents(tenant_id);

ALTER TABLE memories ADD COLUMN IF NOT EXISTS tenant_id TEXT;
ALTER TABLE memories ADD CONSTRAINT IF NOT EXISTS fk_memories_tenant
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_memories_tenant_id ON memories(tenant_id);
