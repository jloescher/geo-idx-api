package geocoderepo

import (
	"context"
	"fmt"

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

// NewRepository creates a geocode listings repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// SelectPending returns address-display listings missing coordinates after cursorID.
func (r *Repository) SelectPending(ctx context.Context, cursorID int64, limit int, datasetSlug string) ([]PendingRow, error) {
	if limit <= 0 {
		limit = 200
	}
	query := `
		SELECT id, dataset_slug, unparsed_address, city, state_or_province, postal_code,
		       street_number, street_name
		FROM listings
		WHERE id > $1
		  AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
		  AND internet_address_display_yn IS TRUE
		  AND (latitude IS NULL OR longitude IS NULL)
		  AND (
		    (unparsed_address IS NOT NULL AND trim(unparsed_address) <> '')
		    OR (street_number IS NOT NULL AND city IS NOT NULL)
		  )
	`
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
	query := `
		SELECT COUNT(*)
		FROM listings
		WHERE LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
		  AND internet_address_display_yn IS TRUE
		  AND (latitude IS NULL OR longitude IS NULL)
		  AND (
		    (unparsed_address IS NOT NULL AND trim(unparsed_address) <> '')
		    OR (street_number IS NOT NULL AND city IS NOT NULL)
		  )
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

// ApplyCoords updates latitude, longitude, coordinates, geocoded_at, and geocode_query.
func (r *Repository) ApplyCoords(ctx context.Context, id int64, lat, lng float64, query string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE listings SET
			latitude = $2,
			longitude = $3,
			coordinates = ST_SetSRID(ST_MakePoint($3, $2), 4326),
			geocoded_at = NOW(),
			geocode_query = $4,
			updated_at = NOW()
		WHERE id = $1
		  AND (latitude IS NULL OR longitude IS NULL)
	`, id, lat, lng, query)
	return err
}

// HasActiveGeocodeJob reports whether a geocode enrich job is already queued.
func HasActiveGeocodeJob(ctx context.Context, pool *pgxpool.Pool, table string) (bool, error) {
	if table == "" {
		table = "jobs"
	}
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM `+table+`
			WHERE queue NOT LIKE '%:failed'
			  AND finished_at IS NULL
			  AND payload::jsonb->>'type' IN ('mls.geocode_listings_kickoff', 'mls.geocode_listings_batch')
		)
	`).Scan(&exists)
	return exists, err
}
