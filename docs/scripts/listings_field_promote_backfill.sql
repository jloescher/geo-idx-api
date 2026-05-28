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
-- Usage (prefer shell runner on Patroni :5432 — see docs/production-data-backfill.md):
--
--   docs/scripts/run_listings_field_promote_backfill.sh check
--   docs/scripts/run_listings_field_promote_backfill.sh 500 reconnect
--   (BACKFILL_DSN in docs/scripts/.env.backfill.local, gitignored)
--
-- Or psql on primary (install SQL, then reconnect batches or monolithic CALL):
--   psql "$BACKFILL_DSN" -f docs/scripts/listings_field_promote_backfill.sql
--   CALL listings_field_promote_step_call(500, 'primary');
--   CALL listings_field_promote_step_call(500, 'scalars');
--
-- Batches only rows that still need work. Re-run CALL after disconnect; already-done rows are skipped.
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
-- POST-VERIFY (fast — safe in DataGrip; stops at first match per key)
-- ─────────────────────────────────────────────────────────────────────────────────────────────
--
-- SELECT
--   EXISTS (SELECT 1 FROM listings WHERE custom_fields ? 'GarageSpaces' LIMIT 1) AS cf_garage_left,
--   EXISTS (SELECT 1 FROM listings WHERE custom_fields ? 'InternetEntireListingDisplayYN' LIMIT 1) AS cf_ield_left,
--   EXISTS (SELECT 1 FROM listings WHERE custom_fields ? 'IDXParticipationYN' LIMIT 1) AS cf_idx_left,
--   EXISTS (SELECT 1 FROM listings WHERE raw_data ? 'InternetEntireListingDisplayYN' LIMIT 1) AS raw_ield_left,
--   EXISTS (SELECT 1 FROM listings WHERE raw_data ? 'IDXParticipationYN' LIMIT 1) AS raw_idx_left;
-- -- All five should be false when strip is complete.
--
-- Pending (must be false / 0 before you are done):
--   SELECT COUNT(*) FROM listings l WHERE listings_row_needs_field_promote_row(l);
--
-- Coverage (run one at a time in IDE, or use psql with statement_timeout=0):
--   SELECT COUNT(*) FROM listings WHERE internet_entire_listing_display_yn IS NOT NULL;
--
-- Heavy full-table verify (psql only — one scan, three JSONB filters; JDBC often times out at ~30s):
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
-- Do NOT run full pending COUNT(*) in DataGrip — it scans ~190k rows and JDBC drops ~30s.
-- Use fast EXISTS verify instead (below). Run CALL via psql when possible.
--
-- Completion spot-check:
--   SELECT COUNT(*) FILTER (WHERE internet_entire_listing_display_yn IS NOT NULL) AS ield_ok,
--          COUNT(*) FILTER (WHERE custom_fields ?| ARRAY['GarageSpaces','InternetEntireListingDisplayYN']) AS cf_keys_left,
--          COUNT(*) FILTER (WHERE raw_data ?| ARRAY['InternetEntireListingDisplayYN','IDXParticipationYN']) AS raw_idx_left
--   FROM listings;
--
-- NOTE: GarageSpaces etc. stay in raw_data on purpose — they must NOT count as pending.
--
-- ─────────────────────────────────────────────────────────────────────────────────────────────

DROP FUNCTION IF EXISTS listings_row_needs_field_promote(jsonb, jsonb);

-- Phase 1: keys still in custom_fields or strippable keys in raw_data (main backfill driver).
CREATE OR REPLACE FUNCTION listings_row_needs_field_promote_primary(l listings)
RETURNS boolean
LANGUAGE sql
STABLE
AS $$
  SELECT
    COALESCE(l.custom_fields, '{}'::jsonb) ?| ARRAY[
      'GarageSpaces', 'MLSAreaMajor', 'DaysOnMarket', 'TaxAnnualAmount',
      'HeatingYN', 'CoolingYN', 'CarportYN', 'AttachedGarageYN',
      'InternetConsumerCommentYN', 'InternetAddressDisplayYN',
      'InternetEntireListingDisplayYN', 'InternetAutomatedValuationDisplayYN',
      'IDXParticipationYN', 'IDXOfficeParticipationYN',
      'UnparsedAddress', 'PublicRemarks'
    ]::text[]
    OR COALESCE(l.raw_data, '{}'::jsonb) ?| ARRAY[
      'InternetConsumerCommentYN', 'InternetAddressDisplayYN',
      'InternetEntireListingDisplayYN', 'InternetAutomatedValuationDisplayYN',
      'IDXParticipationYN', 'IDXOfficeParticipationYN',
      'UnparsedAddress', 'PublicRemarks'
    ]::text[];
$$;

CREATE OR REPLACE FUNCTION backfill_promote_boolean(p_cf jsonb, p_rd jsonb, p_key text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
AS $$
  SELECT CASE lower(trim(COALESCE(p_cf->>p_key, p_rd->>p_key, '')))
    WHEN 'true' THEN true
    WHEN 'false' THEN false
    ELSE NULL
  END;
$$;

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

-- Phase 2: typed column null AND a promotable value exists in cf/raw_data.
-- Do not match key-present/unparseable rows (would loop forever updating 500 rows/batch).
CREATE OR REPLACE FUNCTION listings_row_needs_field_promote_scalars(l listings)
RETURNS boolean
LANGUAGE sql
STABLE
AS $$
  SELECT
    (l.garage_spaces IS NULL AND backfill_promote_garage_spaces(l.custom_fields, l.raw_data) IS NOT NULL)
    OR (l.mls_area_major IS NULL AND backfill_promote_mls_area_major(l.custom_fields, l.raw_data) IS NOT NULL)
    OR (l.days_on_market IS NULL AND backfill_promote_days_on_market(l.custom_fields, l.raw_data) IS NOT NULL)
    OR (l.tax_annual_amount IS NULL AND backfill_promote_tax_annual_amount(l.custom_fields, l.raw_data) IS NOT NULL)
    OR (l.heating_yn IS NULL AND backfill_promote_boolean(l.custom_fields, l.raw_data, 'HeatingYN') IS NOT NULL)
    OR (l.cooling_yn IS NULL AND backfill_promote_boolean(l.custom_fields, l.raw_data, 'CoolingYN') IS NOT NULL)
    OR (l.carport_yn IS NULL AND backfill_promote_boolean(l.custom_fields, l.raw_data, 'CarportYN') IS NOT NULL)
    OR (l.attached_garage_yn IS NULL AND backfill_promote_boolean(l.custom_fields, l.raw_data, 'AttachedGarageYN') IS NOT NULL)
    OR (l.internet_consumer_comment_yn IS NULL AND backfill_promote_boolean(l.custom_fields, l.raw_data, 'InternetConsumerCommentYN') IS NOT NULL)
    OR (l.internet_address_display_yn IS NULL AND backfill_promote_boolean(l.custom_fields, l.raw_data, 'InternetAddressDisplayYN') IS NOT NULL)
    OR (l.internet_entire_listing_display_yn IS NULL AND backfill_promote_boolean(l.custom_fields, l.raw_data, 'InternetEntireListingDisplayYN') IS NOT NULL)
    OR (l.internet_automated_valuation_display_yn IS NULL AND backfill_promote_boolean(l.custom_fields, l.raw_data, 'InternetAutomatedValuationDisplayYN') IS NOT NULL)
    OR (l.idx_participation_yn IS NULL AND backfill_promote_boolean(l.custom_fields, l.raw_data, 'IDXParticipationYN') IS NOT NULL)
    OR (l.idx_office_participation_yn IS NULL AND backfill_promote_boolean(l.custom_fields, l.raw_data, 'IDXOfficeParticipationYN') IS NOT NULL)
    OR (l.unparsed_address IS NULL AND NULLIF(trim(COALESCE(l.custom_fields->>'UnparsedAddress', l.raw_data->>'UnparsedAddress')), '') IS NOT NULL)
    OR (l.public_remarks IS NULL AND NULLIF(trim(COALESCE(l.custom_fields->>'PublicRemarks', l.raw_data->>'PublicRemarks')), '') IS NOT NULL);
$$;

CREATE OR REPLACE FUNCTION listings_row_needs_field_promote_row(l listings)
RETURNS boolean
LANGUAGE sql
STABLE
AS $$
  SELECT listings_row_needs_field_promote_primary(l)
      OR listings_row_needs_field_promote_scalars(l);
$$;

-- One batch per call. No COMMIT here: psql reconnect mode commits on disconnect;
-- apply_batch commits after each step when run as top-level CALL.
CREATE OR REPLACE PROCEDURE listings_field_promote_step(
  p_batch_size bigint,
  p_phase text,
  OUT p_updated int
)
LANGUAGE plpgsql
AS $$
BEGIN
  p_updated := 0;

  WITH batch AS (
    SELECT l.id
    FROM listings l
    WHERE (
      (p_phase = 'primary' AND listings_row_needs_field_promote_primary(l))
      OR (p_phase = 'scalars' AND listings_row_needs_field_promote_scalars(l))
    )
    ORDER BY l.id
    LIMIT p_batch_size
  )
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
        backfill_promote_boolean(custom_fields, raw_data, 'HeatingYN')
      ),

      cooling_yn = COALESCE(
        cooling_yn,
        backfill_promote_boolean(custom_fields, raw_data, 'CoolingYN')
      ),

      carport_yn = COALESCE(
        carport_yn,
        backfill_promote_boolean(custom_fields, raw_data, 'CarportYN')
      ),

      attached_garage_yn = COALESCE(
        attached_garage_yn,
        backfill_promote_boolean(custom_fields, raw_data, 'AttachedGarageYN')
      ),

      -- ────────────────────────────────────────────────────────────────────
      -- IDX / INTERNET BOOLEAN FIELDS (compliance gates)
      -- Present in ~190k rows across both datasets; IDXParticipationYN is
      -- Stellar-only (NULL for beaches rows after strip — that is correct).
      -- IDXOfficeParticipationYN has 0 rows today; column added for forward compat.
      -- ────────────────────────────────────────────────────────────────────

      internet_consumer_comment_yn = COALESCE(
        internet_consumer_comment_yn,
        backfill_promote_boolean(custom_fields, raw_data, 'InternetConsumerCommentYN')
      ),

      internet_address_display_yn = COALESCE(
        internet_address_display_yn,
        backfill_promote_boolean(custom_fields, raw_data, 'InternetAddressDisplayYN')
      ),

      internet_entire_listing_display_yn = COALESCE(
        internet_entire_listing_display_yn,
        backfill_promote_boolean(custom_fields, raw_data, 'InternetEntireListingDisplayYN')
      ),

      internet_automated_valuation_display_yn = COALESCE(
        internet_automated_valuation_display_yn,
        backfill_promote_boolean(custom_fields, raw_data, 'InternetAutomatedValuationDisplayYN')
      ),

      idx_participation_yn = COALESCE(
        idx_participation_yn,
        backfill_promote_boolean(custom_fields, raw_data, 'IDXParticipationYN')
      ),

      idx_office_participation_yn = COALESCE(
        idx_office_participation_yn,
        backfill_promote_boolean(custom_fields, raw_data, 'IDXOfficeParticipationYN')
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
    FROM batch
    WHERE listings.id = batch.id;

  GET DIAGNOSTICS p_updated = ROW_COUNT;
  RAISE NOTICE 'phase % step: updated % rows', p_phase, p_updated;
END;
$$;

-- psql-friendly entry (plain CALL cannot omit OUT args on listings_field_promote_step).
CREATE OR REPLACE PROCEDURE listings_field_promote_step_call(p_batch_size bigint, p_phase text)
LANGUAGE plpgsql
AS $$
DECLARE
  v_updated int;
BEGIN
  CALL listings_field_promote_step(p_batch_size, p_phase, v_updated);
END;
$$;

-- Shared UPDATE body for primary and scalar phases (single long-lived connection).
CREATE OR REPLACE PROCEDURE listings_field_promote_apply_batch(p_batch_size bigint, p_phase text)
LANGUAGE plpgsql
AS $$
DECLARE
  v_batch int;
  v_total bigint := 0;
  v_iter  int := 0;
BEGIN
  LOOP
    v_iter := v_iter + 1;
    CALL listings_field_promote_step(p_batch_size, p_phase, v_batch);
    v_total := v_total + v_batch;
    RAISE NOTICE 'phase % batch %: updated % rows (phase total %)', p_phase, v_iter, v_batch, v_total;
    COMMIT;
    EXIT WHEN v_batch = 0;
  END LOOP;

  RAISE NOTICE 'phase % finished: % rows in % batches', p_phase, v_total, v_iter;
END;
$$;

CREATE OR REPLACE PROCEDURE run_listings_field_promote_backfill(p_batch_size bigint DEFAULT 2500)
LANGUAGE plpgsql
AS $$
DECLARE
  v_has_primary boolean;
  v_has_scalars boolean;
BEGIN
  IF p_batch_size IS NULL OR p_batch_size < 1 THEN
    RAISE EXCEPTION 'p_batch_size must be >= 1';
  END IF;

  -- Fast EXISTS checks only (do not COUNT(*) — full scan times out in DataGrip JDBC).
  SELECT EXISTS (
    SELECT 1 FROM listings l WHERE listings_row_needs_field_promote_primary(l) LIMIT 1
  ) INTO v_has_primary;

  SELECT EXISTS (
    SELECT 1 FROM listings l WHERE listings_row_needs_field_promote_scalars(l) LIMIT 1
  ) INTO v_has_scalars;

  RAISE NOTICE 'listings_field_promote_backfill: primary=%, scalars=%, batch_size=%',
    v_has_primary, v_has_scalars, p_batch_size;

  IF NOT v_has_primary AND NOT v_has_scalars THEN
    RAISE NOTICE 'No rows need backfill. Re-run fast EXISTS verify; deploy Go sync if typed cols are still NULL.';
    RETURN;
  END IF;

  IF v_has_primary THEN
    RAISE NOTICE 'starting phase primary (promote + strip custom_fields and raw_data IDX keys)';
    CALL listings_field_promote_apply_batch(p_batch_size, 'primary');
  END IF;

  IF v_has_scalars THEN
    RAISE NOTICE 'starting phase scalars (null typed columns with values still in raw_data)';
    CALL listings_field_promote_apply_batch(p_batch_size, 'scalars');
  END IF;

  RAISE NOTICE 'listings_field_promote_backfill: all phases complete';
END;
$$;

-- After installing functions/procedures above, run manually (prefer psql over DataGrip for CALL):
--   SET statement_timeout = 0;
--   CALL run_listings_field_promote_backfill(500);

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
