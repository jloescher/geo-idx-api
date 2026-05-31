-- +goose Up
CREATE TABLE IF NOT EXISTS mcp_keys (
    id                  BIGSERIAL PRIMARY KEY,
    name                TEXT NOT NULL,
    key_hash            TEXT NOT NULL UNIQUE,
    scopes              JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by_user_id  BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at        TIMESTAMPTZ,
    revoked_at          TIMESTAMPTZ,
    notes               TEXT
);

CREATE INDEX IF NOT EXISTS idx_mcp_keys_created_by ON mcp_keys(created_by_user_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mcp_keys_key_hash ON mcp_keys(key_hash) WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS mcp_keys;
