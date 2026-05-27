-- listings_nav_jsonb_cleanup.sql
--
-- One-time repair: move misplaced navigation collections (Media, Unit, Room, OpenHouse
-- and Bridge/Spark aliases) from custom_fields/raw_data into media/unit/room/open_house
-- JSONB columns, then strip those keys from custom_fields and raw_data.
--
-- NOT a Goose migration — run manually on the Patroni primary during a maintenance window.
--
-- Runs in ID-range batches with COMMIT after each batch (reduces WAL/IO spikes vs one
-- monolithic UPDATE). Default batch size: 2500 rows per id slice. Lower if you still
-- see I/O errors (e.g. 1000). Raise only if batches are fast and idle I/O is ample.
--
-- Requires PostgreSQL 11+ (PROCEDURE + COMMIT in loop).
--
-- Usage (psql on primary):
--   \i docs/scripts/listings_nav_jsonb_cleanup.sql
--
-- Or call with a custom batch size:
--   CALL run_listings_nav_jsonb_cleanup(1000);
--   DROP PROCEDURE IF EXISTS run_listings_nav_jsonb_cleanup(bigint);
--
-- Pre-flight (expect non-zero before repair on polluted rows):
--   SELECT COUNT(*) FROM listings
--   WHERE custom_fields ?| ARRAY['Media','Unit','Units','UnitTypes','Room','Rooms','OpenHouse','OpenHouses'];
--
-- Post-verify (expect 0):
--   SELECT COUNT(*) FROM listings
--   WHERE custom_fields ?| ARRAY['Media','Unit','Units','UnitTypes','Room','Rooms','OpenHouse','OpenHouses'];

-- Optional: relax session limits for long maintenance (uncomment if needed)
-- SET statement_timeout = 0;
-- SET lock_timeout = '30s';

CREATE OR REPLACE PROCEDURE run_listings_nav_jsonb_cleanup(p_batch_size bigint DEFAULT 2500)
LANGUAGE plpgsql
AS $$
DECLARE
  v_lo     bigint := 0;
  v_hi     bigint;
  v_max    bigint;
  v_batch  int;
  v_total  bigint := 0;
BEGIN
  IF p_batch_size IS NULL OR p_batch_size < 1 THEN
    RAISE EXCEPTION 'p_batch_size must be >= 1';
  END IF;

  SELECT COALESCE(MAX(id), 0) INTO v_max FROM listings;
  RAISE NOTICE 'listings_nav_jsonb_cleanup: max(id)=%, batch_size=%', v_max, p_batch_size;

  WHILE v_lo < v_max LOOP
    v_hi := v_lo + p_batch_size;

    UPDATE listings SET
      media = COALESCE(media, custom_fields->'Media', raw_data->'Media'),
      unit = COALESCE(
        unit,
        custom_fields->'Unit',
        custom_fields->'UnitTypes',
        custom_fields->'Units',
        raw_data->'Unit',
        raw_data->'UnitTypes',
        raw_data->'Units'
      ),
      room = COALESCE(
        room,
        custom_fields->'Room',
        custom_fields->'Rooms',
        raw_data->'Room',
        raw_data->'Rooms'
      ),
      open_house = COALESCE(
        open_house,
        custom_fields->'OpenHouse',
        custom_fields->'OpenHouses',
        raw_data->'OpenHouse',
        raw_data->'OpenHouses'
      ),
      custom_fields = NULLIF(
        COALESCE(custom_fields, '{}'::jsonb)
        - 'Media' - 'Unit' - 'Units' - 'UnitTypes'
        - 'Room' - 'Rooms' - 'OpenHouse' - 'OpenHouses',
        '{}'::jsonb
      ),
      raw_data = NULLIF(
        COALESCE(raw_data, '{}'::jsonb)
        - 'Media' - 'Unit' - 'Units' - 'UnitTypes'
        - 'Room' - 'Rooms' - 'OpenHouse' - 'OpenHouses',
        '{}'::jsonb
      ),
      updated_at = NOW()
    WHERE id > v_lo AND id <= v_hi
      AND (
        custom_fields ?| ARRAY['Media','Unit','Units','UnitTypes','Room','Rooms','OpenHouse','OpenHouses']
        OR raw_data ?| ARRAY['Media','Unit','Units','UnitTypes','Room','Rooms','OpenHouse','OpenHouses']
        OR (
          (room IS NULL OR room = 'null'::jsonb)
          AND (custom_fields ? 'Room' OR custom_fields ? 'Rooms' OR raw_data ? 'Room' OR raw_data ? 'Rooms')
        )
        OR (
          (open_house IS NULL OR open_house = 'null'::jsonb)
          AND (custom_fields ? 'OpenHouse' OR custom_fields ? 'OpenHouses' OR raw_data ? 'OpenHouse' OR raw_data ? 'OpenHouses')
        )
        OR (
          (unit IS NULL OR unit = 'null'::jsonb)
          AND (custom_fields ? 'Unit' OR custom_fields ? 'UnitTypes' OR custom_fields ? 'Units'
               OR raw_data ? 'Unit' OR raw_data ? 'UnitTypes' OR raw_data ? 'Units')
        )
      );

    GET DIAGNOSTICS v_batch = ROW_COUNT;
    v_total := v_total + v_batch;
    RAISE NOTICE 'id (% , %]: updated % rows (running total %)', v_lo, v_hi, v_batch, v_total;

    COMMIT;
    v_lo := v_hi;
  END LOOP;

  RAISE NOTICE 'listings_nav_jsonb_cleanup: finished, % rows updated across % batches',
    v_total, CEIL(v_max::numeric / NULLIF(p_batch_size, 0));
END;
$$;

-- Run with default batch size (2500). Do not wrap in BEGIN; the procedure commits each batch.
CALL run_listings_nav_jsonb_cleanup(2500);

DROP PROCEDURE IF EXISTS run_listings_nav_jsonb_cleanup(bigint);

-- listings_cache: if closed comp write format changes, operators may purge stale blobs:
--   DELETE FROM listings_cache WHERE last_refreshed_at < NOW() - INTERVAL '30 days';
