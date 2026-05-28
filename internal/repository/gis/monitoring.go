package gisrepo

import (
	"context"
	"time"
)

// SourceStateRow is a GIS upstream source generation snapshot.
type SourceStateRow struct {
	SourceKey           string     `json:"source_key"`
	CountySlug          string     `json:"county_slug,omitempty"`
	SyncMode            string     `json:"sync_mode,omitempty"`
	Enabled             bool       `json:"enabled"`
	Generation          int64      `json:"generation"`
	LastCheckedAt       *time.Time `json:"last_checked_at,omitempty"`
	LastProbeAt         *time.Time `json:"last_probe_at,omitempty"`
	LastProbeOK         *bool      `json:"last_probe_ok,omitempty"`
	LastProbeHTTPStatus *int       `json:"last_probe_http_status,omitempty"`
	LastProbeError      string     `json:"last_probe_error,omitempty"`
	APIStatus           string     `json:"api_status"` // reachable | unreachable | unknown
	MaxGeneration       int64      `json:"max_generation"`
	MaxSyncedAt         *time.Time `json:"max_synced_at,omitempty"`
	ParcelCount         int64      `json:"parcel_count"`
	ActiveSyncJob       bool       `json:"active_sync_job"`
	Status              string     `json:"status"` // healthy | stale | unknown
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
		SELECT s.source_key,
		       COALESCE(ps.county_slug, ''),
		       COALESCE(ps.sync_mode, ''),
		       COALESCE(ps.enabled, TRUE),
		       s.generation,
		       s.last_checked_at,
		       s.last_probe_at,
		       s.last_probe_ok,
		       s.last_probe_http_status,
		       COALESCE(s.last_probe_error, ''),
		       COALESCE(p.max_gen, 0),
		       p.max_synced,
		       COALESCE(p.cnt, 0),
		       EXISTS (
		           SELECT 1 FROM jobs j
		           WHERE j.payload::jsonb->>'type' = 'gis.parcel_sync_page'
		             AND j.payload::jsonb->'args'->>'source_key' = s.source_key
		             AND (
		               j.reserved_at IS NULL
		               OR j.reserved_at > EXTRACT(EPOCH FROM NOW())::bigint - 7200
		             )
		       ) AS active_sync
		FROM gis_source_states s
		LEFT JOIN gis_parcel_sources ps ON ps.source_key = s.source_key
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
		if err := rows.Scan(
			&row.SourceKey, &row.CountySlug, &row.SyncMode, &row.Enabled,
			&row.Generation, &row.LastCheckedAt,
			&row.LastProbeAt, &row.LastProbeOK, &row.LastProbeHTTPStatus, &row.LastProbeError,
			&row.MaxGeneration, &row.MaxSyncedAt, &row.ParcelCount, &row.ActiveSyncJob,
		); err != nil {
			return nil, err
		}
		row.APIStatus = apiStatusFromProbe(row.LastProbeOK, row.LastProbeAt)
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

func apiStatusFromProbe(ok *bool, probedAt *time.Time) string {
	if ok == nil || probedAt == nil {
		return "unknown"
	}
	if *ok {
		return "reachable"
	}
	return "unreachable"
}

// UpdateProbeResult persists the latest ArcGIS metadata probe outcome.
func (r *Repository) UpdateProbeResult(ctx context.Context, sourceKey string, ok bool, httpStatus int, errMsg string) error {
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE gis_source_states
		SET last_probe_at = NOW(),
		    last_probe_ok = $2,
		    last_probe_http_status = $3,
		    last_probe_error = NULLIF($4, ''),
		    last_checked_at = NOW(),
		    updated_at = NOW()
		WHERE source_key = $1
	`, sourceKey, ok, httpStatus, errMsg)
	return err
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

// MaxCitySyncedAt returns the newest city boundary sync timestamp.
func (r *Repository) MaxCitySyncedAt(ctx context.Context) (*time.Time, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	var t *time.Time
	err = pool.QueryRow(ctx, `SELECT MAX(last_synced_at) FROM gis_cities`).Scan(&t)
	return t, err
}

// MaxCountySyncedAt returns the newest county boundary sync timestamp.
func (r *Repository) MaxCountySyncedAt(ctx context.Context) (*time.Time, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	var t *time.Time
	err = pool.QueryRow(ctx, `SELECT MAX(last_synced_at) FROM gis_counties`).Scan(&t)
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
