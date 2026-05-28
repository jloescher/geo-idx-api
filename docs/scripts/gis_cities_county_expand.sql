-- gis_cities_county_expand.sql
--
-- One-time (idempotent) expansion of gis_cities to one row per (city_name, county_slug).
-- Run on Patroni primary BEFORE migration 00008_gis_cities_county_not_null.sql.
--
-- Prefer Patroni primary :5432 via runner (HAProxy :5000 drops long sessions):
--   docs/scripts/run_gis_cities_county_expand.sh check
--   docs/scripts/run_gis_cities_county_expand.sh 5 reconnect
--
-- Or psql on primary:
--   \i docs/scripts/gis_cities_county_expand.sql
--   CALL run_gis_cities_county_expand(5);          -- monolithic (one connection)
--   -- reconnect: CALL gis_expand_cities_step_call(5); in a loop (see shell runner)
--
-- p_commit_every (monolithic): COMMIT after this many cities inside run_gis_cities_county_expand.
--
-- Pre-flight:
--   SELECT COUNT(*) FROM gis_cities WHERE county IS NULL;
--
-- Post-verify (expect 0 NULL counties):
--   SELECT COUNT(*) FROM gis_cities WHERE county IS NULL;

-- Expand a single city in one generation (no COMMIT — caller commits).
CREATE OR REPLACE PROCEDURE gis_expand_one_city_county_pairs(
  p_generation int,
  p_city_name text,
  OUT p_upserted bigint,
  OUT p_deleted bigint
)
LANGUAGE plpgsql
AS $$
BEGIN
  p_upserted := 0;
  p_deleted := 0;

  WITH cities AS (
    SELECT DISTINCT ON (city_name)
      city_name, source_generation, geometry, properties, source_key, source_fingerprint
    FROM gis_cities
    WHERE source_generation = p_generation
      AND city_name = p_city_name
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
    WHERE c.source_generation = p_generation
      AND c.city_name = p_city_name
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
  SELECT
    COALESCE((SELECT COUNT(*)::bigint FROM upserted), 0),
    COALESCE((SELECT COUNT(*)::bigint FROM removed), 0)
  INTO p_upserted, p_deleted;
END;
$$;

-- Loop all cities; COMMIT every p_commit_every cities to bound WAL and IDE session time.
CREATE OR REPLACE PROCEDURE run_gis_cities_county_expand(p_commit_every int DEFAULT 1)
LANGUAGE plpgsql
AS $$
DECLARE
  v_gen int;
  v_city text;
  v_ins bigint;
  v_del bigint;
  v_total_ins bigint := 0;
  v_total_del bigint := 0;
  v_city_n int := 0;
  v_gen_n int := 0;
  v_pending_cities bigint;
BEGIN
  IF p_commit_every IS NULL OR p_commit_every < 1 THEN
    RAISE EXCEPTION 'p_commit_every must be >= 1';
  END IF;

  SELECT COUNT(DISTINCT city_name) INTO v_pending_cities FROM gis_cities;
  RAISE NOTICE 'gis_cities_county_expand: % distinct city names to process', v_pending_cities;

  FOR v_gen IN SELECT DISTINCT source_generation FROM gis_cities ORDER BY 1
  LOOP
    v_gen_n := v_gen_n + 1;
    RAISE NOTICE 'generation %: starting', v_gen;

    FOR v_city IN
      SELECT DISTINCT city_name
      FROM gis_cities
      WHERE source_generation = v_gen
      ORDER BY 1
    LOOP
      CALL gis_expand_one_city_county_pairs(v_gen, v_city, v_ins, v_del);
      v_total_ins := v_total_ins + v_ins;
      v_total_del := v_total_del + v_del;
      v_city_n := v_city_n + 1;

      IF v_city_n % 50 = 0 THEN
        RAISE NOTICE 'progress: % cities (last=%), upserted %, deleted %',
          v_city_n, v_city, v_total_ins, v_total_del;
      END IF;

      IF v_city_n % p_commit_every = 0 THEN
        COMMIT;
      END IF;
    END LOOP;

    COMMIT;
    RAISE NOTICE 'generation %: done (running totals upserted %, deleted %)', v_gen, v_total_ins, v_total_del;
  END LOOP;

  RAISE NOTICE 'gis_cities_county_expand: finished % cities in % generations, upserted %, deleted %',
    v_city_n, v_gen_n, v_total_ins, v_total_del;
  RAISE NOTICE 'NULL county check: %', (SELECT COUNT(*) FROM gis_cities WHERE county IS NULL);
END;
$$;

-- Process up to p_limit cities that still have a NULL county row (resumable; no in-procedure COMMIT).
CREATE OR REPLACE PROCEDURE gis_expand_cities_step(
  p_limit int,
  OUT p_processed int
)
LANGUAGE plpgsql
AS $$
DECLARE
  v_gen int;
  v_city text;
  v_ins bigint;
  v_del bigint;
  v_pending bigint;
BEGIN
  p_processed := 0;

  IF p_limit IS NULL OR p_limit < 1 THEN
    RAISE EXCEPTION 'p_limit must be >= 1';
  END IF;

  SELECT COUNT(*) INTO v_pending
  FROM (
    SELECT DISTINCT source_generation, city_name
    FROM gis_cities
    WHERE county IS NULL
  ) q;

  IF v_pending = 0 THEN
    RAISE NOTICE 'gis_expand_cities_step: no cities with NULL county (pending=0)';
    RETURN;
  END IF;

  RAISE NOTICE 'gis_expand_cities_step: % city/generation pairs with NULL county; processing up to %',
    v_pending, p_limit;

  FOR v_gen, v_city IN
    SELECT DISTINCT source_generation, city_name
    FROM gis_cities
    WHERE county IS NULL
    ORDER BY source_generation, city_name
    LIMIT p_limit
  LOOP
    CALL gis_expand_one_city_county_pairs(v_gen, v_city, v_ins, v_del);
    p_processed := p_processed + 1;
    RAISE NOTICE 'gis_expand_cities_step: gen=% city=% upserted=% deleted=% (%/% this step)',
      v_gen, v_city, v_ins, v_del, p_processed, p_limit;
  END LOOP;

  RAISE NOTICE 'gis_expand_cities_step: processed % cities; NULL counties remaining: %',
    p_processed, (SELECT COUNT(*) FROM gis_cities WHERE county IS NULL);
END;
$$;

-- psql-friendly entry (CALL cannot omit OUT args on gis_expand_cities_step).
CREATE OR REPLACE PROCEDURE gis_expand_cities_step_call(p_limit int)
LANGUAGE plpgsql
AS $$
DECLARE
  v_processed int;
BEGIN
  CALL gis_expand_cities_step(p_limit, v_processed);
END;
$$;

-- Manual monolithic run:
--   CALL run_gis_cities_county_expand(5);
-- Manual reconnect step:
--   CALL gis_expand_cities_step_call(5);

-- Emergency unblock for 00008 only (Keys islands with no intersection) — run if full expand cannot finish:
-- UPDATE gis_cities g SET county = sub.county_slug, last_synced_at = NOW()
-- FROM (
--   SELECT c.id,
--     (SELECT co.county_slug FROM gis_counties co
--      WHERE co.source_generation = c.source_generation
--      ORDER BY ST_Distance(c.geometry::geography, ST_Centroid(co.geometry)::geography)
--      LIMIT 1) AS county_slug
--   FROM gis_cities c WHERE c.county IS NULL
-- ) sub
-- WHERE g.id = sub.id AND sub.county_slug IS NOT NULL AND sub.county_slug <> '';
