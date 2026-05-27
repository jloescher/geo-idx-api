-- gis_cities_county_expand.sql
--
-- One-time (idempotent) expansion of gis_cities to one row per (city_name, county_slug).
-- Run on Patroni primary BEFORE migration 00008_gis_cities_county_not_null.sql.
--
-- Usage:
--   psql "$GOOSE_DBSTRING" -f docs/scripts/gis_cities_county_expand.sql
--
-- Pre-flight:
--   SELECT COUNT(*) FROM gis_cities WHERE county IS NULL;
--
-- Post-verify (expect 0 NULL counties after expand; run 00008 for NOT NULL constraint):
--   SELECT COUNT(*) FROM gis_cities WHERE county IS NULL;
--
-- Pair completeness (expect 0):
--   WITH intersecting AS (
--     SELECT c.city_name, COUNT(DISTINCT co.county_slug) AS need
--     FROM gis_cities c
--     JOIN gis_counties co ON ST_Intersects(c.geometry, co.geometry)
--       AND c.source_generation = co.source_generation
--     GROUP BY c.city_name
--   ),
--   have AS (
--     SELECT city_name, COUNT(DISTINCT county) AS have
--     FROM gis_cities WHERE county IS NOT NULL GROUP BY city_name
--   )
--   SELECT COUNT(*) FROM intersecting i
--   LEFT JOIN have h ON lower(i.city_name) = lower(h.city_name)
--   WHERE COALESCE(h.have, 0) < i.need;

DO $$
DECLARE
  gen int;
  ins bigint;
  del bigint;
BEGIN
  FOR gen IN SELECT DISTINCT source_generation FROM gis_cities ORDER BY 1
  LOOP
    RAISE NOTICE 'expanding generation %', gen;
    WITH cities AS (
      SELECT DISTINCT ON (city_name)
        city_name, source_generation, geometry, properties, source_key, source_fingerprint
      FROM gis_cities
      WHERE source_generation = gen
      ORDER BY city_name, id
    ),
    intersecting AS (
      SELECT c.city_name, c.source_generation, co.county_slug,
             c.geometry, c.properties, c.source_key, c.source_fingerprint
      FROM cities c
      JOIN gis_counties co ON co.source_generation = c.source_generation
        AND ST_Intersects(c.geometry, co.geometry)
    ),
    no_intersect AS (
      SELECT c.city_name, c.source_generation,
             nearest.county_slug,
             c.geometry, c.properties, c.source_key, c.source_fingerprint
      FROM cities c
      CROSS JOIN LATERAL (
        SELECT co.county_slug
        FROM gis_counties co
        WHERE co.source_generation = c.source_generation
        ORDER BY ST_Distance(
          c.geometry::geography,
          ST_Centroid(co.geometry)::geography
        )
        LIMIT 1
      ) nearest
      WHERE NOT EXISTS (
        SELECT 1 FROM intersecting i
        WHERE i.city_name = c.city_name AND i.source_generation = c.source_generation
      )
    ),
    pairs AS (
      SELECT city_name, source_generation, county_slug, geometry, properties, source_key, source_fingerprint
      FROM intersecting
      UNION ALL
      SELECT city_name, source_generation, county_slug, geometry, properties, source_key, source_fingerprint
      FROM no_intersect
      WHERE county_slug IS NOT NULL AND county_slug <> ''
    ),
    upserted AS (
      INSERT INTO gis_cities (
        city_name, county, source_key, geometry, properties,
        source_generation, source_fingerprint, last_synced_at
      )
      SELECT city_name, county_slug, source_key, geometry, properties,
             source_generation, source_fingerprint, NOW()
      FROM pairs
      ON CONFLICT (city_name, county) DO UPDATE SET
        source_key = EXCLUDED.source_key,
        geometry = EXCLUDED.geometry,
        properties = EXCLUDED.properties,
        source_generation = EXCLUDED.source_generation,
        source_fingerprint = EXCLUDED.source_fingerprint,
        last_synced_at = NOW()
      RETURNING 1
    ),
    removed AS (
      DELETE FROM gis_cities c
      WHERE c.source_generation = gen
        AND (
          c.county IS NULL
          OR NOT EXISTS (
            SELECT 1 FROM pairs p
            WHERE p.city_name = c.city_name
              AND p.county_slug = c.county
          )
        )
      RETURNING 1
    )
    SELECT (SELECT COUNT(*)::bigint FROM upserted), (SELECT COUNT(*)::bigint FROM removed)
    INTO ins, del;
    RAISE NOTICE 'generation %: upserted %, deleted %', gen, ins, del;
  END LOOP;
END $$;
