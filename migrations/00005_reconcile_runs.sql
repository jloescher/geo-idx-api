-- +goose Up
-- Durable audit log for mirror key reconciliation runs.
CREATE TABLE reconcile_runs (
    run_id              UUID         PRIMARY KEY,
    dataset_slug        VARCHAR(64)  NOT NULL,
    provider            VARCHAR(32)  NOT NULL,
    status              VARCHAR(32)  NOT NULL DEFAULT 'running',
    keys_seen           BIGINT       NOT NULL DEFAULT 0,
    rows_deleted        BIGINT       NOT NULL DEFAULT 0,
    mirror_count_before BIGINT,
    error_message       TEXT,
    started_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    finished_at         TIMESTAMPTZ
);

CREATE INDEX idx_reconcile_runs_dataset_finished ON reconcile_runs (dataset_slug, finished_at DESC);

-- +goose Down
DROP TABLE IF EXISTS reconcile_runs;
