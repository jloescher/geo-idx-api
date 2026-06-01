-- +goose Up
-- MCP hardening: rate limit audit + optional domain binding on keys.

CREATE TABLE IF NOT EXISTS mcp_tool_usage (
    id               BIGSERIAL PRIMARY KEY,
    mcp_key_id       BIGINT REFERENCES mcp_keys(id) ON DELETE SET NULL,
    oauth_client_id  VARCHAR(128),
    tool_name        TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mcp_tool_usage_key_time ON mcp_tool_usage (mcp_key_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mcp_tool_usage_client_time ON mcp_tool_usage (oauth_client_id, created_at DESC);

ALTER TABLE mcp_keys ADD COLUMN IF NOT EXISTS domain_id BIGINT REFERENCES domains(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE mcp_keys DROP COLUMN IF EXISTS domain_id;
DROP TABLE IF EXISTS mcp_tool_usage;
