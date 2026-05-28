-- +goose Up
ALTER TABLE listings
    ADD COLUMN IF NOT EXISTS geocode_query TEXT,
    ADD COLUMN IF NOT EXISTS geocoded_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS geocode_attempted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS geocode_failed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS geocode_failure_reason TEXT,
    ADD COLUMN IF NOT EXISTS geocode_bad_address_yn BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS geocode_attempt_count INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS listings_geocode_bad_address_idx
    ON listings (dataset_slug, geocode_bad_address_yn, geocode_failed_at DESC)
    WHERE geocode_bad_address_yn IS TRUE;

CREATE INDEX IF NOT EXISTS listings_geocode_retry_queue_idx
    ON listings (dataset_slug, id)
    WHERE geocode_bad_address_yn IS FALSE
      AND (latitude IS NULL OR longitude IS NULL)
      AND internet_address_display_yn IS TRUE
      AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending');

-- +goose Down
DROP INDEX IF EXISTS listings_geocode_retry_queue_idx;
DROP INDEX IF EXISTS listings_geocode_bad_address_idx;

ALTER TABLE listings
    DROP COLUMN IF EXISTS geocode_attempt_count,
    DROP COLUMN IF EXISTS geocode_bad_address_yn,
    DROP COLUMN IF EXISTS geocode_failure_reason,
    DROP COLUMN IF EXISTS geocode_failed_at,
    DROP COLUMN IF EXISTS geocode_attempted_at,
    DROP COLUMN IF EXISTS geocoded_at,
    DROP COLUMN IF EXISTS geocode_query;
