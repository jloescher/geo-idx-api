-- +goose Up
-- OAuth auto-provisioned MCP keys and refresh tokens.

ALTER TABLE mcp_keys
    ADD COLUMN IF NOT EXISTS oauth_client_id VARCHAR(128) NULL
        REFERENCES oauth_clients(client_id) ON DELETE SET NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mcp_keys_oauth_user_client
    ON mcp_keys (created_by_user_id, oauth_client_id)
    WHERE revoked_at IS NULL AND oauth_client_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS oauth_refresh_tokens (
    token_hash           VARCHAR(128) PRIMARY KEY,
    client_id            VARCHAR(128) NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    user_id              BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scope                TEXT NOT NULL,
    granted_mcp_key_ids  JSONB NULL,
    expires_at           TIMESTAMPTZ NOT NULL,
    revoked_at           TIMESTAMPTZ NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_client_user
    ON oauth_refresh_tokens (client_id, user_id)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_expires
    ON oauth_refresh_tokens (expires_at);

-- +goose Down
DROP TABLE IF EXISTS oauth_refresh_tokens;
DROP INDEX IF EXISTS idx_mcp_keys_oauth_user_client;
ALTER TABLE mcp_keys DROP COLUMN IF EXISTS oauth_client_id;
