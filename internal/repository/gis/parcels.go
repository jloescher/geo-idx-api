package gisrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const bboxEnvelopeSQL = `ST_MakeEnvelope($1, $2, $3, $4, 4326)`

// BulkUpsertParcels inserts or updates parcel rows in batches.
// Revenue impact: pre-loaded parcels eliminate per-request ArcGIS latency on map pan/zoom.
func (r *Repository) BulkUpsertParcels(ctx context.Context, rows []ParcelRow) error {
	if len(rows) == 0 {
		return nil
	}
	for i := 0; i < len(rows); i += 100 {
		end := i + 100
		if end > len(rows) {
			end = len(rows)
		}
		if err := r.upsertParcelBatch(ctx, rows[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) upsertParcelBatch(ctx context.Context, rows []ParcelRow) error {
	var b strings.Builder
	b.WriteString(`INSERT INTO gis_parcels (
		parcel_id, source_key, county, geometry, properties,
		site_address, owner_name, city, zip_code,
		just_value, assessed_value, land_value, living_area_sqft,
		year_built, acres, land_use_code, last_sale_price, last_sale_date,
		last_synced_at, source_generation, source_fingerprint
	) VALUES `)
	args := make([]any, 0, len(rows)*21)
	n := 1
	for i, row := range rows {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf(`(
			$%d, $%d, $%d,
			ST_Multi(ST_SetSRID(ST_GeomFromGeoJSON($%d), 4326)),
			$%d::jsonb,
			$%d, $%d, $%d, $%d,
			$%d, $%d, $%d, $%d,
			$%d, $%d, $%d, $%d, $%d,
			NOW(), $%d, $%d
		)`, n, n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8,
			n+9, n+10, n+11, n+12, n+13, n+14, n+15, n+16, n+17, n+18, n+19))
		props := row.Properties
		if len(props) == 0 {
			props = json.RawMessage(`{}`)
		}
		args = append(args,
			row.ParcelID, row.SourceKey, row.County, row.GeometryJSON, string(props),
			row.SiteAddress, row.OwnerName, row.City, row.ZipCode,
			row.JustValue, row.AssessedValue, row.LandValue, row.LivingAreaSqft,
			row.YearBuilt, row.Acres, row.LandUseCode, row.LastSalePrice, row.LastSaleDate,
			row.SourceGeneration, row.SourceFingerprint,
		)
		n += 20
	}
	b.WriteString(` ON CONFLICT (parcel_id, source_key) DO UPDATE SET
		county = EXCLUDED.county,
		geometry = EXCLUDED.geometry,
		properties = EXCLUDED.properties,
		site_address = EXCLUDED.site_address,
		owner_name = EXCLUDED.owner_name,
		city = EXCLUDED.city,
		zip_code = EXCLUDED.zip_code,
		just_value = EXCLUDED.just_value,
		assessed_value = EXCLUDED.assessed_value,
		land_value = EXCLUDED.land_value,
		living_area_sqft = EXCLUDED.living_area_sqft,
		year_built = EXCLUDED.year_built,
		acres = EXCLUDED.acres,
		land_use_code = EXCLUDED.land_use_code,
		last_sale_price = EXCLUDED.last_sale_price,
		last_sale_date = EXCLUDED.last_sale_date,
		last_synced_at = NOW(),
		source_generation = EXCLUDED.source_generation,
		source_fingerprint = EXCLUDED.source_fingerprint`)
	_, err := r.db.Pool.Exec(ctx, b.String(), args...)
	return err
}

// DeleteStaleParcels removes rows superseded by a successful generation swap.
func (r *Repository) DeleteStaleParcels(ctx context.Context, sourceKey string, currentGen int) (int64, error) {
	tag, err := r.db.Pool.Exec(ctx, `
		DELETE FROM gis_parcels
		WHERE source_key = $1 AND source_generation < $2
	`, sourceKey, currentGen)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// CountParcels returns total parcel count (optionally filtered by county).
func (r *Repository) CountParcels(ctx context.Context, county string) (int64, error) {
	var count int64
	var err error
	if county == "" {
		err = r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM gis_parcels`).Scan(&count)
	} else {
		err = r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM gis_parcels WHERE county = $1`, county).Scan(&count)
	}
	return count, err
}

// HasParcelsInBBox checks whether any parcel intersects the envelope.
func (r *Repository) HasParcelsInBBox(ctx context.Context, west, south, east, north float64, counties []string) (bool, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return false, err
	}
	q := fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM gis_parcels
			WHERE geometry && %s AND ST_Intersects(geometry, %s)
	`, bboxEnvelopeSQL, bboxEnvelopeSQL)
	args := []any{west, south, east, north, west, south, east, north}
	if len(counties) > 0 {
		q += ` AND county = ANY($5)`
		args = append(args, counties)
	}
	q += `)`
	var exists bool
	err = pool.QueryRow(ctx, q, args...).Scan(&exists)
	return exists, err
}

// QueryParcelsByBBox returns GeoJSON geometry + properties for parcels in bbox.
func (r *Repository) QueryParcelsByBBox(ctx context.Context, west, south, east, north float64, counties []string, limit int) ([]FeatureResult, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	q := fmt.Sprintf(`
		SELECT ST_AsGeoJSON(geometry)::text, properties
		FROM gis_parcels
		WHERE geometry && %s AND ST_Intersects(geometry, %s)
	`, bboxEnvelopeSQL, bboxEnvelopeSQL)
	args := []any{west, south, east, north, west, south, east, north}
	if len(counties) > 0 {
		q += ` AND county = ANY($5)`
		args = append(args, counties)
	}
	q += fmt.Sprintf(` LIMIT $%d`, len(args)+1)
	args = append(args, limit)

	rows, err := pool.Query(ctx, q, args...)
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

// SourceGeneration reads generation from gis_source_states.
func (r *Repository) SourceGeneration(ctx context.Context, sourceKey string) (int64, error) {
	var gen int64
	err := r.db.Pool.QueryRow(ctx, `SELECT generation FROM gis_source_states WHERE source_key = $1`, sourceKey).Scan(&gen)
	return gen, err
}

// BumpSourceGeneration increments generation for a source key.
func (r *Repository) BumpSourceGeneration(ctx context.Context, sourceKey string) (int64, error) {
	var gen int64
	err := r.db.Pool.QueryRow(ctx, `
		UPDATE gis_source_states
		SET generation = generation + 1, last_changed_at = NOW(), updated_at = NOW()
		WHERE source_key = $1
		RETURNING generation
	`, sourceKey).Scan(&gen)
	return gen, err
}

// SetSourceGeneration sets generation to an explicit value after sync.
func (r *Repository) SetSourceGeneration(ctx context.Context, sourceKey string, gen int64, fingerprint string) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE gis_source_states
		SET generation = $2, fingerprint = $3, last_changed_at = NOW(), last_checked_at = NOW(), updated_at = NOW()
		WHERE source_key = $1
	`, sourceKey, gen, fingerprint)
	return err
}
