-- Recompute low_risk_flood_zone_yn from MLS flood_zone_code where FEMA returned no_nfhl_feature.
-- Run on Patroni primary only after review (see docs/production-data-backfill.md).
--
-- Usage:
--   1) Run preview SELECT and verify counts.
--   2) Uncomment UPDATE to apply.

BEGIN;

SELECT
    COUNT(*) AS eligible,
    COUNT(*) FILTER (WHERE UPPER(TRIM(flood_zone_code)) LIKE '%X%') AS with_mls_x
FROM listings
WHERE fema_failure_reason = 'no_nfhl_feature'
  AND flood_zone_code IS NOT NULL
  AND TRIM(flood_zone_code) <> '';

-- Preview low_risk changes (Go ComputeLowRiskFloodZoneYN rules approximated in SQL).
SELECT id, listing_key, flood_zone_code, low_risk_flood_zone_yn AS current_low_risk
FROM listings
WHERE fema_failure_reason = 'no_nfhl_feature'
  AND flood_zone_code IS NOT NULL
  AND TRIM(flood_zone_code) <> ''
  AND (
    UPPER(flood_zone_code) LIKE '%X500%'
    OR LOWER(flood_zone_code) LIKE '%no%'
    OR UPPER(flood_zone_code) LIKE '%X%'
  )
  AND UPPER(flood_zone_code) NOT LIKE '%A%'
  AND UPPER(flood_zone_code) NOT LIKE '%V%'
LIMIT 20;

-- UPDATE listings
-- SET low_risk_flood_zone_yn = (
--   CASE
--     WHEN flood_zone_code IS NULL OR TRIM(flood_zone_code) = '' THEN FALSE
--     WHEN UPPER(flood_zone_code) LIKE '%A%' OR UPPER(flood_zone_code) LIKE '%V%' THEN FALSE
--     WHEN UPPER(flood_zone_code) LIKE '%X500%' THEN TRUE
--     WHEN LOWER(flood_zone_code) LIKE '%no%' THEN TRUE
--     WHEN UPPER(flood_zone_code) LIKE '%X%' THEN TRUE
--     ELSE FALSE
--   END
-- ),
-- updated_at = NOW()
-- WHERE fema_failure_reason = 'no_nfhl_feature'
--   AND flood_zone_code IS NOT NULL
--   AND TRIM(flood_zone_code) <> '';

ROLLBACK;
