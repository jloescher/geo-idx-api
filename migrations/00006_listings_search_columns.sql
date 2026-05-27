-- +goose Up
-- +goose StatementBegin

ALTER TABLE listings
  ADD COLUMN IF NOT EXISTS garage_spaces NUMERIC(5, 2) NULL,
  ADD COLUMN IF NOT EXISTS mls_area_major VARCHAR(400) NULL,
  ADD COLUMN IF NOT EXISTS days_on_market INTEGER NULL,
  ADD COLUMN IF NOT EXISTS tax_annual_amount NUMERIC(14, 2) NULL,
  ADD COLUMN IF NOT EXISTS heating_yn BOOLEAN NULL,
  ADD COLUMN IF NOT EXISTS cooling_yn BOOLEAN NULL,
  ADD COLUMN IF NOT EXISTS carport_yn BOOLEAN NULL,
  ADD COLUMN IF NOT EXISTS attached_garage_yn BOOLEAN NULL,
  ADD COLUMN IF NOT EXISTS internet_consumer_comment_yn BOOLEAN NULL,
  ADD COLUMN IF NOT EXISTS internet_address_display_yn BOOLEAN NULL,
  ADD COLUMN IF NOT EXISTS internet_entire_listing_display_yn BOOLEAN NULL,
  ADD COLUMN IF NOT EXISTS internet_automated_valuation_display_yn BOOLEAN NULL,
  ADD COLUMN IF NOT EXISTS idx_participation_yn BOOLEAN NULL,
  ADD COLUMN IF NOT EXISTS idx_office_participation_yn BOOLEAN NULL,
  ADD COLUMN IF NOT EXISTS unparsed_address VARCHAR(500) NULL,
  ADD COLUMN IF NOT EXISTS public_remarks TEXT NULL,
  ADD COLUMN IF NOT EXISTS geocoded_at TIMESTAMPTZ NULL,
  ADD COLUMN IF NOT EXISTS geocode_query TEXT NULL;

CREATE INDEX IF NOT EXISTS listings_ap_ds_city_idx ON listings (dataset_slug, lower(trim(city)))
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND city IS NOT NULL AND trim(city) <> '';

CREATE INDEX IF NOT EXISTS listings_ap_ds_county_lower_idx ON listings (dataset_slug, lower(trim(county_or_parish)))
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND county_or_parish IS NOT NULL AND trim(county_or_parish) <> '';

CREATE INDEX IF NOT EXISTS listings_ap_ds_postal_idx ON listings (dataset_slug, postal_code)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND postal_code IS NOT NULL;

CREATE INDEX IF NOT EXISTS listings_ap_ds_property_type_idx ON listings (dataset_slug, lower(property_type))
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND property_type IS NOT NULL;

CREATE INDEX IF NOT EXISTS listings_ap_ds_property_sub_type_idx ON listings (dataset_slug, lower(property_sub_type))
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND property_sub_type IS NOT NULL;

CREATE INDEX IF NOT EXISTS listings_ap_ds_living_area_idx ON listings (dataset_slug, living_area)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND living_area IS NOT NULL;

CREATE INDEX IF NOT EXISTS listings_ap_ds_baths_idx ON listings (dataset_slug, bathrooms_total_decimal)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND bathrooms_total_decimal IS NOT NULL;

CREATE INDEX IF NOT EXISTS listings_ap_ds_lot_acres_idx ON listings (dataset_slug, lot_size_acres)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND lot_size_acres IS NOT NULL;

CREATE INDEX IF NOT EXISTS listings_ap_ds_year_built_idx ON listings (dataset_slug, year_built)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND year_built IS NOT NULL;

CREATE INDEX IF NOT EXISTS listings_ap_ds_monthly_fees_idx ON listings (dataset_slug, estimated_total_monthly_fees)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND estimated_total_monthly_fees IS NOT NULL;

CREATE INDEX IF NOT EXISTS listings_ap_price_change_ts_idx ON listings (dataset_slug, price_change_timestamp DESC)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND price_change_timestamp IS NOT NULL;

CREATE INDEX IF NOT EXISTS listings_ap_pool_idx ON listings (dataset_slug)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND pool_private_yn = TRUE;

CREATE INDEX IF NOT EXISTS listings_ap_waterfront_idx ON listings (dataset_slug)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND waterfront_yn = TRUE;

CREATE INDEX IF NOT EXISTS listings_ap_garage_spaces_idx ON listings (dataset_slug, garage_spaces)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND garage_spaces IS NOT NULL;

CREATE INDEX IF NOT EXISTS listings_ap_mls_area_major_idx ON listings (dataset_slug, lower(trim(mls_area_major)))
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND mls_area_major IS NOT NULL AND trim(mls_area_major) <> '';

CREATE INDEX IF NOT EXISTS listings_ap_days_on_market_idx ON listings (dataset_slug, days_on_market)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND days_on_market IS NOT NULL;

CREATE INDEX IF NOT EXISTS listings_ap_tax_annual_idx ON listings (dataset_slug, tax_annual_amount)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND tax_annual_amount IS NOT NULL;

CREATE INDEX IF NOT EXISTS listings_ap_heating_idx ON listings (dataset_slug)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND heating_yn = TRUE;

CREATE INDEX IF NOT EXISTS listings_ap_cooling_idx ON listings (dataset_slug)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND cooling_yn = TRUE;

CREATE INDEX IF NOT EXISTS listings_ap_carport_idx ON listings (dataset_slug)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND carport_yn = TRUE;

CREATE INDEX IF NOT EXISTS listings_ap_attached_garage_idx ON listings (dataset_slug)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND attached_garage_yn = TRUE;

CREATE INDEX IF NOT EXISTS listings_ap_ds_mod_ts_compliant_idx ON listings (dataset_slug, modification_timestamp DESC)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND modification_timestamp IS NOT NULL
    AND internet_entire_listing_display_yn IS NOT FALSE
    AND (idx_participation_yn IS NOT FALSE OR idx_participation_yn IS NULL);

CREATE INDEX IF NOT EXISTS listings_ap_public_remarks_fts_idx ON listings
  USING gin (to_tsvector('english', COALESCE(public_remarks, '')))
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND public_remarks IS NOT NULL AND trim(public_remarks) <> '';

CREATE INDEX IF NOT EXISTS listings_ap_geocode_pending_idx ON listings (id)
  WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
    AND internet_address_display_yn IS TRUE
    AND (latitude IS NULL OR longitude IS NULL)
    AND (
      (unparsed_address IS NOT NULL AND trim(unparsed_address) <> '')
      OR (street_number IS NOT NULL AND city IS NOT NULL)
    );

CREATE INDEX IF NOT EXISTS idx_gis_cities_city_name_prefix ON gis_cities (lower(city_name) text_pattern_ops);
CREATE INDEX IF NOT EXISTS idx_gis_cities_county_city_prefix ON gis_cities (county, lower(city_name) text_pattern_ops)
  WHERE county IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_gis_counties_county_name_prefix ON gis_counties (lower(county_name) text_pattern_ops);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_gis_counties_county_name_prefix;
DROP INDEX IF EXISTS idx_gis_cities_county_city_prefix;
DROP INDEX IF EXISTS idx_gis_cities_city_name_prefix;
DROP INDEX IF EXISTS listings_ap_geocode_pending_idx;
DROP INDEX IF EXISTS listings_ap_public_remarks_fts_idx;
DROP INDEX IF EXISTS listings_ap_ds_mod_ts_compliant_idx;
DROP INDEX IF EXISTS listings_ap_attached_garage_idx;
DROP INDEX IF EXISTS listings_ap_carport_idx;
DROP INDEX IF EXISTS listings_ap_cooling_idx;
DROP INDEX IF EXISTS listings_ap_heating_idx;
DROP INDEX IF EXISTS listings_ap_tax_annual_idx;
DROP INDEX IF EXISTS listings_ap_days_on_market_idx;
DROP INDEX IF EXISTS listings_ap_mls_area_major_idx;
DROP INDEX IF EXISTS listings_ap_garage_spaces_idx;
DROP INDEX IF EXISTS listings_ap_waterfront_idx;
DROP INDEX IF EXISTS listings_ap_pool_idx;
DROP INDEX IF EXISTS listings_ap_price_change_ts_idx;
DROP INDEX IF EXISTS listings_ap_ds_monthly_fees_idx;
DROP INDEX IF EXISTS listings_ap_ds_year_built_idx;
DROP INDEX IF EXISTS listings_ap_ds_lot_acres_idx;
DROP INDEX IF EXISTS listings_ap_ds_baths_idx;
DROP INDEX IF EXISTS listings_ap_ds_living_area_idx;
DROP INDEX IF EXISTS listings_ap_ds_property_sub_type_idx;
DROP INDEX IF EXISTS listings_ap_ds_property_type_idx;
DROP INDEX IF EXISTS listings_ap_ds_postal_idx;
DROP INDEX IF EXISTS listings_ap_ds_county_lower_idx;
DROP INDEX IF EXISTS listings_ap_ds_city_idx;

ALTER TABLE listings
  DROP COLUMN IF EXISTS geocode_query,
  DROP COLUMN IF EXISTS geocoded_at,
  DROP COLUMN IF EXISTS public_remarks,
  DROP COLUMN IF EXISTS unparsed_address,
  DROP COLUMN IF EXISTS idx_office_participation_yn,
  DROP COLUMN IF EXISTS idx_participation_yn,
  DROP COLUMN IF EXISTS internet_automated_valuation_display_yn,
  DROP COLUMN IF EXISTS internet_entire_listing_display_yn,
  DROP COLUMN IF EXISTS internet_address_display_yn,
  DROP COLUMN IF EXISTS internet_consumer_comment_yn,
  DROP COLUMN IF EXISTS attached_garage_yn,
  DROP COLUMN IF EXISTS carport_yn,
  DROP COLUMN IF EXISTS cooling_yn,
  DROP COLUMN IF EXISTS heating_yn,
  DROP COLUMN IF EXISTS tax_annual_amount,
  DROP COLUMN IF EXISTS days_on_market,
  DROP COLUMN IF EXISTS mls_area_major,
  DROP COLUMN IF EXISTS garage_spaces;

-- +goose StatementEnd
