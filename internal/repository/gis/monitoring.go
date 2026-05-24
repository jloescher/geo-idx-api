package gisrepo

import (
	"context"
	"time"
)

// SourceStateRow is a GIS upstream source generation snapshot.
type SourceStateRow struct {
	SourceKey     string
	Generation    int64
	LastCheckedAt *time.Time
	MaxGeneration int64
	MaxSyncedAt   *time.Time
	ParcelCount   int64
	Status        string // healthy | stale | unknown
}

// GISCounts holds aggregate GIS table counts.
type GISCounts struct {
	ParcelsTotal  int64
	CitiesTotal   int64
	CountiesTotal int64
	ZipsTotal     int64
}

// MonitoringCounts returns parcel and boundary row totals.
func (r *Repository) MonitoringCounts(ctx context.Context) (GISCounts, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return GISCounts{}, err
	}
	var c GISCounts
	err = pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM gis_parcels),
			(SELECT COUNT(*) FROM gis_cities),
			(SELECT COUNT(*) FROM gis_counties),
			(SELECT COUNT(*) FROM gis_zips)
	`).Scan(&c.ParcelsTotal, &c.CitiesTotal, &c.CountiesTotal, &c.ZipsTotal)
	return c, err
}

// ParcelsByCounty returns parcel counts grouped by county slug.
func (r *Repository) ParcelsByCounty(ctx context.Context) (map[string]int64, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `SELECT county, COUNT(*) FROM gis_parcels GROUP BY county ORDER BY county`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int64)
	for rows.Next() {
		var county string
		var n int64
		if err := rows.Scan(&county, &n); err != nil {
			return nil, err
		}
		out[county] = n
	}
	return out, rows.Err()
}

// ListSourceStates returns GIS source states with parcel sync metadata and health status.
func (r *Repository) ListSourceStates(ctx context.Context) ([]SourceStateRow, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT s.source_key, s.generation, s.last_checked_at,
		       COALESCE(p.max_gen, 0), p.max_synced, COALESCE(p.cnt, 0)
		FROM gis_source_states s
		LEFT JOIN (
			SELECT source_key,
			       MAX(source_generation) AS max_gen,
			       MAX(last_synced_at) AS max_synced,
			       COUNT(*) AS cnt
			FROM gis_parcels
			GROUP BY source_key
		) p ON p.source_key = s.source_key
		ORDER BY s.source_key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	staleCutoff := time.Now().Add(-35 * 24 * time.Hour)
	var out []SourceStateRow
	for rows.Next() {
		var row SourceStateRow
		if err := rows.Scan(&row.SourceKey, &row.Generation, &row.LastCheckedAt,
			&row.MaxGeneration, &row.MaxSyncedAt, &row.ParcelCount); err != nil {
			return nil, err
		}
		row.Status = "unknown"
		if row.MaxSyncedAt != nil {
			if row.MaxSyncedAt.Before(staleCutoff) || (row.ParcelCount > 0 && row.MaxGeneration != row.Generation) {
				row.Status = "stale"
			} else {
				row.Status = "healthy"
			}
		} else if row.ParcelCount == 0 {
			row.Status = "unknown"
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// MaxParcelSyncedAt returns the newest parcel sync timestamp across all sources.
func (r *Repository) MaxParcelSyncedAt(ctx context.Context) (*time.Time, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	var t *time.Time
	err = pool.QueryRow(ctx, `SELECT MAX(last_synced_at) FROM gis_parcels`).Scan(&t)
	return t, err
}

// MaxZipSyncedAt returns the newest zip boundary sync timestamp.
func (r *Repository) MaxZipSyncedAt(ctx context.Context) (*time.Time, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	var t *time.Time
	err = pool.QueryRow(ctx, `SELECT MAX(last_synced_at) FROM gis_zips`).Scan(&t)
	return t, err
}
