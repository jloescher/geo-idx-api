package geocoderepo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PendingRow is a listing that needs coordinate backfill.
type PendingRow struct {
	ID            int64
	DatasetSlug   string
	Unparsed      *string
	City          *string
	State         *string
	Postal        *string
	StreetNumber  *string
	StreetName    *string
}

// Repository selects listings for geocode enrichment.
type Repository struct {
	pool *pgxpool.Pool
}

const selectPendingBaseSQL = `
		SELECT id, dataset_slug, unparsed_address, city, state_or_province, postal_code,
		       street_number, street_name
		FROM listings
		WHERE id > $1
		  AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
		  AND internet_address_display_yn IS TRUE
		  AND geocode_bad_address_yn IS FALSE
		  AND (latitude IS NULL OR longitude IS NULL)
		  AND (
		    (unparsed_address IS NOT NULL AND trim(unparsed_address) <> '')
		    OR (street_number IS NOT NULL AND city IS NOT NULL)
		  )
	`

const countPendingBaseSQL = `
		SELECT COUNT(*)
		FROM listings
		WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
		  AND internet_address_display_yn IS TRUE
		  AND geocode_bad_address_yn IS FALSE
		  AND (latitude IS NULL OR longitude IS NULL)
		  AND (
		    (unparsed_address IS NOT NULL AND trim(unparsed_address) <> '')
		    OR (street_number IS NOT NULL AND city IS NOT NULL)
		  )
	`

const applyCoordsSQL = `
		UPDATE listings SET
			latitude = $2,
			longitude = $3,
			coordinates = ST_SetSRID(ST_MakePoint($3, $2), 4326),
			geocoded_at = NOW(),
			geocode_attempted_at = NOW(),
			geocode_query = $4,
			geocode_bad_address_yn = FALSE,
			geocode_failed_at = NULL,
			geocode_failure_reason = NULL,
			geocode_attempt_count = COALESCE(geocode_attempt_count, 0) + 1,
			updated_at = NOW()
		WHERE id = $1
		  AND (latitude IS NULL OR longitude IS NULL)
	`

const markFailedAttemptSQL = `
		UPDATE listings SET
			geocode_attempted_at = NOW(),
			geocode_failed_at = NOW(),
			geocode_failure_reason = $2,
			geocode_query = NULLIF($3, ''),
			geocode_bad_address_yn = $4,
			geocode_attempt_count = COALESCE(geocode_attempt_count, 0) + 1,
			updated_at = NOW()
		WHERE id = $1
		  AND (latitude IS NULL OR longitude IS NULL)
	`

// NewRepository creates a geocode listings repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// SelectPending returns address-display listings missing coordinates after cursorID.
func (r *Repository) SelectPending(ctx context.Context, cursorID int64, limit int, datasetSlug string) ([]PendingRow, error) {
	if limit <= 0 {
		limit = 200
	}
	query := selectPendingBaseSQL
	args := []any{cursorID}
	if datasetSlug != "" {
		query += ` AND dataset_slug = $2`
		args = append(args, datasetSlug)
	}
	query += ` ORDER BY id LIMIT ` + fmt.Sprintf("%d", limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PendingRow
	for rows.Next() {
		var row PendingRow
		if err := rows.Scan(
			&row.ID, &row.DatasetSlug, &row.Unparsed, &row.City, &row.State, &row.Postal,
			&row.StreetNumber, &row.StreetName,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// CountPending returns how many listings match the geocode job selection.
func (r *Repository) CountPending(ctx context.Context, datasetSlug string) (int64, error) {
	query := countPendingBaseSQL
	args := []any{}
	if datasetSlug != "" {
		query += ` AND dataset_slug = $1`
		args = append(args, datasetSlug)
	}
	var n int64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&n)
	return n, err
}

// ApplyCoords updates latitude, longitude, coordinates, geocoded_at, and geocode_query.
func (r *Repository) ApplyCoords(ctx context.Context, id int64, lat, lng float64, query string) error {
	_, err := r.pool.Exec(ctx, applyCoordsSQL, id, lat, lng, query)
	return err
}

// MarkFailedAttempt records a failed geocode attempt and optional bad-address flag.
func (r *Repository) MarkFailedAttempt(ctx context.Context, id int64, reason string, query string, badAddress bool) error {
	_, err := r.pool.Exec(ctx, markFailedAttemptSQL, id, reason, query, badAddress)
	return err
}

// activeGeocodeJobExistsSQL checks pending/in-flight geocode jobs (jobs table has no finished_at).
const activeGeocodeJobExistsSQL = `
		SELECT EXISTS (
			SELECT 1 FROM %s
			WHERE queue NOT LIKE '%%:failed'
			  AND payload::jsonb->>'type' IN ('mls.geocode_listings_kickoff', 'mls.geocode_listings_batch')
			  AND (
			    reserved_at IS NULL
			    OR reserved_at > EXTRACT(EPOCH FROM NOW())::bigint - 7200
			  )
		)`

// HasActiveGeocodeJob reports whether a geocode enrich job is already queued or in-flight.
func HasActiveGeocodeJob(ctx context.Context, pool *pgxpool.Pool, table string) (bool, error) {
	if table == "" {
		table = "jobs"
	}
	var exists bool
	err := pool.QueryRow(ctx, fmt.Sprintf(activeGeocodeJobExistsSQL, table)).Scan(&exists)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return exists, nil
}

// FailureSample is a dashboard drill-down row for geocode failures.
type FailureSample struct {
	ID                  int64
	DatasetSlug         string
	ListingKey          string
	GeocodeFailureReason *string
	GeocodeAttemptCount int
	GeocodeQuery        *string
	GeocodeFailedAt     *time.Time
}

// OutcomeCount groups listings by geocode_failure_reason.
type OutcomeCount struct {
	Reason string
	Count  int64
}

// DatasetOutcomeCount is per-dataset geocode breakdown.
type DatasetOutcomeCount struct {
	DatasetSlug string
	Reason      string
	Count       int64
}

// CountMissingCoords counts displayable active/pending listings without coordinates.
func (r *Repository) CountMissingCoords(ctx context.Context, datasetSlug string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM listings
		WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
		  AND internet_address_display_yn IS TRUE
		  AND (latitude IS NULL OR longitude IS NULL)
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

// CountBadAddress counts listings permanently skipped by geocode.
func (r *Repository) CountBadAddress(ctx context.Context, datasetSlug string) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM listings
		WHERE geocode_bad_address_yn IS TRUE
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

// CountByFailureReason groups missing-coord listings by failure reason.
func (r *Repository) CountByFailureReason(ctx context.Context, datasetSlug string) ([]OutcomeCount, error) {
	query := `
		SELECT COALESCE(geocode_failure_reason, 'never_attempted') AS reason, COUNT(*)
		FROM listings
		WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
		  AND internet_address_display_yn IS TRUE
		  AND (latitude IS NULL OR longitude IS NULL)
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

// CountByDatasetFailureReason returns per-dataset geocode failure breakdown.
func (r *Repository) CountByDatasetFailureReason(ctx context.Context) ([]DatasetOutcomeCount, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT dataset_slug, COALESCE(geocode_failure_reason, 'never_attempted') AS reason, COUNT(*)
		FROM listings
		WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
		  AND internet_address_display_yn IS TRUE
		  AND (latitude IS NULL OR longitude IS NULL)
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

// CountHighAttemptRequestErrors counts retryable geocode failures with many attempts.
func (r *Repository) CountHighAttemptRequestErrors(ctx context.Context, minAttempts int) (int64, error) {
	if minAttempts <= 0 {
		minAttempts = 5
	}
	var n int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM listings
		WHERE geocode_failure_reason = 'request_error'
		  AND geocode_bad_address_yn IS FALSE
		  AND geocode_attempt_count >= $1
		  AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
	`, minAttempts).Scan(&n)
	return n, err
}

// ListFailureSamples returns recent geocode failures for dashboard drill-down.
func (r *Repository) ListFailureSamples(ctx context.Context, limit int) ([]FailureSample, error) {
	if limit <= 0 {
		limit = 25
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, dataset_slug, listing_key, geocode_failure_reason, geocode_attempt_count,
		       geocode_query, geocode_failed_at
		FROM listings
		WHERE internet_address_display_yn IS TRUE
		  AND (latitude IS NULL OR longitude IS NULL)
		  AND geocode_failure_reason IS NOT NULL
		  AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
		ORDER BY geocode_failed_at DESC NULLS LAST, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FailureSample
	for rows.Next() {
		var row FailureSample
		if err := rows.Scan(
			&row.ID, &row.DatasetSlug, &row.ListingKey, &row.GeocodeFailureReason,
			&row.GeocodeAttemptCount, &row.GeocodeQuery, &row.GeocodeFailedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
