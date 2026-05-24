-- +goose Up
CREATE TABLE IF NOT EXISTS gis_parcels (
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
    CONSTRAINT valid_county CHECK (county IN ('pinellas', 'hillsborough', 'statewide'))
);

CREATE TABLE IF NOT EXISTS gis_cities (
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

CREATE TABLE IF NOT EXISTS gis_counties (
    id                  BIGSERIAL PRIMARY KEY,
    county_name         TEXT NOT NULL,
    fips_code           TEXT,
    source_key          TEXT NOT NULL DEFAULT 'fdot_admin_boundaries',
    geometry            GEOMETRY(MultiPolygon, 4326) NOT NULL,
    properties          JSONB NOT NULL DEFAULT '{}',
    last_synced_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_generation   INTEGER NOT NULL,
    source_fingerprint  TEXT,
    UNIQUE (fips_code)
);

CREATE TABLE IF NOT EXISTS gis_zips (
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

CREATE INDEX idx_gis_parcels_geometry ON gis_parcels USING GIST (geometry);
CREATE INDEX idx_gis_cities_geometry   ON gis_cities USING GIST (geometry);
CREATE INDEX idx_gis_counties_geometry ON gis_counties USING GIST (geometry);
CREATE INDEX idx_gis_zips_geometry     ON gis_zips USING GIST (geometry);

CREATE INDEX idx_gis_parcels_county ON gis_parcels (county);
CREATE INDEX idx_gis_parcels_source_generation ON gis_parcels (source_key, source_generation);
CREATE INDEX idx_gis_parcels_last_synced_at ON gis_parcels (last_synced_at);

CREATE INDEX idx_gis_cities_county ON gis_cities (county);
CREATE INDEX idx_gis_cities_source_generation ON gis_cities (source_key, source_generation);

CREATE INDEX idx_gis_counties_source_generation ON gis_counties (source_key, source_generation);

CREATE INDEX idx_gis_zips_source_generation ON gis_zips (source_key, source_generation);

INSERT INTO gis_source_states (source_key) VALUES
    ('fdot_admin_boundaries')
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS gis_zips;
DROP TABLE IF EXISTS gis_counties;
DROP TABLE IF EXISTS gis_cities;
DROP TABLE IF EXISTS gis_parcels;

DELETE FROM gis_source_states WHERE source_key = 'fdot_admin_boundaries';
