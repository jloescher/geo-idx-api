-- listings_field_promote_backfill.sql
--
-- Batched backfill: promote 14 RESO scalar keys from custom_fields / raw_data
-- into typed listings columns, then strip those keys from custom_fields
-- (and optionally raw_data — see STRIP POLICY note below).
--
-- MIGRATION PREREQUISITE:
--   Run Goose migration 00006_listings_idx_internet_columns.sql first to ADD the
--   typed columns before this backfill is executed.  Example skeleton:
--
--     ALTER TABLE listings
--       ADD COLUMN IF NOT EXISTS garage_spaces                            NUMERIC(5,1) NULL,
--       ADD COLUMN IF NOT EXISTS mls_area_major                           VARCHAR(160) NULL,
--       ADD COLUMN IF NOT EXISTS days_on_market                           SMALLINT NULL,
--       ADD COLUMN IF NOT EXISTS tax_annual_amount                        NUMERIC(14,2) NULL,
--       ADD COLUMN IF NOT EXISTS heating_yn                               BOOLEAN NULL,
--       ADD COLUMN IF NOT EXISTS cooling_yn                               BOOLEAN NULL,
--       ADD COLUMN IF NOT EXISTS carport_yn                               BOOLEAN NULL,
--       ADD COLUMN IF NOT EXISTS attached_garage_yn                       BOOLEAN NULL,
--       ADD COLUMN IF NOT EXISTS internet_consumer_comment_yn             BOOLEAN NULL,
--       ADD COLUMN IF NOT EXISTS internet_address_display_yn              BOOLEAN NULL,
--       ADD COLUMN IF NOT EXISTS internet_entire_listing_display_yn       BOOLEAN NULL,
--       ADD COLUMN IF NOT EXISTS internet_automated_valuation_display_yn  BOOLEAN NULL,
--       ADD COLUMN IF NOT EXISTS idx_participation_yn                     BOOLEAN NULL,   -- Stellar only; NULL on beaches
--       ADD COLUMN IF NOT EXISTS idx_office_participation_yn              BOOLEAN NULL;   -- sparse today; added for forward compat
--
-- COLUMNS PROMOTED (14 IDX/facet + 2 address/remarks):
--
--   RESO key                             Typed column                              Type         Notes
--   ─────────────────────────────────────────────────────────────────────────────────────────────────────────────
--   GarageSpaces                         garage_spaces                             NUMERIC(5,2) Clamped 0–999.99 (matches Go garageSpacesPtr)
--   MLSAreaMajor                         mls_area_major                            VARCHAR(400) Truncated to 400 chars on promote
--   DaysOnMarket                         days_on_market                            INTEGER      Changes every sync — see WARNING below
--   TaxAnnualAmount                      tax_annual_amount                         NUMERIC(14,2) e.g. 7741.92
--   HeatingYN                            heating_yn                                BOOLEAN
--   CoolingYN                            cooling_yn                                BOOLEAN
--   CarportYN                            carport_yn                                BOOLEAN
--   AttachedGarageYN                     attached_garage_yn                        BOOLEAN
--   InternetConsumerCommentYN            internet_consumer_comment_yn              BOOLEAN      IDX compliance — see STRIP POLICY
--   InternetAddressDisplayYN             internet_address_display_yn               BOOLEAN      IDX compliance
--   InternetEntireListingDisplayYN       internet_entire_listing_display_yn        BOOLEAN      IDX compliance (search gate)
--   InternetAutomatedValuationDisplayYN  internet_automated_valuation_display_yn   BOOLEAN      IDX compliance
--   IDXParticipationYN                   idx_participation_yn                      BOOLEAN      Stellar only (143,837 rows); NULL on beaches
--   IDXOfficeParticipationYN             idx_office_participation_yn               BOOLEAN      0 rows today; added for forward compat
--   UnparsedAddress                      unparsed_address                            VARCHAR(500) Geocode input
--   PublicRemarks                        public_remarks                              TEXT         FTS on public search
--
-- ─────────────────────────────────────────────────────────────────────────────────────────────
-- STRIP POLICY:
--
--   custom_fields (overflow store): ALL 14 keys are stripped.
--     These fields are now promoted to typed columns and no longer belong in the
--     overflow JSONB bucket.  After the Go sync code is updated (BuildListingRecord),
--     new syncs will not write them back to custom_fields.
--
--   raw_data (RESO Property JSON — drives API response):
--     ● Scalar fields (GarageSpaces, MLSAreaMajor, DaysOnMarket, TaxAnnualAmount,
--       HeatingYN, CoolingYN, CarportYN, AttachedGarageYN): KEPT in raw_data.
--       MergeMirrorListing returns raw_data as the core RESO Property payload;
--       stripping these would remove them from every API consumer response.
--       They remain as zero-duplication-cost keys (raw_data is read-only for the API).
--     ● IDX/Internet booleans (6 keys): STRIPPED from raw_data per prior plan audit.
--       These are compliance gates applied server-side before search results are
--       returned.  API consumers do not need to re-evaluate them.
--
-- ─────────────────────────────────────────────────────────────────────────────────────────────
-- BOOLEAN PARSE:
--   RESO stores booleans as JSON-native true/false (not "true"/"false" strings).
--   In JSONB, ->> returns the text representation: 'true' or 'false'.
--   PostgreSQL casts ('true')::boolean → TRUE and ('false')::boolean → FALSE cleanly.
--   NULL ->> 'key' returns NULL; NULL::boolean → NULL, so absent keys produce NULL.
--   This means no explicit CASE WHEN guard is needed for the cast.
--
-- ─────────────────────────────────────────────────────────────────────────────────────────────
-- COALESCE SEMANTICS:
--   COALESCE(existing_col, <cf parse>, <raw parse>)
--     ● If the typed column is already non-NULL (re-run safety), it is preserved.
--     ● Otherwise, custom_fields is tried first; raw_data is the fallback.
--     ● COALESCE short-circuits: FALSE is non-NULL and is returned as-is (correct).
--
-- ─────────────────────────────────────────────────────────────────────────────────────────────
-- WARNING — DaysOnMarket is a live counter:
--   DaysOnMarket is recalculated by the MLS on every sync.  After this backfill,
--   the Go sync code (BuildListingRecord / listing_mirror.go upsert) must also map
--   DaysOnMarket → days_on_market on every persist pass so the column stays current.
--   The backfill sets a point-in-time value; it is not self-updating.
--
-- ─────────────────────────────────────────────────────────────────────────────────────────────
-- PRODUCTION AUDIT BASELINE (2026-05-27, 190,528 listings rows):
--
--   custom_fields counts before backfill:
--     InternetConsumerCommentYN / InternetEntireListingDisplayYN / InternetAutomatedValuationDisplayYN:
--       ~190,539 rows (both datasets)
--     InternetAddressDisplayYN:
--       ~190,528 rows (both datasets)
--     IDXParticipationYN:
--       ~143,837 rows (stellar only)
--     IDXOfficeParticipationYN:
--       0 rows (not present in Property rows today)
--     GarageSpaces / MLSAreaMajor / DaysOnMarket / TaxAnnualAmount / HeatingYN /
--     CoolingYN / CarportYN / AttachedGarageYN:
--       present in both datasets; exact counts not yet measured.
--
-- ─────────────────────────────────────────────────────────────────────────────────────────────
-- NOT a Goose migration — run manually on the Patroni primary during a low-traffic window.
-- Runs in ID-range batches with COMMIT after each batch (reduces WAL/IO spikes).
-- Requires PostgreSQL 11+ (PROCEDURE + COMMIT in loop).
--
-- Optional: relax session limits for long maintenance
-- SET statement_timeout = 0;
-- SET lock_timeout = '30s';
--
-- Usage:
--   \i docs/scripts/listings_field_promote_backfill.sql
--
--   Or call with a custom batch size:
--   CALL run_listings_field_promote_backfill(1000);
--   DROP PROCEDURE IF EXISTS run_listings_field_promote_backfill(bigint);
--
-- Default batch size: 2500 rows per id slice.
-- With ~190k rows and batch=2500 expect ~77 batches.
-- Lower to 1000 if you see I/O pressure; raise only when idle I/O is ample.
-- ─────────────────────────────────────────────────────────────────────────────────────────────

-- ─────────────────────────────────────────────────────────────────────────────────────────────
-- PRE-FLIGHT CHECKS (run before backfill; expect non-zero in all columns)
-- ─────────────────────────────────────────────────────────────────────────────────────────────
--
-- SELECT
--   COUNT(*) FILTER (WHERE garage_spaces IS NULL)                            AS garage_spaces_null,
--   COUNT(*) FILTER (WHERE mls_area_major IS NULL)                           AS mls_area_major_null,
--   COUNT(*) FILTER (WHERE days_on_market IS NULL)                           AS days_on_market_null,
--   COUNT(*) FILTER (WHERE tax_annual_amount IS NULL)                        AS tax_annual_amount_null,
--   COUNT(*) FILTER (WHERE heating_yn IS NULL)                               AS heating_yn_null,
--   COUNT(*) FILTER (WHERE cooling_yn IS NULL)                               AS cooling_yn_null,
--   COUNT(*) FILTER (WHERE carport_yn IS NULL)                               AS carport_yn_null,
--   COUNT(*) FILTER (WHERE attached_garage_yn IS NULL)                       AS attached_garage_yn_null,
--   COUNT(*) FILTER (WHERE internet_entire_listing_display_yn IS NULL)       AS ield_yn_null,
--   COUNT(*) FILTER (WHERE idx_participation_yn IS NULL AND dataset_slug='stellar') AS stellar_idx_participation_null,
--   COUNT(*) FILTER (WHERE custom_fields ?| ARRAY[
--     'GarageSpaces','MLSAreaMajor','DaysOnMarket','TaxAnnualAmount',
--     'HeatingYN','CoolingYN','CarportYN','AttachedGarageYN',
--     'InternetConsumerCommentYN','InternetAddressDisplayYN',
--     'InternetEntireListingDisplayYN','InternetAutomatedValuationDisplayYN',
--     'IDXParticipationYN','IDXOfficeParticipationYN'
--   ]) AS rows_with_any_key_in_cf
-- FROM listings;
--
-- ─────────────────────────────────────────────────────────────────────────────────────────────
-- POST-VERIFY CHECKS (run after backfill; cf and raw_idx counts should be 0)
-- ─────────────────────────────────────────────────────────────────────────────────────────────
--
-- SELECT
--   -- custom_fields strip (expect 0 for all)
--   COUNT(*) FILTER (WHERE custom_fields ?| ARRAY[
--     'GarageSpaces','MLSAreaMajor','DaysOnMarket','TaxAnnualAmount',
--     'HeatingYN','CoolingYN','CarportYN','AttachedGarageYN',
--     'InternetConsumerCommentYN','InternetAddressDisplayYN',
--     'InternetEntireListingDisplayYN','InternetAutomatedValuationDisplayYN',
--     'IDXParticipationYN','IDXOfficeParticipationYN'
--   ]) AS keys_remaining_in_cf,
--
--   -- raw_data strip for IDX booleans only (expect 0 for all 6)
--   COUNT(*) FILTER (WHERE raw_data ?| ARRAY[
--     'InternetConsumerCommentYN','InternetAddressDisplayYN',
--     'InternetEntireListingDisplayYN','InternetAutomatedValuationDisplayYN',
--     'IDXParticipationYN','IDXOfficeParticipationYN'
--   ]) AS idx_keys_remaining_in_raw,
--
--   -- scalar fields kept in raw_data (expect same as total non-null rows)
--   COUNT(*) FILTER (WHERE raw_data ? 'GarageSpaces')     AS raw_has_garage_spaces,
--   COUNT(*) FILTER (WHERE raw_data ? 'HeatingYN')        AS raw_has_heating_yn,
--
--   -- typed column coverage spot-check
--   COUNT(*) FILTER (WHERE internet_entire_listing_display_yn IS NOT NULL) AS ield_yn_populated,
--   COUNT(*) FILTER (WHERE idx_participation_yn IS NOT NULL)               AS idx_part_yn_populated,
--   COUNT(*) FILTER (WHERE garage_spaces IS NOT NULL)                      AS garage_spaces_populated
-- FROM listings;
--
-- Overflow audit (run before backfill; garage > 999.99 causes NUMERIC(5,2) errors without clamp):
--   SELECT COUNT(*) FROM listings
--   WHERE (custom_fields->>'GarageSpaces') ~ '^-?[0-9]+(\.[0-9]+)?$'
--     AND (custom_fields->>'GarageSpaces')::numeric > 999.99;
--
-- ─────────────────────────────────────────────────────────────────────────────────────────────

-- Safe parsers: mirror Go clamp rules (listing_row.go garageSpacesPtr, normalize.go).
CREATE OR REPLACE FUNCTION backfill_promote_garage_spaces(p_cf jsonb, p_rd jsonb)
RETURNS numeric(5,2)
LANGUAGE sql
IMMUTABLE
AS $$
  WITH src AS (
    SELECT NULLIF(trim(COALESCE(p_cf->>'GarageSpaces', p_rd->>'GarageSpaces')), '') AS s
  )
  SELECT CASE
    WHEN s IS NULL THEN NULL
    WHEN s !~ '^-?[0-9]+(\.[0-9]+)?$' THEN NULL
    ELSE LEAST(999.99::numeric, GREATEST(0::numeric, s::numeric))::numeric(5,2)
  END
  FROM src;
$$;

CREATE OR REPLACE FUNCTION backfill_promote_tax_annual_amount(p_cf jsonb, p_rd jsonb)
RETURNS numeric(14,2)
LANGUAGE sql
IMMUTABLE
AS $$
  WITH src AS (
    SELECT NULLIF(trim(COALESCE(p_cf->>'TaxAnnualAmount', p_rd->>'TaxAnnualAmount')), '') AS s
  )
  SELECT CASE
    WHEN s IS NULL THEN NULL
    WHEN s !~ '^-?[0-9]+(\.[0-9]+)?$' THEN NULL
    ELSE LEAST(999999999999.99::numeric, GREATEST(-999999999999.99::numeric, s::numeric))::numeric(14,2)
  END
  FROM src;
$$;

CREATE OR REPLACE FUNCTION backfill_promote_days_on_market(p_cf jsonb, p_rd jsonb)
RETURNS integer
LANGUAGE sql
IMMUTABLE
AS $$
  WITH src AS (
    SELECT NULLIF(trim(COALESCE(p_cf->>'DaysOnMarket', p_rd->>'DaysOnMarket')), '') AS s
  )
  SELECT CASE
    WHEN s IS NULL THEN NULL
    WHEN s !~ '^-?[0-9]+$' THEN NULL
    WHEN s::bigint < -2147483648 OR s::bigint > 2147483647 THEN NULL
    ELSE s::integer
  END
  FROM src;
$$;

CREATE OR REPLACE FUNCTION backfill_promote_mls_area_major(p_cf jsonb, p_rd jsonb)
RETURNS varchar(400)
LANGUAGE sql
IMMUTABLE
AS $$
  SELECT LEFT(
    NULLIF(trim(COALESCE(p_cf->>'MLSAreaMajor', p_rd->>'MLSAreaMajor')), ''),
    400
  );
$$;

CREATE OR REPLACE PROCEDURE run_listings_field_promote_backfill(p_batch_size bigint DEFAULT 2500)
LANGUAGE plpgsql
AS $$
DECLARE
  v_lo    bigint := 0;
  v_hi    bigint;
  v_max   bigint;
  v_batch int;
  v_total bigint := 0;
BEGIN
  IF p_batch_size IS NULL OR p_batch_size < 1 THEN
    RAISE EXCEPTION 'p_batch_size must be >= 1';
  END IF;

  SELECT COALESCE(MAX(id), 0) INTO v_max FROM listings;
  RAISE NOTICE 'listings_field_promote_backfill: max(id)=%, batch_size=%', v_max, p_batch_size;

  WHILE v_lo < v_max LOOP
    v_hi := v_lo + p_batch_size;

    UPDATE listings SET

      -- ────────────────────────────────────────────────────────────────────
      -- NUMERIC / TEXT SCALAR FIELDS
      -- Source priority: custom_fields first, raw_data fallback.
      -- COALESCE(existing_col, ...) preserves already-promoted values on re-runs.
      -- NULLIF(..., '') guards against empty-string bleed from ->> on numeric cols.
      -- ────────────────────────────────────────────────────────────────────

      garage_spaces = COALESCE(
        garage_spaces,
        backfill_promote_garage_spaces(custom_fields, raw_data)
      ),

      mls_area_major = COALESCE(
        mls_area_major,
        backfill_promote_mls_area_major(custom_fields, raw_data)
      ),

      -- DaysOnMarket: point-in-time snapshot from sync.
      -- The Go sync code must also update this column on every persist pass.
      days_on_market = COALESCE(
        days_on_market,
        backfill_promote_days_on_market(custom_fields, raw_data)
      ),

      tax_annual_amount = COALESCE(
        tax_annual_amount,
        backfill_promote_tax_annual_amount(custom_fields, raw_data)
      ),

      -- ────────────────────────────────────────────────────────────────────
      -- BOOLEAN FIELDS (scalar amenity flags)
      -- RESO stores these as JSON-native booleans (true / false, not strings).
      -- ->> extracts as text 'true' / 'false'; ::boolean casts cleanly.
      -- NULL key → NULL text → NULL boolean (no NULLIF needed for booleans).
      -- ────────────────────────────────────────────────────────────────────

      heating_yn = COALESCE(
        heating_yn,
        (custom_fields->>'HeatingYN')::boolean,
        (raw_data->>'HeatingYN')::boolean
      ),

      cooling_yn = COALESCE(
        cooling_yn,
        (custom_fields->>'CoolingYN')::boolean,
        (raw_data->>'CoolingYN')::boolean
      ),

      carport_yn = COALESCE(
        carport_yn,
        (custom_fields->>'CarportYN')::boolean,
        (raw_data->>'CarportYN')::boolean
      ),

      attached_garage_yn = COALESCE(
        attached_garage_yn,
        (custom_fields->>'AttachedGarageYN')::boolean,
        (raw_data->>'AttachedGarageYN')::boolean
      ),

      -- ────────────────────────────────────────────────────────────────────
      -- IDX / INTERNET BOOLEAN FIELDS (compliance gates)
      -- Present in ~190k rows across both datasets; IDXParticipationYN is
      -- Stellar-only (NULL for beaches rows after strip — that is correct).
      -- IDXOfficeParticipationYN has 0 rows today; column added for forward compat.
      -- ────────────────────────────────────────────────────────────────────

      internet_consumer_comment_yn = COALESCE(
        internet_consumer_comment_yn,
        (custom_fields->>'InternetConsumerCommentYN')::boolean,
        (raw_data->>'InternetConsumerCommentYN')::boolean
      ),

      internet_address_display_yn = COALESCE(
        internet_address_display_yn,
        (custom_fields->>'InternetAddressDisplayYN')::boolean,
        (raw_data->>'InternetAddressDisplayYN')::boolean
      ),

      internet_entire_listing_display_yn = COALESCE(
        internet_entire_listing_display_yn,
        (custom_fields->>'InternetEntireListingDisplayYN')::boolean,
        (raw_data->>'InternetEntireListingDisplayYN')::boolean
      ),

      internet_automated_valuation_display_yn = COALESCE(
        internet_automated_valuation_display_yn,
        (custom_fields->>'InternetAutomatedValuationDisplayYN')::boolean,
        (raw_data->>'InternetAutomatedValuationDisplayYN')::boolean
      ),

      idx_participation_yn = COALESCE(
        idx_participation_yn,
        (custom_fields->>'IDXParticipationYN')::boolean,
        (raw_data->>'IDXParticipationYN')::boolean
      ),

      idx_office_participation_yn = COALESCE(
        idx_office_participation_yn,
        (custom_fields->>'IDXOfficeParticipationYN')::boolean,
        (raw_data->>'IDXOfficeParticipationYN')::boolean
      ),

      unparsed_address = COALESCE(
        unparsed_address,
        NULLIF(COALESCE(
          custom_fields->>'UnparsedAddress',
          raw_data->>'UnparsedAddress'
        ), '')
      ),

      public_remarks = COALESCE(
        public_remarks,
        NULLIF(COALESCE(
          custom_fields->>'PublicRemarks',
          raw_data->>'PublicRemarks'
        ), '')
      ),

      -- ────────────────────────────────────────────────────────────────────
      -- STRIP FROM custom_fields (14 IDX/facet keys + address + remarks)
      -- These are promoted; they no longer belong in the overflow store.
      -- NULLIF(... - key - key, '{}') collapses to NULL if custom_fields
      -- becomes an empty object after stripping, matching existing convention.
      -- ────────────────────────────────────────────────────────────────────

      custom_fields = NULLIF(
        COALESCE(custom_fields, '{}'::jsonb)
          - 'GarageSpaces'
          - 'MLSAreaMajor'
          - 'DaysOnMarket'
          - 'TaxAnnualAmount'
          - 'HeatingYN'
          - 'CoolingYN'
          - 'CarportYN'
          - 'AttachedGarageYN'
          - 'InternetConsumerCommentYN'
          - 'InternetAddressDisplayYN'
          - 'InternetEntireListingDisplayYN'
          - 'InternetAutomatedValuationDisplayYN'
          - 'IDXParticipationYN'
          - 'IDXOfficeParticipationYN'
          - 'UnparsedAddress'
          - 'PublicRemarks',
        '{}'::jsonb
      ),

      -- ────────────────────────────────────────────────────────────────────
      -- STRIP FROM raw_data: IDX/Internet booleans + promoted address/remarks.
      -- Scalar fields (GarageSpaces, MLSAreaMajor, etc.) are intentionally
      -- KEPT in raw_data so that MergeMirrorListing returns them to API consumers
      -- as part of the standard RESO Property payload.
      -- IDX booleans are stripped because they are compliance gates applied
      -- server-side; they do not need to be surfaced to API consumers.
      -- ────────────────────────────────────────────────────────────────────

      raw_data = NULLIF(
        COALESCE(raw_data, '{}'::jsonb)
          - 'InternetConsumerCommentYN'
          - 'InternetAddressDisplayYN'
          - 'InternetEntireListingDisplayYN'
          - 'InternetAutomatedValuationDisplayYN'
          - 'IDXParticipationYN'
          - 'IDXOfficeParticipationYN'
          - 'UnparsedAddress'
          - 'PublicRemarks',
        '{}'::jsonb
      ),

      updated_at = NOW()

    WHERE id > v_lo AND id <= v_hi
      AND (
        custom_fields ?| ARRAY[
          'GarageSpaces', 'MLSAreaMajor', 'DaysOnMarket', 'TaxAnnualAmount',
          'HeatingYN', 'CoolingYN', 'CarportYN', 'AttachedGarageYN',
          'InternetConsumerCommentYN', 'InternetAddressDisplayYN',
          'InternetEntireListingDisplayYN', 'InternetAutomatedValuationDisplayYN',
          'IDXParticipationYN', 'IDXOfficeParticipationYN',
          'UnparsedAddress', 'PublicRemarks'
        ]
        OR raw_data ?| ARRAY[
          'InternetConsumerCommentYN', 'InternetAddressDisplayYN',
          'InternetEntireListingDisplayYN', 'InternetAutomatedValuationDisplayYN',
          'IDXParticipationYN', 'IDXOfficeParticipationYN',
          'UnparsedAddress', 'PublicRemarks'
        ]
      );

    GET DIAGNOSTICS v_batch = ROW_COUNT;
    v_total := v_total + v_batch;
    RAISE NOTICE 'id (% , %]: updated % rows (running total %)', v_lo, v_hi, v_batch, v_total;

    COMMIT;
    v_lo := v_hi;
  END LOOP;

  RAISE NOTICE 'listings_field_promote_backfill: finished, % rows updated across % batches',
    v_total, CEIL(v_max::numeric / NULLIF(p_batch_size, 0));
END;
$$;

-- Run with default batch size (2500). Do not wrap in BEGIN; procedure commits each batch.
CALL run_listings_field_promote_backfill(2500);

DROP PROCEDURE IF EXISTS run_listings_field_promote_backfill(bigint);

-- ─────────────────────────────────────────────────────────────────────────────────────────────
-- GO SYNC CODE FOLLOW-UP REQUIRED
-- ─────────────────────────────────────────────────────────────────────────────────────────────
-- After running this backfill, update the Go sync pipeline to prevent these keys
-- from being re-written to custom_fields on future replication cycles:
--
-- 1. internal/service/mls/listing_row.go (ListingRecord struct):
--    Add fields:
--      GarageSpaces              *float64
--      MLSAreaMajor              *string
--      DaysOnMarket              *int16
--      TaxAnnualAmount           *float64
--      HeatingYN                 *bool
--      CoolingYN                 *bool
--      CarportYN                 *bool
--      AttachedGarageYN          *bool
--      InternetConsumerCommentYN         *bool
--      InternetAddressDisplayYN          *bool
--      InternetEntireListingDisplayYN    *bool
--      InternetAutomatedValuationDisplayYN *bool
--      IDXParticipationYN                *bool
--      IDXOfficeParticipationYN          *bool
--
-- 2. BuildListingRecord() mappings (same file):
--      GarageSpaces:                       ClampNumeric5_1Ptr(row["GarageSpaces"]),
--      MLSAreaMajor:                       optionalString(row["MLSAreaMajor"]),
--      DaysOnMarket:                       BoundedInt16Ptr(row["DaysOnMarket"]),
--      TaxAnnualAmount:                    ClampNumeric14_2Ptr(row["TaxAnnualAmount"]),
--      HeatingYN:                          boolPtr(row["HeatingYN"]),
--      CoolingYN:                          boolPtr(row["CoolingYN"]),
--      CarportYN:                          boolPtr(row["CarportYN"]),
--      AttachedGarageYN:                   boolPtr(row["AttachedGarageYN"]),
--      InternetConsumerCommentYN:          boolPtr(row["InternetConsumerCommentYN"]),
--      InternetAddressDisplayYN:           boolPtr(row["InternetAddressDisplayYN"]),
--      InternetEntireListingDisplayYN:     boolPtr(row["InternetEntireListingDisplayYN"]),
--      InternetAutomatedValuationDisplayYN: boolPtr(row["InternetAutomatedValuationDisplayYN"]),
--      IDXParticipationYN:                 boolPtr(row["IDXParticipationYN"]),
--      IDXOfficeParticipationYN:           boolPtr(row["IDXOfficeParticipationYN"]),
--
-- 3. internal/service/sync/listing_mirror.go (upsert SET clause):
--    Bind the 14 new ListingRecord fields to the corresponding SQL parameters.
--    Existing boolPtr helpers and ClampNumeric helpers cover the type coercions.
--    Add the 14 columns to the INSERT column list and ON CONFLICT DO UPDATE SET list.
--
-- 4. internal/service/mls/listing_payload.go (BuildCustomFields exclusion list):
--    Add the 14 RESO key names to the set of keys excluded from custom_fields
--    so future syncs do not re-populate the overflow bucket with promoted fields.
--
-- 5. internal/service/mls/listing_payload.go (StripJSONBKeysFromRaw):
--    Add 'InternetConsumerCommentYN', 'InternetAddressDisplayYN',
--    'InternetEntireListingDisplayYN', 'InternetAutomatedValuationDisplayYN',
--    'IDXParticipationYN', 'IDXOfficeParticipationYN' to the navigation strip
--    keys (or a new "raw_data strip keys" list) so future syncs also drop the
--    6 IDX booleans from raw_data at write time.
--
-- 6. RECOMMENDED INDEXES (add to migration 00006 or a follow-up 00007):
--
--    -- Compliance gate index (search default: hide non-compliant listings)
--    CREATE INDEX listings_ap_ds_mod_ts_compliant_idx
--      ON listings (dataset_slug, modification_timestamp DESC)
--      WHERE LOWER(TRIM(COALESCE(standard_status,''))) IN ('active','pending')
--        AND internet_entire_listing_display_yn IS NOT FALSE
--        AND modification_timestamp IS NOT NULL;
--
--    -- IDX participation filter (Stellar only, highly selective)
--    CREATE INDEX listings_stellar_idx_participation_idx
--      ON listings (idx_participation_yn)
--      WHERE dataset_slug = 'stellar'
--        AND LOWER(TRIM(COALESCE(standard_status,''))) IN ('active','pending')
--        AND idx_participation_yn IS NOT FALSE;
--
--    -- Garage spaces range filter (comps engine)
--    CREATE INDEX listings_ap_ds_garage_spaces_idx
--      ON listings (dataset_slug, garage_spaces)
--      WHERE LOWER(TRIM(COALESCE(standard_status,''))) IN ('active','pending')
--        AND garage_spaces IS NOT NULL;
--
--    -- MLS area major equality filter (comps engine + search)
--    CREATE INDEX listings_ap_ds_mls_area_major_idx
--      ON listings (dataset_slug, mls_area_major)
--      WHERE LOWER(TRIM(COALESCE(standard_status,''))) IN ('active','pending')
--        AND mls_area_major IS NOT NULL;
--
--    Note: HeatingYN, CoolingYN, CarportYN, AttachedGarageYN are low-selectivity
--    booleans (~100% TRUE in sample data).  Partial indexes on these alone offer
--    no benefit; include only if combined with a high-selectivity predicate in
--    a multi-column covering index.
-- ─────────────────────────────────────────────────────────────────────────────────────────────
