package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// PostgisSearch queries local listings mirror.
type PostgisSearch struct {
	db *repository.DB
}

func NewPostgisSearch(db *repository.DB) *PostgisSearch {
	return &PostgisSearch{db: db}
}

func (p *PostgisSearch) Search(ctx context.Context, feedCode string, req SearchRequest) (SearchResult, error) {
	dataset := strings.TrimPrefix(feedCode, "bridge_")
	dataset = strings.TrimPrefix(dataset, "spark_")
	limit := req.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	skip := req.Skip
	if skip < 0 {
		skip = 0
	}

	q := `
		SELECT raw_data, media FROM listings
		WHERE dataset_slug = $1
		  AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
	`
	args := []any{dataset}
	n := 2
	if req.MinPrice != nil {
		q += fmt.Sprintf(" AND list_price >= $%d", n)
		args = append(args, *req.MinPrice)
		n++
	}
	if req.MaxPrice != nil {
		q += fmt.Sprintf(" AND list_price <= $%d", n)
		args = append(args, *req.MaxPrice)
		n++
	}
	if req.BedsMin != nil {
		q += fmt.Sprintf(" AND bedrooms_total >= $%d", n)
		args = append(args, *req.BedsMin)
		n++
	}
	if req.LowRiskFloodzone != nil && *req.LowRiskFloodzone {
		q += " AND low_risk_flood_zone_yn = TRUE"
	}
	if req.MinMonthlyFees != nil {
		q += fmt.Sprintf(" AND estimated_total_monthly_fees >= $%d", n)
		args = append(args, *req.MinMonthlyFees)
		n++
	}
	if req.MaxMonthlyFees != nil {
		q += fmt.Sprintf(" AND estimated_total_monthly_fees <= $%d", n)
		args = append(args, *req.MaxMonthlyFees)
		n++
	}
	if req.Lat != nil && req.Lng != nil && req.RadiusMiles != nil {
		meters := *req.RadiusMiles * 1609.34
		q += fmt.Sprintf(` AND coordinates IS NOT NULL AND ST_DWithin(
			coordinates::geography,
			ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography,
			$%d
		)`, n, n+1, n+2)
		args = append(args, *req.Lng, *req.Lat, meters)
		n += 3
	}
	q += fmt.Sprintf(" ORDER BY modification_timestamp DESC NULLS LAST LIMIT $%d OFFSET $%d", n, n+1)
	args = append(args, limit+1, skip)

	rows, err := p.db.Pool.Query(ctx, q, args...)
	if err != nil {
		return SearchResult{}, err
	}
	defer rows.Close()

	var results []json.RawMessage
	for rows.Next() {
		var raw []byte
		var media []byte
		if err := rows.Scan(&raw, &media); err != nil {
			return SearchResult{}, err
		}
		if len(raw) == 0 && len(media) == 0 {
			continue
		}
		merged := mls.MergeListingJSON(json.RawMessage(raw), json.RawMessage(media))
		results = append(results, merged)
	}
	hasMore := len(results) > limit
	if hasMore {
		results = results[:limit]
	}
	return SearchResult{
		Results:  results,
		HasMore:  hasMore,
		NextSkip: skip + len(results),
	}, nil
}

// RollingWindowCutoff returns oldest modification timestamp for mirror window.
func RollingWindowCutoff(months int) time.Time {
	return time.Now().AddDate(0, -months, 0)
}
