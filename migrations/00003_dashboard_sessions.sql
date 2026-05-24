-- +goose Up
CREATE TABLE IF NOT EXISTS dashboard_sessions (
    id VARCHAR(128) PRIMARY KEY,
    payload BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS dashboard_sessions_expires_at_idx ON dashboard_sessions(expires_at);

-- +goose Down
DROP TABLE IF EXISTS dashboard_sessions;
