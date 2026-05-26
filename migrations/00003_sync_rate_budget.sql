-- +goose Up
CREATE TABLE sync_rate_budget (
    provider TEXT PRIMARY KEY,
    next_allowed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    window_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    window_count INT NOT NULL DEFAULT 0
);

INSERT INTO sync_rate_budget (provider) VALUES ('bridge'), ('spark');

-- +goose Down
DROP TABLE IF EXISTS sync_rate_budget;
