package comps

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// findMirrorComps loads Active/Pending (and optional rental) comps from the listings mirror.
// Closed/sold comps are not replicated — use live Bridge or Spark RESO instead.
func (e *Engine) findMirrorComps(
	ctx context.Context,
	dataset string,
	subject SubjectProfile,
	statuses []string,
	scope ScopeInput,
	filters FiltersInput,
	limit int,
	rentalOnly bool,
) ([]CompRecord, error) {
	if limit <= 0 {
		limit = 25
	}
	q := `
		SELECT raw_data, media, unit, room, open_house, custom_fields,
		       standard_status, list_price, latitude, longitude, listing_key
		FROM listings WHERE dataset_slug = $1
	`
	args := []any{dataset}
	n := 2
	if len(statuses) > 0 {
		lower := make([]string, len(statuses))
		for i, s := range statuses {
			lower[i] = strings.ToLower(strings.TrimSpace(s))
		}
		q += fmt.Sprintf(" AND LOWER(TRIM(COALESCE(standard_status,''))) = ANY($%d)", n)
		args = append(args, lower)
		n++
	}
	if scope.Type == "zip" && len(scope.PostalCodes) > 0 {
		q += fmt.Sprintf(" AND postal_code = ANY($%d)", n)
		args = append(args, scope.PostalCodes)
		n++
	} else if scope.RadiusMiles != nil && *scope.RadiusMiles > 0 {
		meters := *scope.RadiusMiles * 1609.34
		q += fmt.Sprintf(` AND coordinates IS NOT NULL AND ST_DWithin(
			coordinates::geography,
			ST_SetSRID(ST_MakePoint($%d, $%d), 4326)::geography,
			$%d
		)`, n, n+1, n+2)
		args = append(args, subject.Lng, subject.Lat, meters)
		n += 3
	}
	if subject.ListingKey != "" {
		q += fmt.Sprintf(" AND listing_key <> $%d", n)
		args = append(args, subject.ListingKey)
		n++
	}
	if rentalOnly {
		q += ` AND (
			LOWER(COALESCE(property_type, '')) LIKE '%lease%'
			OR LOWER(COALESCE(property_sub_type, '')) LIKE '%lease%'
		)`
	} else {
		q += ` AND (
			property_type IS NULL
			OR LOWER(property_type) NOT LIKE '%lease%'
		)`
	}
	applyFilterSQL(&q, &args, &n, subject, filters)
	q += fmt.Sprintf(" ORDER BY modification_timestamp DESC NULLS LAST LIMIT $%d", n)
	args = append(args, limit)

	pool, err := e.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CompRecord
	for rows.Next() {
		var raw, media, unit, room, oh, custom []byte
		var status string
		var listPrice *float64
		var lat, lng *float64
		var key string
		if err := rows.Scan(&raw, &media, &unit, &room, &oh, &custom, &status, &listPrice, &lat, &lng, &key); err != nil {
			return nil, err
		}
		merged := mls.MergeMirrorListing(json.RawMessage(raw), mls.ExpandedPayload{
			Media: json.RawMessage(media), Unit: json.RawMessage(unit),
			Room: json.RawMessage(room), OpenHouse: json.RawMessage(oh),
			HasMedia: len(media) > 0,
		}, json.RawMessage(custom))
		c := parseProperty(merged)
		c.ListingKey = key
		c.StandardStatus = status
		if listPrice != nil {
			c.ListPrice = *listPrice
		}
		if lat != nil && lng != nil {
			c.Lat, c.Lng = *lat, *lng
		}
		c.DistanceMiles = haversineMiles(subject.Lat, subject.Lng, c.Lat, c.Lng)
		out = append(out, c)
	}
	return out, rows.Err()
}

func applyFilterSQL(q *string, args *[]any, n *int, subject SubjectProfile, f FiltersInput) {
	if f.LivingAreaPct != nil && subject.LivingArea > 0 {
		pct := *f.LivingAreaPct / 100
		lo := subject.LivingArea * (1 - pct)
		hi := subject.LivingArea * (1 + pct)
		*q += fmt.Sprintf(" AND living_area BETWEEN $%d AND $%d", *n, *n+1)
		*args = append(*args, int(lo), int(hi))
		*n += 2
	}
	if f.BedsTolerance != nil {
		*q += fmt.Sprintf(" AND bedrooms_total BETWEEN $%d AND $%d", *n, *n+1)
		lo := int(subject.Bedrooms - *f.BedsTolerance)
		hi := int(subject.Bedrooms + *f.BedsTolerance)
		if lo < 0 {
			lo = 0
		}
		*args = append(*args, lo, hi)
		*n += 2
	}
}
