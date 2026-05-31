-- +goose Up
-- Minimal OAuth 2.1 tables to support Custom MCP connectors (e.g. Grok Web)
-- while keeping the existing direct mcp_key bearer token path fully intact.

CREATE TABLE oauth_clients (
    id BIGSERIAL PRIMARY KEY,
    client_id VARCHAR(128) NOT NULL UNIQUE,
    client_secret_hash VARCHAR(255) NULL,           -- NULL for public clients (PKCE)
    name VARCHAR(255) NOT NULL,
    redirect_uris JSONB NOT NULL,                   -- array of allowed redirect URIs
    created_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    is_trusted BOOLEAN NOT NULL DEFAULT FALSE,      -- trusted clients skip consent (optional)
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE oauth_authorization_codes (
    code VARCHAR(128) PRIMARY KEY,
    client_id VARCHAR(128) NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    scope TEXT NOT NULL,                            -- space-separated scopes or mcp_key_ids
    code_challenge VARCHAR(128) NULL,
    code_challenge_method VARCHAR(10) NULL,
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE oauth_access_tokens (
    token_hash VARCHAR(128) PRIMARY KEY,            -- store hash of the token, not the token itself
    client_id VARCHAR(128) NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scope TEXT NOT NULL,
    granted_mcp_key_ids JSONB NULL,                 -- array of mcp_key ids this token grants access to
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth_access_tokens_user_id ON oauth_access_tokens(user_id);
CREATE INDEX idx_oauth_access_tokens_expires_at ON oauth_access_tokens(expires_at);

-- Optional lightweight consent audit table (can be expanded later)
CREATE TABLE oauth_consents (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    client_id VARCHAR(128) NOT NULL REFERENCES oauth_clients(client_id) ON DELETE CASCADE,
    scope TEXT NOT NULL,
    granted_mcp_key_ids JSONB NULL,
    consented_at TIMESTAMP NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMP NULL
);

CREATE INDEX idx_oauth_consents_user_client ON oauth_consents(user_id, client_id);

-- +goose Down
DROP TABLE IF EXISTS oauth_consents;
DROP TABLE IF EXISTS oauth_access_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS oauth_clients;