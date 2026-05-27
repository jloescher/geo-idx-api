package gisrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const fdotSourceKey = "fdot_admin_boundaries"

// BulkUpsertCities upserts city boundary rows.
func (r *Repository) BulkUpsertCities(ctx context.Context, rows []CityRow) error {
	if len(rows) == 0 {
		return nil
	}
	for i := 0; i < len(rows); i += 100 {
		end := i + 100
		if end > len(rows) {
			end = len(rows)
		}
		if err := r.upsertCityBatch(ctx, rows[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) upsertCityBatch(ctx context.Context, rows []CityRow) error {
	var b strings.Builder
	b.WriteString(`INSERT INTO gis_cities (
		city_name, county, source_key, geometry, properties,
		last_synced_at, source_generation, source_fingerprint
	) VALUES `)
	args := make([]any, 0, len(rows)*8)
	n := 1
	for i, row := range rows {
		if i > 0 {
			b.WriteString(", ")
		}
		sk := row.SourceKey
		if sk == "" {
			sk = fdotSourceKey
		}
		b.WriteString(fmt.Sprintf(`(
			$%d, $%d, $%d,
			ST_Multi(ST_SetSRID(ST_GeomFromGeoJSON($%d), 4326)),
			$%d::jsonb, NOW(), $%d, $%d
		)`, n, n+1, n+2, n+3, n+4, n+5, n+6))
		props := row.Properties
		if len(props) == 0 {
			props = json.RawMessage(`{}`)
		}
		args = append(args, row.CityName, row.County, sk, row.GeometryJSON, string(props), row.SourceGeneration, row.SourceFingerprint)
		n += 7
	}
	b.WriteString(` ON CONFLICT (city_name, county) DO UPDATE SET
		source_key = EXCLUDED.source_key,
		geometry = EXCLUDED.geometry,
		properties = EXCLUDED.properties,
		last_synced_at = NOW(),
		source_generation = EXCLUDED.source_generation,
		source_fingerprint = EXCLUDED.source_fingerprint`)
	_, err := r.db.Pool.Exec(ctx, b.String(), args...)
	return err
}

// BulkUpsertCounties upserts county boundary rows.
func (r *Repository) BulkUpsertCounties(ctx context.Context, rows []CountyRow) error {
	if len(rows) == 0 {
		return nil
	}
	for i := 0; i < len(rows); i += 100 {
		end := i + 100
		if end > len(rows) {
			end = len(rows)
		}
		if err := r.upsertCountyBatch(ctx, rows[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) upsertCountyBatch(ctx context.Context, rows []CountyRow) error {
	var b strings.Builder
	b.WriteString(`INSERT INTO gis_counties (
		county_name, county_slug, fips_code, mls_stellar, mls_beaches, source_key, geometry, properties,
		last_synced_at, source_generation, source_fingerprint
	) VALUES `)
	args := make([]any, 0, len(rows)*11)
	n := 1
	for i, row := range rows {
		if i > 0 {
			b.WriteString(", ")
		}
		sk := row.SourceKey
		if sk == "" {
			sk = fdotSourceKey
		}
		slug := row.CountySlug
		if slug == "" {
			slug = row.CountyName
		}
		b.WriteString(fmt.Sprintf(`(
			$%d, $%d, $%d, $%d, $%d, $%d,
			ST_Multi(ST_SetSRID(ST_GeomFromGeoJSON($%d), 4326)),
			$%d::jsonb, NOW(), $%d, $%d
		)`, n, n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8, n+9))
		props := row.Properties
		if len(props) == 0 {
			props = json.RawMessage(`{}`)
		}
		args = append(args, row.CountyName, slug, row.FIPSCode, row.MLSStellar, row.MLSBeaches, sk, row.GeometryJSON, string(props), row.SourceGeneration, row.SourceFingerprint)
		n += 10
	}
	b.WriteString(` ON CONFLICT (county_slug) DO UPDATE SET
		county_name = EXCLUDED.county_name,
		fips_code = EXCLUDED.fips_code,
		mls_stellar = EXCLUDED.mls_stellar,
		mls_beaches = EXCLUDED.mls_beaches,
		source_key = EXCLUDED.source_key,
		geometry = EXCLUDED.geometry,
		properties = EXCLUDED.properties,
		last_synced_at = NOW(),
		source_generation = EXCLUDED.source_generation,
		source_fingerprint = EXCLUDED.source_fingerprint`)
	_, err := r.db.Pool.Exec(ctx, b.String(), args...)
	return err
}

// BulkUpsertZips upserts zip boundary rows.
func (r *Repository) BulkUpsertZips(ctx context.Context, rows []ZipRow) error {
	if len(rows) == 0 {
		return nil
	}
	for i := 0; i < len(rows); i += 100 {
		end := i + 100
		if end > len(rows) {
			end = len(rows)
		}
		if err := r.upsertZipBatch(ctx, rows[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) upsertZipBatch(ctx context.Context, rows []ZipRow) error {
	var b strings.Builder
	b.WriteString(`INSERT INTO gis_zips (
		zip_code, source_key, geometry, properties,
		last_synced_at, source_generation, source_fingerprint
	) VALUES `)
	args := make([]any, 0, len(rows)*7)
	n := 1
	for i, row := range rows {
		if i > 0 {
			b.WriteString(", ")
		}
		sk := row.SourceKey
		if sk == "" {
			sk = fdotSourceKey
		}
		b.WriteString(fmt.Sprintf(`(
			$%d, $%d,
			ST_Multi(ST_SetSRID(ST_GeomFromGeoJSON($%d), 4326)),
			$%d::jsonb, NOW(), $%d, $%d
		)`, n, n+1, n+2, n+3, n+4, n+5))
		props := row.Properties
		if len(props) == 0 {
			props = json.RawMessage(`{}`)
		}
		args = append(args, row.ZipCode, sk, row.GeometryJSON, string(props), row.SourceGeneration, row.SourceFingerprint)
		n += 6
	}
	b.WriteString(` ON CONFLICT (zip_code) DO UPDATE SET
		source_key = EXCLUDED.source_key,
		geometry = EXCLUDED.geometry,
		properties = EXCLUDED.properties,
		last_synced_at = NOW(),
		source_generation = EXCLUDED.source_generation,
		source_fingerprint = EXCLUDED.source_fingerprint`)
	_, err := r.db.Pool.Exec(ctx, b.String(), args...)
	return err
}

// DeleteStaleCities removes superseded city rows.
func (r *Repository) DeleteStaleCities(ctx context.Context, sourceKey string, currentGen int) (int64, error) {
	return r.deleteStale(ctx, "gis_cities", sourceKey, currentGen)
}

// DeleteStaleCounties removes superseded county rows.
func (r *Repository) DeleteStaleCounties(ctx context.Context, sourceKey string, currentGen int) (int64, error) {
	return r.deleteStale(ctx, "gis_counties", sourceKey, currentGen)
}

// DeleteStaleZips removes superseded zip rows.
func (r *Repository) DeleteStaleZips(ctx context.Context, sourceKey string, currentGen int) (int64, error) {
	return r.deleteStale(ctx, "gis_zips", sourceKey, currentGen)
}

func (r *Repository) deleteStale(ctx context.Context, table, sourceKey string, currentGen int) (int64, error) {
	q := fmt.Sprintf(`DELETE FROM %s WHERE source_key = $1 AND source_generation < $2`, table)
	tag, err := r.db.Pool.Exec(ctx, q, sourceKey, currentGen)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// HasCitiesInBBox checks for city boundaries in bbox.
func (r *Repository) HasCitiesInBBox(ctx context.Context, west, south, east, north float64) (bool, error) {
	return r.hasInBBox(ctx, "gis_cities", west, south, east, north)
}

// HasCountiesInBBox checks for county boundaries in bbox.
func (r *Repository) HasCountiesInBBox(ctx context.Context, west, south, east, north float64) (bool, error) {
	return r.hasInBBox(ctx, "gis_counties", west, south, east, north)
}

// HasZipsInBBox checks for zip boundaries in bbox.
func (r *Repository) HasZipsInBBox(ctx context.Context, west, south, east, north float64) (bool, error) {
	return r.hasInBBox(ctx, "gis_zips", west, south, east, north)
}

func (r *Repository) hasInBBox(ctx context.Context, table string, west, south, east, north float64) (bool, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return false, err
	}
	q := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %s
			WHERE geometry && %s AND ST_Intersects(geometry, %s)
		)`, table, bboxEnvelopeSQL, bboxEnvelopeSQL)
	var exists bool
	err = pool.QueryRow(ctx, q, west, south, east, north, west, south, east, north).Scan(&exists)
	return exists, err
}

// QueryCitiesByBBox returns city features in bbox.
func (r *Repository) QueryCitiesByBBox(ctx context.Context, west, south, east, north float64, limit int) ([]FeatureResult, error) {
	return r.queryByBBox(ctx, "gis_cities", west, south, east, north, limit)
}

// QueryCountiesByBBox returns county features in bbox.
func (r *Repository) QueryCountiesByBBox(ctx context.Context, west, south, east, north float64, limit int) ([]FeatureResult, error) {
	return r.queryByBBox(ctx, "gis_counties", west, south, east, north, limit)
}

// QueryZipsByBBox returns zip features in bbox.
func (r *Repository) QueryZipsByBBox(ctx context.Context, west, south, east, north float64, limit int) ([]FeatureResult, error) {
	return r.queryByBBox(ctx, "gis_zips", west, south, east, north, limit)
}

func (r *Repository) queryByBBox(ctx context.Context, table string, west, south, east, north float64, limit int) ([]FeatureResult, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	q := fmt.Sprintf(`
		SELECT ST_AsGeoJSON(geometry)::text, properties
		FROM %s
		WHERE geometry && %s AND ST_Intersects(geometry, %s)
		LIMIT $5
	`, table, bboxEnvelopeSQL, bboxEnvelopeSQL)
	rows, err := pool.Query(ctx, q, west, south, east, north, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FeatureResult
	for rows.Next() {
		var fr FeatureResult
		if err := rows.Scan(&fr.GeometryJSON, &fr.Properties); err != nil {
			return nil, err
		}
		out = append(out, fr)
	}
	return out, rows.Err()
}

// CountBoundaries returns counts for boundary tables.
func (r *Repository) CountBoundaries(ctx context.Context) (cities, counties, zips int64, err error) {
	err = r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM gis_cities`).Scan(&cities)
	if err != nil {
		return
	}
	err = r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM gis_counties`).Scan(&counties)
	if err != nil {
		return
	}
	err = r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM gis_zips`).Scan(&zips)
	return
}

// CountCitiesMissingCounty returns cities with no county slug assigned.
func (r *Repository) CountCitiesMissingCounty(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM gis_cities WHERE county IS NULL`).Scan(&n)
	return n, err
}

// BackfillMissingCityCounties expands city|county pairs for all generations (replaces NULL-county UPDATE-only path).
func (r *Repository) BackfillMissingCityCounties(ctx context.Context) (int64, error) {
	inserted, deleted, err := r.ExpandAllCityCountyPairs(ctx)
	if err != nil {
		return 0, err
	}
	return inserted + deleted, nil
}

// BackfillCityCounties assigns county slugs to cities via largest-area intersection with county boundaries.
// Deprecated for sync: use ExpandCityCountyPairs for multi-county rows.
func (r *Repository) BackfillCityCounties(ctx context.Context, generation int) (int64, error) {
	inserted, deleted, err := r.ExpandCityCountyPairs(ctx, generation)
	if err != nil {
		return 0, err
	}
	return inserted + deleted, nil
}

// ExpandCityCountyPairs upserts one gis_cities row per (city_name, county_slug) for the generation,
// using spatial intersection or nearest-county fallback, and deletes stale county rows.
func (r *Repository) ExpandCityCountyPairs(ctx context.Context, generation int) (inserted int64, deleted int64, err error) {
	const expandSQL = `
		WITH cities AS (
			SELECT DISTINCT ON (city_name)
				city_name, source_generation, geometry, properties, source_key, source_fingerprint
			FROM gis_cities
			WHERE source_generation = $1
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
			WHERE c.source_generation = $1
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
	`
	var ins, del int64
	if err := r.db.Pool.QueryRow(ctx, expandSQL, generation).Scan(&ins, &del); err != nil {
		return 0, 0, err
	}
	return ins, del, nil
}

// ExpandAllCityCountyPairs runs ExpandCityCountyPairs for every distinct source_generation in gis_cities.
func (r *Repository) ExpandAllCityCountyPairs(ctx context.Context) (inserted int64, deleted int64, err error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT DISTINCT source_generation FROM gis_cities ORDER BY 1`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var gen int
		if err := rows.Scan(&gen); err != nil {
			return inserted, deleted, err
		}
		ins, del, err := r.ExpandCityCountyPairs(ctx, gen)
		if err != nil {
			return inserted, deleted, err
		}
		inserted += ins
		deleted += del
	}
	return inserted, deleted, rows.Err()
}

// QueryCountySlugsByBBox returns county slugs intersecting the envelope.
func (r *Repository) QueryCountySlugsByBBox(ctx context.Context, west, south, east, north float64) ([]string, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT county_slug FROM gis_counties
		WHERE geometry && `+bboxEnvelopeSQL+` AND ST_Intersects(geometry, `+bboxEnvelopeSQL+`)
		ORDER BY county_slug
	`, west, south, east, north, west, south, east, north)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		out = append(out, slug)
	}
	return out, rows.Err()
}

// CityAutocompleteRow is one city|county autocomplete suggestion.
type CityAutocompleteRow struct {
	CityName   string
	County     string
	CountySlug string
	CountyName string
}

// CountyAutocompleteRow is one county autocomplete suggestion.
type CountyAutocompleteRow struct {
	CountyName string
	CountySlug string
}

// AutocompleteCities returns city|county pairs matching q (county NOT NULL).
func (r *Repository) AutocompleteCities(ctx context.Context, q string, limit int) ([]CityAutocompleteRow, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	qLower := strings.ToLower(q)
	pattern := "%" + qLower + "%"
	rows, err := pool.Query(ctx, `
		SELECT c.city_name, c.county,
		       COALESCE(co.county_slug, c.county),
		       COALESCE(co.county_name, c.county)
		FROM gis_cities c
		LEFT JOIN gis_counties co ON lower(co.county_slug) = lower(c.county)
		WHERE c.county IS NOT NULL
		  AND (
		    lower(c.city_name) LIKE $1
		    OR similarity(lower(c.city_name), $2) > 0.15
		  )
		ORDER BY similarity(lower(c.city_name), $2) DESC, c.city_name, c.county
		LIMIT $3
	`, pattern, qLower, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CityAutocompleteRow
	for rows.Next() {
		var row CityAutocompleteRow
		if err := rows.Scan(&row.CityName, &row.County, &row.CountySlug, &row.CountyName); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// AutocompleteCounties returns counties matching q by name or slug.
func (r *Repository) AutocompleteCounties(ctx context.Context, q string, limit int) ([]CountyAutocompleteRow, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	qLower := strings.ToLower(q)
	pattern := "%" + qLower + "%"
	rows, err := pool.Query(ctx, `
		SELECT county_name, county_slug
		FROM gis_counties
		WHERE lower(county_name) LIKE $1
		   OR lower(county_slug) LIKE $1
		   OR similarity(lower(county_name), $2) > 0.15
		   OR similarity(lower(county_slug), $2) > 0.15
		ORDER BY GREATEST(
			similarity(lower(county_name), $2),
			similarity(lower(county_slug), $2)
		) DESC, county_name
		LIMIT $3
	`, pattern, qLower, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CountyAutocompleteRow
	for rows.Next() {
		var row CountyAutocompleteRow
		if err := rows.Scan(&row.CountyName, &row.CountySlug); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// IsPersistentEmpty returns true when all persistent GIS tables are empty.
func (r *Repository) IsPersistentEmpty(ctx context.Context) (bool, error) {
	parcels, err := r.CountParcels(ctx, "")
	if err != nil {
		return false, err
	}
	cities, counties, zips, err := r.CountBoundaries(ctx)
	if err != nil {
		return false, err
	}
	return parcels == 0 && cities == 0 && counties == 0 && zips == 0, nil
}
