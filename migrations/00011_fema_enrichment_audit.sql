-- +goose Up
ALTER TABLE listings
    ADD COLUMN IF NOT EXISTS fema_attempted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS fema_failed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS fema_failure_reason TEXT,
    ADD COLUMN IF NOT EXISTS fema_attempt_count INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS listings_fema_null_coords_idx
    ON listings (dataset_slug, fema_failure_reason)
    WHERE latitude IS NOT NULL
      AND longitude IS NOT NULL
      AND fema_flood_zone_code IS NULL
      AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending');

-- +goose Down
DROP INDEX IF EXISTS listings_fema_null_coords_idx;

ALTER TABLE listings
    DROP COLUMN IF EXISTS fema_attempt_count,
    DROP COLUMN IF EXISTS fema_failure_reason,
    DROP COLUMN IF EXISTS fema_failed_at,
    DROP COLUMN IF EXISTS fema_attempted_at;
