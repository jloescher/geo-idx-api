package geo

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

// trackChildrenEvent sends analytics event for children endpoints.
func (h *Handler) trackChildrenEvent(ctx context.Context, childrenType string, parentID int64, resultsCount int, durationMs int64) {
	if h.Analytics == nil {
		return
	}
	props := map[string]interface{}{
		"type":        childrenType,
		"parent_id":   parentID,
		"count":       resultsCount,
		"duration_ms": durationMs,
	}
	h.Analytics.CaptureWithCorrelation(ctx, "go_api_children_fetched", props)
}

// CountyChild represents a county in a state children response.
type CountyChild struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	NormalizedName string  `json:"normalized_name"`
	Slug           string  `json:"slug"`
	StateID        int64   `json:"state_id"`
	StateName      string  `json:"state_name"`
	StateAbbr      string  `json:"state_abbr"`
	PropertyCount  *int64  `json:"property_count,omitempty"`
	Latitude       *string `json:"latitude,omitempty"`
	Longitude      *string `json:"longitude,omitempty"`
}

// CityChild represents a city in a county children response.
type CityChild struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	NormalizedName string  `json:"normalized_name"`
	Slug           string  `json:"slug"`
	StateID        int64   `json:"state_id"`
	StateName      string  `json:"state_name"`
	StateAbbr      string  `json:"state_abbr"`
	CountyID       int64   `json:"county_id"`
	CountyName     string  `json:"county_name"`
	Latitude       *string `json:"latitude,omitempty"`
	Longitude      *string `json:"longitude,omitempty"`
}

// SubdivisionChild represents a subdivision in a county or city children response.
type SubdivisionChild struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	NormalizedName string  `json:"normalized_name"`
	Slug           string  `json:"slug"`
	StateID        int64   `json:"state_id"`
	StateName      string  `json:"state_name"`
	StateAbbr      string  `json:"state_abbr"`
	CountyID       *int64  `json:"county_id,omitempty"`
	CountyName     *string `json:"county_name,omitempty"`
	CityID         *int64  `json:"city_id,omitempty"`
	CityName       *string `json:"city_name,omitempty"`
}

// ListCountiesByState returns all counties in a state as a JSON array.
// GET /api/v1/geo/children/states/{id}/counties
func (h *Handler) ListCountiesByState(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid state ID", http.StatusBadRequest)
		return
	}

	var counties []CountyChild
	err = pgxscan.Select(ctx, h.Pool, &counties, h.Registry.SQL("ListCountiesByState"), id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Return empty array instead of null
	if counties == nil {
		counties = []CountyChild{}
	}

	h.trackChildrenEvent(ctx, "counties_by_state", id, len(counties), time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300") // 5 min cache
	json.NewEncoder(w).Encode(counties)
}

// ListCitiesByCounty returns all cities in a county as a JSON array.
// GET /api/v1/geo/children/counties/{id}/cities
func (h *Handler) ListCitiesByCounty(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid county ID", http.StatusBadRequest)
		return
	}

	var cities []CityChild
	err = pgxscan.Select(ctx, h.Pool, &cities, h.Registry.SQL("ListCitiesByCounty"), id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if cities == nil {
		cities = []CityChild{}
	}

	h.trackChildrenEvent(ctx, "cities_by_county", id, len(cities), time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	json.NewEncoder(w).Encode(cities)
}

// ListSubdivisionsByCounty returns all subdivisions in a county as a JSON array.
// GET /api/v1/geo/children/counties/{id}/subdivisions
func (h *Handler) ListSubdivisionsByCounty(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid county ID", http.StatusBadRequest)
		return
	}

	var subdivisions []SubdivisionChild
	err = pgxscan.Select(ctx, h.Pool, &subdivisions, h.Registry.SQL("ListSubdivisionsByCounty"), id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if subdivisions == nil {
		subdivisions = []SubdivisionChild{}
	}

	h.trackChildrenEvent(ctx, "subdivisions_by_county", id, len(subdivisions), time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	json.NewEncoder(w).Encode(subdivisions)
}

// ListSubdivisionsByCity returns all subdivisions in a city as a JSON array.
// GET /api/v1/geo/children/cities/{id}/subdivisions
func (h *Handler) ListSubdivisionsByCity(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid city ID", http.StatusBadRequest)
		return
	}

	var subdivisions []SubdivisionChild
	err = pgxscan.Select(ctx, h.Pool, &subdivisions, h.Registry.SQL("ListSubdivisionsByCity"), id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if subdivisions == nil {
		subdivisions = []SubdivisionChild{}
	}

	h.trackChildrenEvent(ctx, "subdivisions_by_city", id, len(subdivisions), time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	json.NewEncoder(w).Encode(subdivisions)
}
