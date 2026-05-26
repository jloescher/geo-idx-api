package fema

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ListingCoord is a listing id with coordinates for NFHL point queries.
type ListingCoord struct {
	ID        int64
	Latitude  float64
	Longitude float64
}

// FEMAUpdate is one row of FEMA enrichment to persist.
type FEMAUpdate struct {
	ID                   int64
	FEMAFloodZoneCode    *string
	FloodZoneSFHA_TF     *string
	FloodZoneRaw         json.RawMessage
	FloodZoneUpdatedAt   time.Time
	LowRiskFloodZoneYN   bool
}

// Repository reads stale listings and batch-updates FEMA columns.
type Repository struct {
	pool         *pgxpool.Pool
	staleInterval time.Duration
}

// NewRepository creates a FEMA listings repository.
func NewRepository(pool *pgxpool.Pool, staleDays int) *Repository {
	days := staleDays
	if days <= 0 {
		days = 30
	}
	return &Repository{
		pool:          pool,
		staleInterval: time.Duration(days) * 24 * time.Hour,
	}
}

// SelectStaleForEnrichment returns listings needing FEMA refresh after cursorID.
func (r *Repository) SelectStaleForEnrichment(ctx context.Context, cursorID int64, limit int, datasetSlug string) ([]ListingCoord, error) {
	if limit <= 0 {
		limit = 2000
	}
	query := `
		SELECT id, latitude::float8, longitude::float8
		FROM listings
		WHERE id > $1
		  AND latitude IS NOT NULL AND longitude IS NOT NULL
		  AND latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180
		  AND (flood_zone_updated_at IS NULL OR flood_zone_updated_at < NOW() - $2::interval)
	`
	args := []any{cursorID, r.staleInterval}
	if datasetSlug != "" {
		query += ` AND dataset_slug = $3`
		args = append(args, datasetSlug)
	}
	query += ` ORDER BY id LIMIT ` + fmt.Sprintf("%d", limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ListingCoord
	for rows.Next() {
		var row ListingCoord
		if err := rows.Scan(&row.ID, &row.Latitude, &row.Longitude); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// CountStale returns how many listings would be selected for enrichment.
func (r *Repository) CountStale(ctx context.Context, datasetSlug string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM listings
		WHERE latitude IS NOT NULL AND longitude IS NOT NULL
		  AND latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180
		  AND (flood_zone_updated_at IS NULL OR flood_zone_updated_at < NOW() - $1::interval)
	`
	args := []any{r.staleInterval}
	if datasetSlug != "" {
		query += ` AND dataset_slug = $2`
		args = append(args, datasetSlug)
	}
	var n int64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&n)
	return n, err
}

// BatchUpdateFEMA applies FEMA flood fields in a single transaction.
func (r *Repository) BatchUpdateFEMA(ctx context.Context, updates []FEMAUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, u := range updates {
		var raw any
		if len(u.FloodZoneRaw) > 0 {
			raw = u.FloodZoneRaw
		}
		_, err := tx.Exec(ctx, `
			UPDATE listings SET
				fema_flood_zone_code = $2,
				flood_zone_sfha_tf = $3,
				flood_zone_raw = $4,
				flood_zone_updated_at = $5,
				low_risk_flood_zone_yn = $6,
				updated_at = NOW()
			WHERE id = $1
		`, u.ID, u.FEMAFloodZoneCode, u.FloodZoneSFHA_TF, raw, u.FloodZoneUpdatedAt, u.LowRiskFloodZoneYN)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// HasActiveFloodEnrichJob returns true if a pending/reserved fema.flood_enrich% job exists.
func HasActiveFloodEnrichJob(ctx context.Context, pool *pgxpool.Pool, table string) (bool, error) {
	if table == "" {
		table = "jobs"
	}
	var exists bool
	err := pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %s
			WHERE payload::jsonb->>'type' IN ('fema.flood_enrich_kickoff', 'fema.flood_enrich_batch')
			  AND (
			    reserved_at IS NULL
			    OR reserved_at > EXTRACT(EPOCH FROM NOW())::bigint - 7200
			  )
		)
	`, table)).Scan(&exists)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return exists, nil
}
