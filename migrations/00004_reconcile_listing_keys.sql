-- +goose Up
-- Staging table for nightly mirror key reconciliation (anti-join delete).
CREATE TABLE reconcile_listing_keys (
    run_id       UUID         NOT NULL,
    dataset_slug VARCHAR(64)  NOT NULL,
    listing_key  VARCHAR(255) NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (run_id, dataset_slug, listing_key)
);

CREATE INDEX idx_reconcile_listing_keys_run ON reconcile_listing_keys (run_id);

-- +goose Down
DROP TABLE IF EXISTS reconcile_listing_keys;
