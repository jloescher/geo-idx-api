package fema

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	FailureReasonNoNFHLFeature     = "no_nfhl_feature"
	FailureReasonRequestError      = "request_error"
	FailureReasonInsufficientCoords = "insufficient_coords"
)

// ListingCoord is a listing id with coordinates for NFHL point queries.
type ListingCoord struct {
	ID        int64
	Latitude  float64
	Longitude float64
}

// FEMAUpdate is one row of FEMA enrichment to persist.
type FEMAUpdate struct {
	ID                 int64
	Outcome            string // success, no_nfhl_feature
	FEMAFloodZoneCode    *string
	FloodZoneSFHA_TF     *string
	FloodZoneRaw         json.RawMessage
	FloodZoneUpdatedAt   time.Time
	LowRiskFloodZoneYN   bool
	FEMAFailureReason    *string
}

// NullWithCoordsSample is a dashboard drill-down row for FEMA gaps.
type NullWithCoordsSample struct {
	ID                int64
	DatasetSlug       string
	ListingKey        string
	FEMAFailureReason *string
	FEMAAttemptCount  int
	FloodZoneUpdatedAt *time.Time
	FEMAAttemptedAt   *time.Time
}

// OutcomeCount groups listings by fema_failure_reason (or never_attempted).
type OutcomeCount struct {
	Reason string
	Count  int64
}

// DatasetOutcomeCount is per-dataset FEMA breakdown.
type DatasetOutcomeCount struct {
	DatasetSlug string
	Reason      string
	Count       int64
}

// Repository reads stale listings and batch-updates FEMA columns.
type Repository struct {
	pool          *pgxpool.Pool
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

// CountNullFEMAWithCoords counts active/pending listings with coords but no FEMA zone code.
func (r *Repository) CountNullFEMAWithCoords(ctx context.Context, datasetSlug string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM listings
		WHERE latitude IS NOT NULL AND longitude IS NOT NULL
		  AND fema_flood_zone_code IS NULL
		  AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
	`
	args := []any{}
	if datasetSlug != "" {
		query += ` AND dataset_slug = $1`
		args = append(args, datasetSlug)
	}
	var n int64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&n)
	return n, err
}

// CountByOutcome groups null-with-coords rows by failure reason (empty = never attempted).
func (r *Repository) CountByOutcome(ctx context.Context, datasetSlug string) ([]OutcomeCount, error) {
	query := `
		SELECT COALESCE(fema_failure_reason, 'never_attempted') AS reason, COUNT(*)
		FROM listings
		WHERE latitude IS NOT NULL AND longitude IS NOT NULL
		  AND fema_flood_zone_code IS NULL
		  AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
	`
	args := []any{}
	if datasetSlug != "" {
		query += ` AND dataset_slug = $1`
		args = append(args, datasetSlug)
	}
	query += ` GROUP BY 1 ORDER BY 2 DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OutcomeCount
	for rows.Next() {
		var row OutcomeCount
		if err := rows.Scan(&row.Reason, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// CountByDatasetOutcome returns per-dataset FEMA null breakdown.
func (r *Repository) CountByDatasetOutcome(ctx context.Context) ([]DatasetOutcomeCount, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT dataset_slug, COALESCE(fema_failure_reason, 'never_attempted') AS reason, COUNT(*)
		FROM listings
		WHERE latitude IS NOT NULL AND longitude IS NOT NULL
		  AND fema_flood_zone_code IS NULL
		  AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
		GROUP BY 1, 2
		ORDER BY 1, 3 DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DatasetOutcomeCount
	for rows.Next() {
		var row DatasetOutcomeCount
		if err := rows.Scan(&row.DatasetSlug, &row.Reason, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ListNullWithCoordsSamples returns recent listings missing FEMA codes for dashboard drill-down.
func (r *Repository) ListNullWithCoordsSamples(ctx context.Context, limit int) ([]NullWithCoordsSample, error) {
	if limit <= 0 {
		limit = 25
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, dataset_slug, listing_key, fema_failure_reason, fema_attempt_count,
		       flood_zone_updated_at, fema_attempted_at
		FROM listings
		WHERE latitude IS NOT NULL AND longitude IS NOT NULL
		  AND fema_flood_zone_code IS NULL
		  AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
		ORDER BY COALESCE(fema_attempted_at, updated_at) DESC NULLS LAST, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NullWithCoordsSample
	for rows.Next() {
		var row NullWithCoordsSample
		if err := rows.Scan(
			&row.ID, &row.DatasetSlug, &row.ListingKey, &row.FEMAFailureReason,
			&row.FEMAAttemptCount, &row.FloodZoneUpdatedAt, &row.FEMAAttemptedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// BatchUpdateFEMA applies successful or no-feature FEMA outcomes in a single transaction.
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
		reason := u.FEMAFailureReason
		_, err := tx.Exec(ctx, `
			UPDATE listings SET
				fema_flood_zone_code = $2,
				flood_zone_sfha_tf = $3,
				flood_zone_raw = $4,
				flood_zone_updated_at = $5,
				low_risk_flood_zone_yn = $6,
				fema_attempted_at = $5,
				fema_failure_reason = $7,
				fema_failed_at = NULL,
				fema_attempt_count = COALESCE(fema_attempt_count, 0) + 1,
				updated_at = NOW()
			WHERE id = $1
		`, u.ID, u.FEMAFloodZoneCode, u.FloodZoneSFHA_TF, raw, u.FloodZoneUpdatedAt, u.LowRiskFloodZoneYN, reason)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// BatchMarkRequestError records transient NFHL query failures without advancing flood_zone_updated_at.
func (r *Repository) BatchMarkRequestError(ctx context.Context, ids []int64, attemptedAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	reason := FailureReasonRequestError
	_, err := r.pool.Exec(ctx, `
		UPDATE listings SET
			fema_attempted_at = $2,
			fema_failed_at = $2,
			fema_failure_reason = $3,
			fema_attempt_count = COALESCE(fema_attempt_count, 0) + 1,
			updated_at = NOW()
		WHERE id = ANY($1)
	`, ids, attemptedAt, reason)
	return err
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
