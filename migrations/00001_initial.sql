-- +goose Up
-- Consolidated schema (fresh DB only): core IDX tables, PostGIS listings mirror,
-- listing_sync_cursors (last_modification_timestamp), replica_pages,
-- listings payload split (raw_data + media/unit/room/open_house JSONB + custom_fields overflow),
-- single modification_timestamp per row (dataset_slug resolves upstream field at sync). No Telescope/Pulse.

CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    email_verified_at TIMESTAMP NULL,
    password VARCHAR(255) NOT NULL,
    two_factor_secret TEXT NULL,
    two_factor_recovery_codes TEXT NULL,
    two_factor_confirmed_at TIMESTAMP NULL,
    remember_token VARCHAR(100) NULL,
    widget_embed_site_key VARCHAR(64) NULL UNIQUE,
    widget_palette JSONB NULL,
    mls_id VARCHAR(255) NULL,
    mls_email VARCHAR(255) NULL,
    assigned_mls_datasets JSONB NULL,
    mls_membership_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    mls_membership_verified_at TIMESTAMP NULL,
    mls_membership_next_reverify_at TIMESTAMP NULL,
    mls_membership_last_error TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE dashboard_sessions (
    id VARCHAR(128) PRIMARY KEY,
    payload BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX dashboard_sessions_expires_at_idx ON dashboard_sessions(expires_at);

CREATE TABLE jobs (
    id BIGSERIAL PRIMARY KEY,
    queue VARCHAR(255) NOT NULL,
    payload TEXT NOT NULL,
    attempts SMALLINT NOT NULL,
    reserved_at INT NULL,
    available_at INT NOT NULL,
    created_at INT NOT NULL
);
CREATE INDEX jobs_queue_index ON jobs(queue);

CREATE TABLE job_batches (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    total_jobs INT NOT NULL,
    pending_jobs INT NOT NULL,
    failed_jobs INT NOT NULL,
    failed_job_ids TEXT NOT NULL,
    options TEXT NULL,
    cancelled_at INT NULL,
    created_at INT NOT NULL,
    finished_at INT NULL
);

CREATE TABLE failed_jobs (
    id BIGSERIAL PRIMARY KEY,
    uuid VARCHAR(255) NOT NULL UNIQUE,
    connection TEXT NOT NULL,
    queue TEXT NOT NULL,
    payload TEXT NOT NULL,
    exception TEXT NOT NULL,
    failed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE personal_access_tokens (
    id BIGSERIAL PRIMARY KEY,
    tokenable_type VARCHAR(255) NOT NULL,
    tokenable_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    token VARCHAR(64) NOT NULL UNIQUE,
    abilities TEXT NULL,
    last_used_at TIMESTAMP NULL,
    expires_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX personal_access_tokens_tokenable_index ON personal_access_tokens(tokenable_type, tokenable_id);
CREATE INDEX personal_access_tokens_expires_at_index ON personal_access_tokens(expires_at);

CREATE TABLE user_invitations (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    invited_by BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    expires_at TIMESTAMP NOT NULL,
    accepted_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX user_invitations_email_index ON user_invitations(email);

CREATE TABLE domains (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    domain_slug VARCHAR(255) NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    mls_dataset VARCHAR(255) NULL,
    allowed_mls_datasets JSONB NULL,
    verification_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    verification_method VARCHAR(32) NULL,
    txt_verification_name VARCHAR(255) NULL,
    txt_verification_value VARCHAR(255) NULL,
    txt_verified_at TIMESTAMP NULL,
    verification_checked_at TIMESTAMP NULL,
    verification_metadata JSONB NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE mls_search_cache (
    partition_key VARCHAR(255) NOT NULL,
    fingerprint VARCHAR(64) NOT NULL,
    compressed_data BYTEA NOT NULL,
    last_updated TIMESTAMP NOT NULL,
    PRIMARY KEY (partition_key, fingerprint)
);

CREATE TABLE listings_cache (
    domain_slug VARCHAR(255) NOT NULL,
    feed_code VARCHAR(64) NOT NULL,
    listing_key VARCHAR(191) NOT NULL,
    standard_status VARCHAR(64) NOT NULL,
    close_date DATE NULL,
    latitude DOUBLE PRECISION NULL,
    longitude DOUBLE PRECISION NULL,
    close_price NUMERIC(14, 2) NULL,
    compressed_payload BYTEA NOT NULL,
    first_cached_at TIMESTAMPTZ NOT NULL,
    last_refreshed_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (domain_slug, feed_code, listing_key)
);
CREATE INDEX listings_cache_domain_feed_refreshed_idx ON listings_cache(domain_slug, feed_code, last_refreshed_at);
CREATE INDEX listings_cache_closed_scope_idx ON listings_cache(domain_slug, feed_code, close_date DESC)
    WHERE LOWER(TRIM(standard_status)) = 'closed'
      AND latitude IS NOT NULL AND longitude IS NOT NULL;

CREATE TABLE mls_proxy_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    logged_at TIMESTAMP NOT NULL DEFAULT NOW(),
    domain_slug VARCHAR(255) NULL,
    token_name VARCHAR(255) NULL,
    request_type VARCHAR(255) NOT NULL,
    listing_count INT NULL,
    cache_hit VARCHAR(8) NULL,
    ip_address VARCHAR(45) NULL,
    user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX mls_proxy_audit_logs_logged_at_index ON mls_proxy_audit_logs(logged_at);

CREATE TABLE gis_cache (
    id BIGSERIAL PRIMARY KEY,
    query_hash VARCHAR(64) NOT NULL UNIQUE,
    geojson TEXT NOT NULL,
    county VARCHAR(48) NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    source_used VARCHAR(96) NOT NULL,
    source_generation BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX gis_cache_expires_at_index ON gis_cache(expires_at);

CREATE TABLE gis_source_states (
    source_key VARCHAR(96) PRIMARY KEY,
    fingerprint VARCHAR(128) NULL,
    generation BIGINT NOT NULL DEFAULT 0,
    last_checked_at TIMESTAMPTZ NULL,
    last_changed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO gis_source_states (source_key) VALUES
    ('fdot_admin_boundaries')
ON CONFLICT DO NOTHING;

CREATE TABLE gis_parcels (
    id                  BIGSERIAL PRIMARY KEY,
    parcel_id           TEXT NOT NULL,
    source_key          TEXT NOT NULL,
    county              TEXT NOT NULL,
    geometry            GEOMETRY(MultiPolygon, 4326) NOT NULL,
    properties          JSONB NOT NULL DEFAULT '{}',
    site_address        TEXT,
    owner_name          TEXT,
    city                TEXT,
    zip_code            TEXT,
    just_value          NUMERIC(15,2),
    assessed_value      NUMERIC(15,2),
    land_value          NUMERIC(15,2),
    living_area_sqft    NUMERIC(12,2),
    year_built          INTEGER,
    acres               NUMERIC(12,4),
    land_use_code       TEXT,
    last_sale_price     NUMERIC(15,2),
    last_sale_date      TIMESTAMPTZ,
    last_synced_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_generation   INTEGER NOT NULL,
    source_fingerprint  TEXT,
    UNIQUE (parcel_id, source_key),
    CONSTRAINT gis_parcels_county_slug CHECK (county ~ '^[a-z0-9-]+$')
);

CREATE TABLE gis_counties (
    id                  BIGSERIAL PRIMARY KEY,
    county_name         TEXT NOT NULL,
    county_slug         TEXT NOT NULL,
    fips_code           TEXT,
    mls_stellar         BOOLEAN NOT NULL DEFAULT FALSE,
    mls_beaches         BOOLEAN NOT NULL DEFAULT FALSE,
    source_key          TEXT NOT NULL DEFAULT 'fdot_admin_boundaries',
    geometry            GEOMETRY(MultiPolygon, 4326) NOT NULL,
    properties          JSONB NOT NULL DEFAULT '{}',
    last_synced_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_generation   INTEGER NOT NULL,
    source_fingerprint  TEXT,
    UNIQUE (county_slug),
    UNIQUE (fips_code)
);

CREATE TABLE gis_cities (
    id                  BIGSERIAL PRIMARY KEY,
    city_name           TEXT NOT NULL,
    county              TEXT,
    source_key          TEXT NOT NULL DEFAULT 'fdot_admin_boundaries',
    geometry            GEOMETRY(MultiPolygon, 4326) NOT NULL,
    properties          JSONB NOT NULL DEFAULT '{}',
    last_synced_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_generation   INTEGER NOT NULL,
    source_fingerprint  TEXT,
    UNIQUE (city_name, county)
);

CREATE TABLE gis_zips (
    id                  BIGSERIAL PRIMARY KEY,
    zip_code            TEXT NOT NULL,
    source_key          TEXT NOT NULL DEFAULT 'fdot_admin_boundaries',
    geometry            GEOMETRY(MultiPolygon, 4326) NOT NULL,
    properties          JSONB NOT NULL DEFAULT '{}',
    last_synced_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_generation   INTEGER NOT NULL,
    source_fingerprint  TEXT,
    UNIQUE (zip_code)
);

CREATE TABLE gis_parcel_sources (
    source_key       TEXT PRIMARY KEY,
    county_slug      TEXT NOT NULL,
    query_url        TEXT NOT NULL,
    sync_mode        TEXT NOT NULL DEFAULT 'bbox',
    arcgis_where     TEXT,
    bbox_west        DOUBLE PRECISION,
    bbox_south       DOUBLE PRECISION,
    bbox_east        DOUBLE PRECISION,
    bbox_north       DOUBLE PRECISION,
    http_timeout_sec INTEGER,
    page_size        INTEGER,
    mls_feed         TEXT NOT NULL,
    enabled          BOOLEAN NOT NULL DEFAULT TRUE,
    priority         SMALLINT NOT NULL DEFAULT 100,
    notes            TEXT,
    last_synced_at   TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT gis_parcel_sources_county_slug CHECK (county_slug ~ '^[a-z0-9-]+$'),
    CONSTRAINT gis_parcel_sources_sync_mode CHECK (sync_mode IN ('bbox', 'paginate', 'where_filter', 'shapefile'))
);

CREATE INDEX idx_gis_parcels_geometry ON gis_parcels USING GIST (geometry);
CREATE INDEX idx_gis_cities_geometry ON gis_cities USING GIST (geometry);
CREATE INDEX idx_gis_counties_geometry ON gis_counties USING GIST (geometry);
CREATE INDEX idx_gis_zips_geometry ON gis_zips USING GIST (geometry);

CREATE INDEX idx_gis_parcels_county ON gis_parcels (county);
CREATE INDEX idx_gis_parcels_source_generation ON gis_parcels (source_key, source_generation);
CREATE INDEX idx_gis_parcels_county_source_gen ON gis_parcels (county, source_key, source_generation);
CREATE INDEX idx_gis_parcels_last_synced_at ON gis_parcels (last_synced_at);

CREATE INDEX idx_gis_cities_county ON gis_cities (county);
CREATE INDEX idx_gis_cities_source_generation ON gis_cities (source_key, source_generation);

CREATE INDEX idx_gis_counties_source_generation ON gis_counties (source_key, source_generation);
CREATE INDEX idx_gis_counties_mls_stellar ON gis_counties (mls_stellar) WHERE mls_stellar = TRUE;
CREATE INDEX idx_gis_counties_mls_beaches ON gis_counties (mls_beaches) WHERE mls_beaches = TRUE;

CREATE INDEX idx_gis_zips_source_generation ON gis_zips (source_key, source_generation);

CREATE INDEX idx_gis_parcel_sources_county ON gis_parcel_sources (county_slug);
CREATE INDEX idx_gis_parcel_sources_enabled ON gis_parcel_sources (enabled) WHERE enabled = TRUE;

CREATE TABLE crypto_price_snapshots (
    id BIGSERIAL PRIMARY KEY,
    asset_key VARCHAR(32) NOT NULL,
    vs_currency VARCHAR(8) NOT NULL,
    price NUMERIC(24, 12) NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (asset_key, vs_currency)
);

CREATE TABLE listings (
    id BIGSERIAL PRIMARY KEY,
    dataset_slug VARCHAR(64) NOT NULL,
    listing_key VARCHAR(255) NOT NULL,
    mls_listing_id VARCHAR(128) NULL,
    standard_status VARCHAR(50) NULL,
    list_price NUMERIC(14, 2) NOT NULL,
    bedrooms_total SMALLINT NULL,
    bathrooms_total_decimal NUMERIC(5, 2) NULL,
    living_area NUMERIC(12, 2) NULL,
    lot_size_acres NUMERIC(12, 4) NULL,
    year_built SMALLINT NULL,
    stories_total SMALLINT NULL,
    city VARCHAR(120) NULL,
    county_or_parish VARCHAR(120) NULL,
    postal_code VARCHAR(20) NULL,
    state_or_province VARCHAR(30) NULL,
    property_type VARCHAR(80) NULL,
    property_sub_type VARCHAR(80) NULL,
    on_market_date DATE NULL,
    close_date DATE NULL,
    modification_timestamp TIMESTAMPTZ NULL,
    price_change_timestamp TIMESTAMPTZ NULL,
    previous_list_price NUMERIC(14, 2) NULL,
    flood_zone_code VARCHAR(80) NULL,
    fema_flood_zone_code VARCHAR(80) NULL,
    flood_zone_sfha_tf VARCHAR(32) NULL,
    flood_zone_updated_at TIMESTAMPTZ NULL,
    flood_zone_raw JSONB NULL,
    estimated_total_monthly_fees NUMERIC(14, 2) NULL,
    low_risk_flood_zone_yn BOOLEAN NOT NULL DEFAULT FALSE,
    latitude DOUBLE PRECISION NULL,
    longitude DOUBLE PRECISION NULL,
    waterfront_yn BOOLEAN NULL,
    pool_private_yn BOOLEAN NULL,
    dock_yn BOOLEAN NULL,
    new_construction_yn BOOLEAN NULL,
    garage_yn BOOLEAN NULL,
    association_yn BOOLEAN NULL,
    spa_yn BOOLEAN NULL,
    fireplace_yn BOOLEAN NULL,
    senior_community_yn BOOLEAN NULL,
    subdivision_name VARCHAR(160) NULL,
    elementary_school VARCHAR(160) NULL,
    middle_or_junior_school VARCHAR(160) NULL,
    high_school VARCHAR(160) NULL,
    special_listing_conditions JSONB NULL,
    raw_data JSONB NULL,
    media JSONB NULL,
    unit JSONB NULL,
    room JSONB NULL,
    open_house JSONB NULL,
    custom_fields JSONB NULL,
    coordinates geography(Point, 4326) NULL,
    street_number VARCHAR(40) NULL,
    street_name VARCHAR(160) NULL,
    list_agent_mls_id VARCHAR(64) NULL,
    list_office_mls_id VARCHAR(64) NULL,
    mirror_persisted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (dataset_slug, listing_key)
);

CREATE INDEX listings_ap_geom_gix ON listings USING GIST (coordinates)
    WHERE coordinates IS NOT NULL AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending');
CREATE INDEX listings_ap_mod_brin ON listings USING BRIN (modification_timestamp)
    WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending');
CREATE INDEX listings_ap_mirror_persisted_brin ON listings USING BRIN (mirror_persisted_at)
    WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending');
CREATE INDEX listings_ap_ds_price_idx ON listings (dataset_slug, list_price)
    WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending');
CREATE INDEX listings_ap_ds_beds_idx ON listings (dataset_slug, bedrooms_total)
    WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending');
CREATE INDEX listings_ap_ds_mod_ts_idx ON listings (dataset_slug, modification_timestamp DESC)
    WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending') AND modification_timestamp IS NOT NULL;
CREATE INDEX listings_ap_flood_zone_idx ON listings (dataset_slug, flood_zone_code)
    WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending') AND flood_zone_code IS NOT NULL;
CREATE INDEX listings_ap_low_risk_flood_idx ON listings (dataset_slug, low_risk_flood_zone_yn)
    WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
      AND low_risk_flood_zone_yn = TRUE;
CREATE INDEX listings_flood_zone_updated_at_idx ON listings (flood_zone_updated_at)
    WHERE latitude IS NOT NULL AND longitude IS NOT NULL;
CREATE INDEX listings_ap_fema_flood_zone_idx ON listings (dataset_slug, fema_flood_zone_code)
    WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
      AND fema_flood_zone_code IS NOT NULL;

CREATE TABLE listing_sync_cursors (
    dataset_slug VARCHAR(64) PRIMARY KEY,
    last_modification_timestamp TIMESTAMPTZ NULL,
    incremental_window_end TIMESTAMPTZ NULL,
    replication_next_url TEXT NULL,
    replication_in_progress BOOLEAN NOT NULL DEFAULT FALSE,
    last_sync_finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE replica_pages (
    id BIGSERIAL PRIMARY KEY,
    provider VARCHAR(16) NOT NULL DEFAULT 'bridge',
    dataset_slug VARCHAR(64) NOT NULL,
    mode VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    compressed_payload TEXT NULL,
    row_count INT NOT NULL DEFAULT 0,
    fetch_url TEXT NULL, -- OData resource URL (provider-agnostic; was bridge_url)
    upstream_url TEXT NULL, -- full HTTP GET URL including query string
    odata_query JSONB NULL,
    batch_id UUID NULL,
    fetched_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX replica_pages_status_processed_idx ON replica_pages(status, processed_at);
CREATE INDEX replica_pages_dataset_status_idx ON replica_pages(dataset_slug, status);
CREATE INDEX replica_pages_provider_dataset_status_idx ON replica_pages(provider, dataset_slug, status);

-- +goose Down
DROP TABLE IF EXISTS replica_pages;
DROP TABLE IF EXISTS listing_sync_cursors;
DROP TABLE IF EXISTS listings;
DROP TABLE IF EXISTS crypto_price_snapshots;
DROP TABLE IF EXISTS gis_parcel_sources;
DROP TABLE IF EXISTS gis_zips;
DROP TABLE IF EXISTS gis_cities;
DROP TABLE IF EXISTS gis_counties;
DROP TABLE IF EXISTS gis_parcels;
DROP TABLE IF EXISTS gis_source_states;
DROP TABLE IF EXISTS gis_cache;
DROP TABLE IF EXISTS mls_proxy_audit_logs;
DROP TABLE IF EXISTS listings_cache;
DROP TABLE IF EXISTS mls_search_cache;
DROP TABLE IF EXISTS domains;
DROP TABLE IF EXISTS user_invitations;
DROP TABLE IF EXISTS personal_access_tokens;
DROP TABLE IF EXISTS failed_jobs;
DROP TABLE IF EXISTS job_batches;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS dashboard_sessions;
DROP TABLE IF EXISTS users;
