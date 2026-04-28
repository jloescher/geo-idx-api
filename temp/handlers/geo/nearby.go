package geo

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/georgysavva/scany/v2/pgxscan"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
)

const (
	defaultNearbyLimit       = 10
	maxNearbyLimit           = 30
	defaultNearbyRadiusMiles = 25.0
	maxNearbyRadiusMiles     = 100.0
)

type nearbyItem struct {
	RefID         int64    `db:"ref_id" json:"ref_id"`
	Name          string   `db:"name" json:"name"`
	Slug          string   `db:"slug" json:"slug"`
	Type          string   `db:"type" json:"type"`
	DistanceMiles *float64 `db:"distance_miles" json:"distance_miles"`
	ListingCount  int64    `db:"listing_count" json:"listing_count"`
}

type nearbyResponse struct {
	Type    string       `json:"type"`
	RefID   int64        `json:"ref_id"`
	Limit   int          `json:"limit"`
	Results []nearbyItem `json:"results"`
}

// batchCountRow maps the result of NearbyBatchCount.
type batchCountRow struct {
	RefID        int64 `db:"ref_id"`
	ListingCount int64 `db:"listing_count"`
}

var validNearbyTypes = map[string]bool{
	"county": true, "city": true, "postal_code": true, "subdivision": true,
}

// GetNearby returns nearby geographic entities of the same type.
// GET /api/v1/geo/nearby?type={county|city|postal_code|subdivision}&ref_id={id}&limit={10}&radius_miles={25}&min_listing_count={0}
func (h *Handler) GetNearby(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	geoType, refID, limit, radiusMiles, minListingCount, ok := parseNearbyParams(w, r)
	if !ok {
		return
	}

	results, err := h.fetchNearby(ctx, geoType, refID, limit, radiusMiles)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch nearby entities"})
		return
	}

	results = filterByMinCount(results, minListingCount)

	w.Header().Set("Cache-Control", "public, max-age=1800")
	writeJSON(w, http.StatusOK, nearbyResponse{
		Type:    geoType,
		RefID:   refID,
		Limit:   limit,
		Results: results,
	})
}

// nearbyPostRequest is the JSON body for POST /api/v1/geo/nearby.
type nearbyPostRequest struct {
	Type            string         `json:"type"`
	RefID           int64          `json:"ref_id"`
	Limit           int            `json:"limit"`
	RadiusMiles     float64        `json:"radius_miles"`
	MinListingCount int            `json:"min_listing_count"`
	SearchFilters   *nearbyFilters `json:"search_filters"`
}

type nearbyFilters struct {
	ListingTypes     []string `json:"listing_types"`
	Statuses         []string `json:"statuses"`
	PropertySubTypes []string `json:"property_sub_types"`
}

// PostNearby returns nearby entities with optional search-filter-based listing counts.
// POST /api/v1/geo/nearby
func (h *Handler) PostNearby(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body nearbyPostRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	geoType := strings.TrimSpace(strings.ToLower(body.Type))
	if geoType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": `missing required field "type"`})
		return
	}
	if !validNearbyTypes[geoType] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": `"type" must be one of: county, city, postal_code, subdivision`})
		return
	}

	if body.RefID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": `invalid "ref_id"`})
		return
	}

	limit := body.Limit
	if limit <= 0 {
		limit = defaultNearbyLimit
	}
	if limit > maxNearbyLimit {
		limit = maxNearbyLimit
	}

	radiusMiles := body.RadiusMiles
	if radiusMiles <= 0 {
		radiusMiles = defaultNearbyRadiusMiles
	}
	if radiusMiles > maxNearbyRadiusMiles {
		radiusMiles = maxNearbyRadiusMiles
	}

	results, err := h.fetchNearby(ctx, geoType, body.RefID, limit, radiusMiles)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch nearby entities"})
		return
	}

	// If search filters provided, replace listing counts with filtered counts.
	if body.SearchFilters != nil && len(results) > 0 {
		refIDs := make([]int64, len(results))
		for i, item := range results {
			refIDs[i] = item.RefID
		}

		var rows []batchCountRow
		err = pgxscan.Select(ctx, h.Pool, &rows,
			h.Registry.SQL(db.QueryName("NearbyBatchCount")),
			geoType, refIDs,
			nilIfEmpty(body.SearchFilters.ListingTypes),
			nilIfEmpty(body.SearchFilters.Statuses),
			nilIfEmpty(body.SearchFilters.PropertySubTypes),
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch filtered counts"})
			return
		}

		countMap := make(map[int64]int64, len(rows))
		for _, row := range rows {
			countMap[row.RefID] = row.ListingCount
		}
		for i := range results {
			results[i].ListingCount = countMap[results[i].RefID] // defaults to 0 for missing
		}
	}

	results = filterByMinCount(results, body.MinListingCount)

	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, nearbyResponse{
		Type:    geoType,
		RefID:   body.RefID,
		Limit:   limit,
		Results: results,
	})
}

// --- helpers ---

// parseNearbyParams extracts and validates GET query parameters. Returns false if an error response was written.
func parseNearbyParams(w http.ResponseWriter, r *http.Request) (geoType string, refID int64, limit int, radiusMiles float64, minListingCount int, ok bool) {
	geoType = strings.TrimSpace(strings.ToLower(r.URL.Query().Get("type")))
	if geoType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": `missing required parameter "type"`})
		return
	}
	if !validNearbyTypes[geoType] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": `"type" must be one of: county, city, postal_code, subdivision`})
		return
	}

	refIDStr := strings.TrimSpace(r.URL.Query().Get("ref_id"))
	if refIDStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": `missing required parameter "ref_id"`})
		return
	}
	var err error
	refID, err = strconv.ParseInt(refIDStr, 10, 64)
	if err != nil || refID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": `invalid "ref_id"`})
		return
	}

	limit = defaultNearbyLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": `invalid "limit"`})
			return
		}
		limit = parsed
		if limit > maxNearbyLimit {
			limit = maxNearbyLimit
		}
	}

	radiusMiles = defaultNearbyRadiusMiles
	if raw := strings.TrimSpace(r.URL.Query().Get("radius_miles")); raw != "" {
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": `invalid "radius_miles"`})
			return
		}
		radiusMiles = parsed
		if radiusMiles > maxNearbyRadiusMiles {
			radiusMiles = maxNearbyRadiusMiles
		}
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("min_listing_count")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": `invalid "min_listing_count"`})
			return
		}
		minListingCount = parsed
	}

	ok = true
	return
}

// fetchNearby runs the appropriate spatial query for the given geo type.
func (h *Handler) fetchNearby(ctx context.Context, geoType string, refID int64, limit int, radiusMiles float64) ([]nearbyItem, error) {
	var results []nearbyItem
	var err error

	switch geoType {
	case "county":
		err = pgxscan.Select(ctx, h.Pool, &results,
			h.Registry.SQL(db.QueryName("GetNearbyCounties")), refID, radiusMiles, limit)
	case "city":
		err = pgxscan.Select(ctx, h.Pool, &results,
			h.Registry.SQL(db.QueryName("GetNearbyCities")), refID, radiusMiles, limit)
	case "postal_code":
		err = pgxscan.Select(ctx, h.Pool, &results,
			h.Registry.SQL(db.QueryName("GetNearbyPostalCodes")), refID, radiusMiles, limit)
	case "subdivision":
		err = pgxscan.Select(ctx, h.Pool, &results,
			h.Registry.SQL(db.QueryName("GetNearbySubdivisions")), refID, limit)
	}

	if err != nil {
		return nil, err
	}
	if results == nil {
		results = []nearbyItem{}
	}
	return results, nil
}

// filterByMinCount removes items with listing_count below the threshold.
func filterByMinCount(items []nearbyItem, min int) []nearbyItem {
	if min <= 0 {
		return items
	}
	filtered := make([]nearbyItem, 0, len(items))
	for _, item := range items {
		if item.ListingCount >= int64(min) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// nilIfEmpty returns nil if the slice is empty, otherwise returns the slice.
// This lets SQL treat the parameter as NULL (match all) when no filter is specified.
func nilIfEmpty(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return s
}
