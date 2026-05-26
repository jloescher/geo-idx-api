package gisrepo

import (
	"context"
)

// ParcelSourceRow mirrors gis_parcel_sources for ops/monitoring.
type ParcelSourceRow struct {
	SourceKey      string
	CountySlug     string
	QueryURL       string
	SyncMode       string
	ArcGISWhere    *string
	BBoxWest       *float64
	BBoxSouth      *float64
	BBoxEast       *float64
	BBoxNorth      *float64
	HTTPTimeoutSec *int
	PageSize       *int
	MLSFeed        string
	Enabled        bool
	Priority       int
	Notes          *string
}

// EnsureParcelSourceCatalog upserts catalog rows from sync kickoff.
func (r *Repository) EnsureParcelSourceCatalog(ctx context.Context, specs []ParcelSourceRow) error {
	for _, spec := range specs {
		if err := r.upsertParcelSource(ctx, spec); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) upsertParcelSource(ctx context.Context, spec ParcelSourceRow) error {
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO gis_parcel_sources (
			source_key, county_slug, query_url, sync_mode, arcgis_where,
			bbox_west, bbox_south, bbox_east, bbox_north,
			http_timeout_sec, page_size, mls_feed, enabled, priority, notes, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,NOW())
		ON CONFLICT (source_key) DO UPDATE SET
			county_slug = EXCLUDED.county_slug,
			query_url = EXCLUDED.query_url,
			sync_mode = EXCLUDED.sync_mode,
			arcgis_where = EXCLUDED.arcgis_where,
			bbox_west = EXCLUDED.bbox_west,
			bbox_south = EXCLUDED.bbox_south,
			bbox_east = EXCLUDED.bbox_east,
			bbox_north = EXCLUDED.bbox_north,
			http_timeout_sec = EXCLUDED.http_timeout_sec,
			page_size = EXCLUDED.page_size,
			mls_feed = EXCLUDED.mls_feed,
			enabled = EXCLUDED.enabled,
			priority = EXCLUDED.priority,
			notes = EXCLUDED.notes,
			updated_at = NOW()
	`, spec.SourceKey, spec.CountySlug, spec.QueryURL, spec.SyncMode, spec.ArcGISWhere,
		spec.BBoxWest, spec.BBoxSouth, spec.BBoxEast, spec.BBoxNorth,
		spec.HTTPTimeoutSec, spec.PageSize, spec.MLSFeed, spec.Enabled, spec.Priority, spec.Notes)
	return err
}

// TouchParcelSourceSynced updates last_synced_at after a county finalize.
func (r *Repository) TouchParcelSourceSynced(ctx context.Context, sourceKey string) error {
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE gis_parcel_sources SET last_synced_at = NOW(), updated_at = NOW()
		WHERE source_key = $1
	`, sourceKey)
	return err
}

// LoadParcelSources returns enabled parcel sources from the catalog table.
func (r *Repository) LoadParcelSources(ctx context.Context) ([]ParcelSourceRow, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT source_key, county_slug, query_url, sync_mode, arcgis_where,
		       bbox_west, bbox_south, bbox_east, bbox_north,
		       http_timeout_sec, page_size, mls_feed, enabled, priority, notes
		FROM gis_parcel_sources
		WHERE enabled = TRUE
		ORDER BY priority, county_slug
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ParcelSourceRow
	for rows.Next() {
		var row ParcelSourceRow
		if err := rows.Scan(&row.SourceKey, &row.CountySlug, &row.QueryURL, &row.SyncMode, &row.ArcGISWhere,
			&row.BBoxWest, &row.BBoxSouth, &row.BBoxEast, &row.BBoxNorth,
			&row.HTTPTimeoutSec, &row.PageSize, &row.MLSFeed, &row.Enabled, &row.Priority, &row.Notes); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
