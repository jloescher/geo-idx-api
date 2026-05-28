package search

import (
	"context"
	"fmt"
	"strings"

	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
)

const maxGeographyPatterns = 10

func appendGeographyFilter(
	ctx context.Context,
	gis *gisrepo.Repository,
	q string,
	args []any,
	n int,
	city, county *string,
) (string, []any, int, error) {
	cityQ := strings.TrimSpace(ptrStr(city))
	countyQ := strings.TrimSpace(ptrStr(county))
	if cityQ == "" && countyQ == "" {
		return q, args, n, nil
	}

	patterns := geographyPatterns(ctx, gis, cityQ, countyQ)
	if len(patterns) == 0 {
		return q, args, n, nil
	}

	// Reuse one pattern array for both city and county OR legs (same $n twice is valid in PostgreSQL).
	q += fmt.Sprintf(`
		AND (
			LOWER(TRIM(COALESCE(city, ''))) LIKE ANY($%d)
			OR LOWER(TRIM(COALESCE(county_or_parish, ''))) LIKE ANY($%d)
		)`, n, n)
	args = append(args, patterns)
	n++
	return q, args, n, nil
}

func geographyPatterns(ctx context.Context, gis *gisrepo.Repository, cityQ, countyQ string) []string {
	var cityNames, countyNames, countySlugs []string
	if cityQ != "" {
		rows, err := gis.AutocompleteCities(ctx, cityQ, maxGeographyPatterns)
		if err == nil {
			for _, r := range rows {
				cityNames = append(cityNames, r.CityName)
			}
		}
	}
	if countyQ != "" {
		rows, err := gis.AutocompleteCounties(ctx, countyQ, maxGeographyPatterns)
		if err == nil {
			for _, r := range rows {
				countyNames = append(countyNames, r.CountyName)
				countySlugs = append(countySlugs, r.CountySlug)
			}
		}
	}
	return mergeGeographyLikePatterns(cityQ, countyQ, cityNames, countyNames, countySlugs)
}

// mergeGeographyLikePatterns builds deduplicated case-insensitive LIKE patterns for city/county search.
func mergeGeographyLikePatterns(cityQ, countyQ string, autocompleteCities, countyNames, countySlugs []string) []string {
	seen := make(map[string]struct{})
	var patterns []string

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		key := strings.ToLower(s)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		if len(patterns) >= maxGeographyPatterns {
			return
		}
		patterns = append(patterns, "%"+key+"%")
	}

	if cityQ != "" {
		add(cityQ)
		for _, name := range autocompleteCities {
			add(name)
		}
	}
	if countyQ != "" {
		add(countyQ)
		for _, name := range countyNames {
			add(name)
		}
		for _, slug := range countySlugs {
			add(slug)
		}
	}
	return patterns
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// publicListingComplianceSQL excludes non-IDX / non-display listings on public search.
const publicListingComplianceSQL = `
	AND internet_entire_listing_display_yn IS NOT FALSE
	AND (idx_participation_yn IS NOT FALSE OR idx_participation_yn IS NULL)`
