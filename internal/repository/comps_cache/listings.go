package compscache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CachedClosedRow is one closed listing row from listings_cache.
type CachedClosedRow struct {
	ListingKey         string
	CompressedPayload  []byte
	Latitude           float64
	Longitude          float64
}

// ClosedUpsertRow is one write-through row for listings_cache.
type ClosedUpsertRow struct {
	ListingKey      string
	StandardStatus  string
	CloseDate       *time.Time
	Latitude        *float64
	Longitude       *float64
	ClosePrice      *float64
	PayloadRaw      json.RawMessage
}

// Repository reads and writes closed comps in listings_cache.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a listings_cache repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ScopeQuery holds geo scope for cache reads (radius only; zip skips cache).
type ScopeQuery struct {
	Type         string
	RadiusMiles  *float64
	PostalCodes  []string
}

// FindClosedInScope returns fresh closed rows for comps cache hit path.
func (r *Repository) FindClosedInScope(
	ctx context.Context,
	domainSlug, feedCode string,
	subjectLat, subjectLng float64,
	scope ScopeQuery,
	closeCutoff time.Time,
	maxAge time.Duration,
	limit int,
) ([]CachedClosedRow, error) {
	if domainSlug == "" || feedCode == "" || limit <= 0 {
		return nil, nil
	}
	if strings.EqualFold(scope.Type, "zip") {
		return nil, nil
	}

	q := `
		SELECT listing_key, compressed_payload, latitude, longitude
		FROM listings_cache
		WHERE domain_slug = $1 AND feed_code = $2
		  AND LOWER(TRIM(standard_status)) = 'closed'
		  AND close_date >= $3::date
		  AND last_refreshed_at >= NOW() - $4::interval
		  AND close_price > 0
		  AND latitude IS NOT NULL AND longitude IS NOT NULL
	`
	args := []any{domainSlug, feedCode, closeCutoff, maxAge}
	n := 5

	radius := 5.0
	if scope.RadiusMiles != nil && *scope.RadiusMiles > 0 {
		radius = *scope.RadiusMiles
	}
	deg := radius / 69.0
	q += fmt.Sprintf(` AND latitude BETWEEN $%d AND $%d AND longitude BETWEEN $%d AND $%d`, n, n+1, n+2, n+3)
	args = append(args, subjectLat-deg, subjectLat+deg, subjectLng-deg, subjectLng+deg)
	n += 4

	q += fmt.Sprintf(` ORDER BY close_date DESC NULLS LAST LIMIT $%d`, n)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CachedClosedRow
	for rows.Next() {
		var row CachedClosedRow
		var lat, lng *float64
		if err := rows.Scan(&row.ListingKey, &row.CompressedPayload, &lat, &lng); err != nil {
			return nil, err
		}
		if lat != nil && lng != nil {
			row.Latitude, row.Longitude = *lat, *lng
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// UpsertClosedBatch stores or refreshes closed listing payloads.
func (r *Repository) UpsertClosedBatch(ctx context.Context, domainSlug, feedCode string, rows []ClosedUpsertRow) error {
	if len(rows) == 0 {
		return nil
	}
	now := time.Now().UTC()
	for _, row := range rows {
		compressed, err := gzipBytes(row.PayloadRaw)
		if err != nil {
			return err
		}
		status := strings.ToLower(strings.TrimSpace(row.StandardStatus))
		if status == "" {
			status = "closed"
		}
		_, err = r.pool.Exec(ctx, `
			INSERT INTO listings_cache (
				domain_slug, feed_code, listing_key, standard_status,
				close_date, latitude, longitude, close_price,
				compressed_payload, first_cached_at, last_refreshed_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
			ON CONFLICT (domain_slug, feed_code, listing_key) DO UPDATE SET
				standard_status = EXCLUDED.standard_status,
				close_date = EXCLUDED.close_date,
				latitude = EXCLUDED.latitude,
				longitude = EXCLUDED.longitude,
				close_price = EXCLUDED.close_price,
				compressed_payload = EXCLUDED.compressed_payload,
				last_refreshed_at = EXCLUDED.last_refreshed_at
		`, domainSlug, feedCode, row.ListingKey, status,
			row.CloseDate, row.Latitude, row.Longitude, row.ClosePrice,
			compressed, now)
		if err != nil {
			return err
		}
	}
	return nil
}

// PurgeExpired deletes listings_cache rows older than retentionDays.
func (r *Repository) PurgeExpired(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		retentionDays = 30
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM listings_cache
		WHERE last_refreshed_at < $1
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// PayloadBytes decompresses a cached payload.
func PayloadBytes(compressed []byte) ([]byte, error) {
	return gunzip(compressed)
}
