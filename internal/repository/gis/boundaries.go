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

// BackfillMissingCityCounties assigns county slugs to cities missing county via spatial join
// within matching source_generation pairs.
func (r *Repository) BackfillMissingCityCounties(ctx context.Context) (int64, error) {
	tag, err := r.db.Pool.Exec(ctx, `
		WITH ranked AS (
			SELECT c.id,
			       co.county_slug,
			       ROW_NUMBER() OVER (
			           PARTITION BY c.id
			           ORDER BY ST_Area(ST_Intersection(c.geometry, co.geometry)) DESC
			       ) AS rn
			FROM gis_cities c
			JOIN gis_counties co ON ST_Intersects(c.geometry, co.geometry)
			WHERE c.county IS NULL AND c.source_generation = co.source_generation
		)
		UPDATE gis_cities c
		SET county = ranked.county_slug
		FROM ranked
		WHERE c.id = ranked.id AND ranked.rn = 1
	`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// BackfillCityCounties assigns county slugs to cities via largest-area intersection with county boundaries.
func (r *Repository) BackfillCityCounties(ctx context.Context, generation int) (int64, error) {
	tag, err := r.db.Pool.Exec(ctx, `
		WITH ranked AS (
			SELECT c.id,
			       co.county_slug,
			       ROW_NUMBER() OVER (
			           PARTITION BY c.id
			           ORDER BY ST_Area(ST_Intersection(c.geometry, co.geometry)) DESC
			       ) AS rn
			FROM gis_cities c
			JOIN gis_counties co ON ST_Intersects(c.geometry, co.geometry)
			WHERE c.source_generation = $1 AND co.source_generation = $1
		)
		UPDATE gis_cities c
		SET county = ranked.county_slug
		FROM ranked
		WHERE c.id = ranked.id AND ranked.rn = 1
	`, generation)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
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
