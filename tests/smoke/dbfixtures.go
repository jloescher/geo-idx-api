//go:build smoke

package smoke

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBFixtures holds listing identifiers resolved from read-only SELECT queries.
type DBFixtures struct {
	ListingKey   string
	MLSListingID string
	PhotoID      string
	DomainSlug   string
}

func ResolveDBFixtures(cfg Config) (DBFixtures, error) {
	fix := DBFixtures{
		ListingKey:   "STELLAR-PLACEHOLDER",
		MLSListingID: "1",
		PhotoID:      "1",
		DomainSlug:   cfg.DomainSlug,
	}
	if cfg.DBDSN == "" {
		return fix, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DBDSN)
	if err != nil {
		return fix, fmt.Errorf("db connect: %w", err)
	}
	defer pool.Close()

	if cfg.DomainSlug == "" || cfg.DomainSlug == "example.com" {
		var slug string
		err = pool.QueryRow(ctx, `
			SELECT domain_slug FROM domains WHERE is_active ORDER BY id LIMIT 1
		`).Scan(&slug)
		if err == nil && strings.TrimSpace(slug) != "" {
			fix.DomainSlug = strings.TrimSpace(slug)
		}
	}

	var listingKey, mlsListingID string
	err = pool.QueryRow(ctx, `
		SELECT listing_key, COALESCE(mls_listing_id, '')
		FROM listings
		WHERE dataset_slug = $1
		  AND LOWER(TRIM(COALESCE(standard_status, ''))) = 'active'
		ORDER BY modification_timestamp DESC NULLS LAST
		LIMIT 1
	`, cfg.Dataset).Scan(&listingKey, &mlsListingID)
	if err == nil {
		if listingKey != "" {
			fix.ListingKey = listingKey
		}
		if mlsListingID != "" {
			fix.MLSListingID = mlsListingID
		}
	}

	var photoID string
	err = pool.QueryRow(ctx, `
		SELECT COALESCE(media->0->>'MediaKey', media->0->>'PhotoId', '')
		FROM listings
		WHERE dataset_slug = $1 AND listing_key = $2
		  AND jsonb_array_length(COALESCE(media, '[]'::jsonb)) > 0
		LIMIT 1
	`, cfg.Dataset, fix.ListingKey).Scan(&photoID)
	if err == nil && strings.TrimSpace(photoID) != "" {
		fix.PhotoID = strings.TrimSpace(photoID)
	}

	return fix, nil
}
