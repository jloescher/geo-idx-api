-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_gis_cities_city_name_trgm ON gis_cities
  USING gin (lower(city_name) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_gis_cities_county_city_trgm ON gis_cities
  USING gin (county gin_trgm_ops, lower(city_name) gin_trgm_ops)
  WHERE county IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_gis_counties_county_name_trgm ON gis_counties
  USING gin (lower(county_name) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_gis_counties_county_slug_prefix ON gis_counties
  (lower(county_slug) text_pattern_ops);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_gis_counties_county_slug_prefix;
DROP INDEX IF EXISTS idx_gis_counties_county_name_trgm;
DROP INDEX IF EXISTS idx_gis_cities_county_city_trgm;
DROP INDEX IF EXISTS idx_gis_cities_city_name_trgm;
-- +goose StatementEnd
