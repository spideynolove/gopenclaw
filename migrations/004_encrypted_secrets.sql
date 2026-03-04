ALTER TABLE tenants ADD COLUMN IF NOT EXISTS encrypted_api_key TEXT;

ALTER TABLE agents ADD COLUMN IF NOT EXISTS encrypted_api_key TEXT;

CREATE INDEX IF NOT EXISTS idx_tenants_encrypted_key ON tenants(encrypted_api_key) WHERE encrypted_api_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_agents_encrypted_key ON agents(encrypted_api_key) WHERE encrypted_api_key IS NOT NULL;
