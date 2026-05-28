-- Manual backfill helper for listings that should be excluded from repeated geocode retries.
-- Run on Patroni primary only after review.
--
-- Usage:
--   1) Fill candidate_ids with the listing IDs you want to flag.
--   2) Run the preview SELECT and verify affected rows.
--   3) Uncomment the UPDATE block to apply.

BEGIN;

CREATE TEMP TABLE candidate_ids (
    id BIGINT PRIMARY KEY
) ON COMMIT DROP;

-- Example:
-- INSERT INTO candidate_ids (id) VALUES
--   (123456),
--   (234567),
--   (345678);

-- Preview affected listings before updating.
SELECT
    l.id,
    l.dataset_slug,
    l.unparsed_address,
    l.city,
    l.state_or_province,
    l.postal_code,
    l.latitude,
    l.longitude,
    l.geocode_bad_address_yn,
    l.geocode_failure_reason
FROM listings l
JOIN candidate_ids c ON c.id = l.id
ORDER BY l.id;

-- Apply after review.
-- UPDATE listings AS l
-- SET
--     geocode_attempted_at = COALESCE(l.geocode_attempted_at, NOW()),
--     geocode_failed_at = NOW(),
--     geocode_failure_reason = 'insufficient_address',
--     geocode_bad_address_yn = TRUE,
--     geocode_attempt_count = CASE
--         WHEN COALESCE(l.geocode_attempt_count, 0) = 0 THEN 1
--         ELSE l.geocode_attempt_count
--     END,
--     updated_at = NOW()
-- FROM candidate_ids c
-- WHERE l.id = c.id
--   AND (l.latitude IS NULL OR l.longitude IS NULL);

ROLLBACK;
