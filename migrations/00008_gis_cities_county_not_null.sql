-- +goose Up
-- +goose StatementBegin
-- Run docs/scripts/gis_cities_county_expand.sql on primary before applying in production.
ALTER TABLE gis_cities ALTER COLUMN county SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE gis_cities ALTER COLUMN county DROP NOT NULL;
-- +goose StatementEnd
