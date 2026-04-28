package comps

import (
	"encoding/json"
	"fmt"
	"strings"
)

// scopeResult holds a SQL clause fragment and its positional arguments.
type scopeResult struct {
	clause string
	args   []any
}

// buildScopeClause returns a SQL WHERE fragment and args for the geographic scope.
// The returned clause uses positional placeholder tokens ($SCOPE1, $SCOPE2, ...)
// that must be renumbered before query execution.
func buildScopeClause(subject *resolvedSubject, scope ScopeInput) (scopeResult, error) {
	switch scope.Type {
	case "radius":
		lat, lng, err := scopeCenter(subject, scope)
		if err != nil {
			return scopeResult{}, err
		}
		radiusMiles := 1.0
		if scope.RadiusMiles != nil {
			radiusMiles = *scope.RadiusMiles
		}
		radiusMeters := radiusMiles * 1609.344
		return scopeResult{
			clause: "AND ST_DWithin(pc.location, ST_SetSRID(ST_MakePoint($SCOPE1, $SCOPE2), 4326)::geography, $SCOPE3)",
			args:   []any{lng, lat, radiusMeters},
		}, nil

	case "polygon":
		if scope.PolygonGeoJSON == nil {
			return scopeResult{}, fmt.Errorf("polygon_geojson is required for polygon scope")
		}
		geoJSON, err := json.Marshal(scope.PolygonGeoJSON)
		if err != nil {
			return scopeResult{}, fmt.Errorf("invalid polygon_geojson: %w", err)
		}
		return scopeResult{
			clause: "AND ST_Contains(ST_GeomFromGeoJSON($SCOPE1)::geometry, pc.location::geometry)",
			args:   []any{string(geoJSON)},
		}, nil

	case "neighborhood":
		if scope.SubdivisionRefID == nil {
			return scopeResult{}, fmt.Errorf("subdivision_ref_id is required for neighborhood scope")
		}
		return scopeResult{
			clause: "AND pc.subdivision_ref_id = $SCOPE1",
			args:   []any{*scope.SubdivisionRefID},
		}, nil

	case "zip":
		if len(scope.PostalCodes) == 0 {
			return scopeResult{}, fmt.Errorf("postal_codes is required for zip scope")
		}
		return scopeResult{
			clause: "AND pc.postal_code = ANY($SCOPE1)",
			args:   []any{scope.PostalCodes},
		}, nil

	default:
		return scopeResult{}, fmt.Errorf("unsupported scope type: %s", scope.Type)
	}
}

// scopeCenter returns the lat/lng center for spatial queries.
// Prefers explicit center_lat/center_lng, falls back to subject coordinates.
func scopeCenter(subject *resolvedSubject, scope ScopeInput) (lat, lng float64, err error) {
	if scope.CenterLat != nil && scope.CenterLng != nil {
		return *scope.CenterLat, *scope.CenterLng, nil
	}
	if subject.Lat != 0 || subject.Lng != 0 {
		return subject.Lat, subject.Lng, nil
	}
	return 0, 0, fmt.Errorf("scope center coordinates required: set center_lat/center_lng or provide subject with lat/lng")
}

// buildFilterClauses returns additional SQL WHERE fragments based on filter tolerances.
func buildFilterClauses(subject *resolvedSubject, filters *FiltersInput) (string, []any) {
	var clauses []string
	var args []any
	argIdx := 1 // placeholder numbering, renumbered later

	// Living area tolerance
	if subject.LivingAreaSqft != nil && filters.livingAreaPct() > 0 {
		pct := float64(filters.livingAreaPct()) / 100.0
		maxDiff := int(float64(*subject.LivingAreaSqft) * pct)
		clauses = append(clauses, fmt.Sprintf("AND ABS(COALESCE(pc.living_area, 0) - $FILTER%d) <= $FILTER%d", argIdx, argIdx+1))
		args = append(args, *subject.LivingAreaSqft, maxDiff)
		argIdx += 2
	}

	// Beds tolerance
	if subject.Bedrooms != nil && filters.bedsTolerance() >= 0 {
		clauses = append(clauses, fmt.Sprintf("AND ABS(COALESCE(pc.bedrooms_total, 0) - $FILTER%d) <= $FILTER%d", argIdx, argIdx+1))
		args = append(args, *subject.Bedrooms, filters.bedsTolerance())
		argIdx += 2
	}

	// Baths tolerance
	if subject.Bathrooms != nil && filters.bathsTolerance() >= 0 {
		clauses = append(clauses, fmt.Sprintf("AND ABS(COALESCE(pc.bathrooms_total, 0) - $FILTER%d) <= $FILTER%d", argIdx, argIdx+1))
		args = append(args, *subject.Bathrooms, filters.bathsTolerance())
		argIdx += 2
	}

	// Pool match
	if filters.matchPool() && subject.Pool != nil {
		clauses = append(clauses, fmt.Sprintf("AND pc.pool_private_yn = $FILTER%d", argIdx))
		args = append(args, *subject.Pool)
		argIdx++
	}

	// HOA match
	if filters.matchHOA() && subject.HOA != nil {
		clauses = append(clauses, fmt.Sprintf("AND pc.association_yn = $FILTER%d", argIdx))
		args = append(args, *subject.HOA)
		argIdx++
	}

	// Senior community match
	if filters.matchSeniorCommunity() && subject.SeniorCommunity != nil {
		clauses = append(clauses, fmt.Sprintf("AND pc.senior_community_yn = $FILTER%d", argIdx))
		args = append(args, *subject.SeniorCommunity)
		argIdx++
	}

	// Flood zone match — handled Go-side via partitionByFloodZone (not SQL-level).
	// The 2% similarity weight in SQL still promotes same-zone comps.

	// Property sub-type match (appraisal: never compare SFR to condo)
	if filters.matchPropertySubType() && subject.PropertySubType != nil {
		clauses = append(clauses, fmt.Sprintf("AND pc.property_sub_type = $FILTER%d", argIdx))
		args = append(args, *subject.PropertySubType)
		argIdx++
	}

	// Waterfront match
	if filters.matchWaterfront() && subject.Waterfront != nil {
		clauses = append(clauses, fmt.Sprintf("AND pc.waterfront_yn = $FILTER%d", argIdx))
		args = append(args, *subject.Waterfront)
		argIdx++
	}

	// Year built tolerance
	if subject.YearBuilt != nil && filters.yearBuiltTolerance() >= 0 {
		clauses = append(clauses, fmt.Sprintf("AND ABS(COALESCE(pc.year_built, 0) - $FILTER%d) <= $FILTER%d", argIdx, argIdx+1))
		args = append(args, *subject.YearBuilt, filters.yearBuiltTolerance())
		argIdx += 2
	}

	// Lot size tolerance
	if subject.LotSizeAcres != nil && filters.lotSizePct() > 0 {
		pct := float64(filters.lotSizePct()) / 100.0
		maxDiff := *subject.LotSizeAcres * pct
		clauses = append(clauses, fmt.Sprintf("AND ABS(COALESCE(pc.lot_size_acres, 0) - $FILTER%d) <= $FILTER%d", argIdx, argIdx+1))
		args = append(args, *subject.LotSizeAcres, maxDiff)
		argIdx += 2
	}

	return strings.Join(clauses, "\n  "), args
}

// renumberPlaceholders replaces $SCOPE1, $SCOPE2, ... and $FILTER1, $FILTER2, ...
// with sequential positional parameters starting at startIdx.
// Returns the renumbered SQL and the next available parameter index.
//
// Replacements are collected in ascending order (for correct index assignment)
// but applied in reverse order to prevent prefix collisions (e.g., replacing
// "$FILTER1" before "$FILTER10" would corrupt "$FILTER10" into "$N0").
func renumberPlaceholders(sql string, startIdx int) (string, int) {
	type replacement struct{ old, new string }
	var reps []replacement
	idx := startIdx

	for i := 1; i <= 10; i++ {
		placeholder := fmt.Sprintf("$SCOPE%d", i)
		if strings.Contains(sql, placeholder) {
			reps = append(reps, replacement{placeholder, fmt.Sprintf("$%d", idx)})
			idx++
		}
	}
	for i := 1; i <= 20; i++ {
		placeholder := fmt.Sprintf("$FILTER%d", i)
		if strings.Contains(sql, placeholder) {
			reps = append(reps, replacement{placeholder, fmt.Sprintf("$%d", idx)})
			idx++
		}
	}

	// Apply in reverse so higher-numbered placeholders are replaced first.
	for i := len(reps) - 1; i >= 0; i-- {
		sql = strings.ReplaceAll(sql, reps[i].old, reps[i].new)
	}
	return sql, idx
}
