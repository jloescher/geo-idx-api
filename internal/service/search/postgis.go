package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/repository"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// PostgisSearch queries local listings mirror.
type PostgisSearch struct {
	db  *repository.DB
	gis *gisrepo.Repository
}

func NewPostgisSearch(db *repository.DB) *PostgisSearch {
	return &PostgisSearch{db: db, gis: gisrepo.New(db)}
}

func (p *PostgisSearch) Search(ctx context.Context, feedCode string, req SearchRequest, rollingMonths int) (SearchResult, error) {
	dataset := mls.DatasetSlugFromFeedCode(feedCode)
	limit := req.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	skip := req.Skip
	if skip < 0 {
		skip = 0
	}

	q := `
		SELECT ` + mls.MirrorListingSearchColumns + ` FROM listings
		WHERE dataset_slug = $1
	`
	if len(req.Statuses) == 0 {
		q += ` AND LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')`
		q += publicListingComplianceSQL
	}
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
	if req.BedsMax != nil {
		q += fmt.Sprintf(" AND bedrooms_total <= $%d", n)
		args = append(args, *req.BedsMax)
		n++
	}
	if req.BathsMin != nil {
		q += fmt.Sprintf(" AND bathrooms_total_decimal >= $%d", n)
		args = append(args, *req.BathsMin)
		n++
	}
	if req.LivingAreaMin != nil {
		q += fmt.Sprintf(" AND living_area >= $%d", n)
		args = append(args, *req.LivingAreaMin)
		n++
	}
	if req.LivingAreaMax != nil {
		q += fmt.Sprintf(" AND living_area <= $%d", n)
		args = append(args, *req.LivingAreaMax)
		n++
	}
	if req.LotSizeAcresMin != nil {
		q += fmt.Sprintf(" AND lot_size_acres >= $%d", n)
		args = append(args, *req.LotSizeAcresMin)
		n++
	}
	if req.YearBuiltMin != nil {
		q += fmt.Sprintf(" AND year_built >= $%d", n)
		args = append(args, *req.YearBuiltMin)
		n++
	}
	if req.PropertyType != nil && strings.TrimSpace(*req.PropertyType) != "" {
		q += fmt.Sprintf(" AND LOWER(property_type) = LOWER($%d)", n)
		args = append(args, strings.TrimSpace(*req.PropertyType))
		n++
	}
	if req.PropertySubType != nil && strings.TrimSpace(*req.PropertySubType) != "" {
		q += fmt.Sprintf(" AND LOWER(property_sub_type) = LOWER($%d)", n)
		args = append(args, strings.TrimSpace(*req.PropertySubType))
		n++
	}
	var err error
	q, args, n, err = appendGeographyFilter(ctx, p.gis, q, args, n, req.City, req.CountyOrParish)
	if err != nil {
		return SearchResult{}, err
	}
	if req.RemarksQuery != nil && strings.TrimSpace(*req.RemarksQuery) != "" {
		q += fmt.Sprintf(` AND to_tsvector('english', COALESCE(public_remarks, '')) @@ plainto_tsquery('english', $%d)`, n)
		args = append(args, strings.TrimSpace(*req.RemarksQuery))
		n++
	}
	if req.PostalCode != nil && strings.TrimSpace(*req.PostalCode) != "" {
		q += fmt.Sprintf(" AND postal_code = $%d", n)
		args = append(args, strings.TrimSpace(*req.PostalCode))
		n++
	}
	if req.PoolPrivate != nil && *req.PoolPrivate {
		q += " AND pool_private_yn = TRUE"
	}
	if req.Waterfront != nil && *req.Waterfront {
		q += " AND waterfront_yn = TRUE"
	}
	if len(req.Statuses) > 0 {
		var lowered []string
		for _, st := range req.Statuses {
			lowered = append(lowered, strings.ToLower(strings.TrimSpace(st)))
		}
		q += fmt.Sprintf(" AND LOWER(TRIM(COALESCE(standard_status, ''))) = ANY($%d)", n)
		args = append(args, lowered)
		n++
	}
	if req.PriceReducedWithinDays != nil && *req.PriceReducedWithinDays > 0 {
		q += fmt.Sprintf(" AND price_change_timestamp >= NOW() - ($%d || ' days')::interval", n)
		args = append(args, *req.PriceReducedWithinDays)
		n++
	}
	// low_risk_flood_zone_yn is set by FEMA NFHL enrichment (fema_flood_zone_code), not MLS persist.
	if req.LowRiskFloodzone != nil && *req.LowRiskFloodzone {
		q += " AND low_risk_flood_zone_yn = TRUE"
	}
	if rollingMonths > 0 {
		q += fmt.Sprintf(" AND modification_timestamp >= $%d", n)
		args = append(args, RollingWindowCutoff(rollingMonths))
		n++
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

	pool, err := p.db.ReadPool(ctx)
	if err != nil {
		return SearchResult{}, err
	}
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return SearchResult{}, err
	}
	defer rows.Close()

	var results []json.RawMessage
	for rows.Next() {
		mirrorRow, err := mls.ScanMirrorListingSearchRow(rows.Scan)
		if err != nil {
			return SearchResult{}, err
		}
		if mirrorRow.ListingKey == "" {
			continue
		}
		body, ok := mls.BuildPublicListingJSONForSearch(mirrorRow)
		if !ok {
			continue
		}
		results = append(results, body)
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
