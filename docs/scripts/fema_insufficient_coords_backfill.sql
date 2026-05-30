-- Backfill listings with bad coordinates that were incorrectly marked no_nfhl_feature.
-- Run on Patroni primary only after review (see docs/production-data-backfill.md).
--
-- Suspicious coordinate rules mirror internal/service/mls/coords_suspicious.go:
--   lat < -60 (Antarctica)
--   lat = 0 AND lng = 0 (Null Island)
--   state FL and (lat outside 24.5–31.2 OR lng outside -87.8–-79.8)
--   state FL and abs(lat) > 50 and abs(lng) < 45 (swapped lat/lng)
--
-- Usage:
--   1) Run the preview SELECT and verify affected rows.
--   2) Uncomment the UPDATE block to reset rows for geocode recovery.
--   3) POST /api/v1/admin/geocode/kickoff then POST /api/v1/admin/flood-enrich

BEGIN;

-- Preview suspicious no_nfhl_feature rows eligible for recovery.
SELECT
    l.id,
    l.dataset_slug,
    l.listing_key,
    l.unparsed_address,
    l.city,
    l.state_or_province,
    l.latitude,
    l.longitude,
    l.fema_failure_reason,
    l.flood_zone_updated_at,
    l.geocoded_at
FROM listings l
WHERE LOWER(TRIM(COALESCE(l.standard_status, ''))) IN ('active', 'pending')
  AND l.internet_address_display_yn IS TRUE
  AND l.latitude IS NOT NULL
  AND l.longitude IS NOT NULL
  AND (
    l.fema_failure_reason = 'no_nfhl_feature'
    OR (
      l.fema_failure_reason IS NULL
      AND l.flood_zone_updated_at IS NOT NULL
      AND l.fema_flood_zone_code IS NULL
    )
  )
  AND (
    l.latitude < -60
    OR (l.latitude = 0 AND l.longitude = 0)
    OR (
      UPPER(TRIM(COALESCE(l.state_or_province, ''))) = 'FL'
      AND (
        ABS(l.latitude) > 50 AND ABS(l.longitude) < 45
        OR l.latitude < 24.5 OR l.latitude > 31.2
        OR l.longitude < -87.8 OR l.longitude > -79.8
      )
    )
  )
ORDER BY l.id
LIMIT 500;

-- Apply after review: reset FEMA watermark so geocode recovery + re-enrichment can run.
-- UPDATE listings AS l
-- SET
--     fema_failure_reason = 'insufficient_coords',
--     flood_zone_updated_at = NULL,
--     fema_failed_at = NULL,
--     updated_at = NOW()
-- WHERE LOWER(TRIM(COALESCE(l.standard_status, ''))) IN ('active', 'pending')
--   AND l.internet_address_display_yn IS TRUE
--   AND l.latitude IS NOT NULL
--   AND l.longitude IS NOT NULL
--   AND (
--     l.fema_failure_reason = 'no_nfhl_feature'
--     OR (
--       l.fema_failure_reason IS NULL
--       AND l.flood_zone_updated_at IS NOT NULL
--       AND l.fema_flood_zone_code IS NULL
--     )
--   )
--   AND (
--     l.latitude < -60
--     OR (l.latitude = 0 AND l.longitude = 0)
--     OR (
--       UPPER(TRIM(COALESCE(l.state_or_province, ''))) = 'FL'
--       AND (
--         ABS(l.latitude) > 50 AND ABS(l.longitude) < 45
--         OR l.latitude < 24.5 OR l.latitude > 31.2
--         OR l.longitude < -87.8 OR l.longitude > -79.8
--       )
--     )
--   );

ROLLBACK;
