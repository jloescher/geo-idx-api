package geo

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/analytics"
)

// trackBoundaryEvent sends the go_api_boundary_fetched analytics event.
func (h *Handler) trackBoundaryEvent(ctx context.Context, boundaryType string, geoID int64, durationMs int64) {
	if h.Analytics == nil {
		return
	}
	props := analytics.BoundaryProperties(boundaryType, geoID, "miss", durationMs) // Cache status is always "miss" for now
	h.Analytics.CaptureWithCorrelation(ctx, analytics.EventAPIBoundaryFetched, props)
}

// GeoJSON Feature structure
type GeoJSONFeature struct {
	Type       string          `json:"type"`
	Properties map[string]any  `json:"properties"`
	Geometry   json.RawMessage `json:"geometry,omitempty"`
}

// GeoJSON FeatureCollection structure
type GeoJSONFeatureCollection struct {
	Type     string           `json:"type"`
	Features []GeoJSONFeature `json:"features"`
}

// BoundaryResponse wraps a single boundary with metadata
type BoundaryResponse struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	StateAbbr  string          `json:"state_abbr,omitempty"`
	ParentName string          `json:"parent_name,omitempty"`
	Bounds     *BoundaryBounds `json:"bounds,omitempty"`
	GeoJSON    json.RawMessage `json:"geojson"`
}

// BoundaryBounds contains the bounding box for map centering
type BoundaryBounds struct {
	MinLng    float64 `json:"min_lng"`
	MinLat    float64 `json:"min_lat"`
	MaxLng    float64 `json:"max_lng"`
	MaxLat    float64 `json:"max_lat"`
	CenterLng float64 `json:"center_lng"`
	CenterLat float64 `json:"center_lat"`
}

// boundaryRow represents a row from the boundary queries
type boundaryRow struct {
	ID         int64           `db:"id"`
	Name       string          `db:"name"`
	StateAbbr  string          `db:"state_abbr"`
	ParentName *string         `db:"parent_name"`
	CountyName *string         `db:"county_name"`
	CityName   *string         `db:"city_name"`
	Code       *string         `db:"code"`
	GeoJSON    json.RawMessage `db:"geojson"`
}

// featureRow represents a row from multi-feature queries
type featureRow struct {
	FeatureType string          `db:"feature_type"`
	ID          int64           `db:"id"`
	Name        string          `db:"name"`
	StateAbbr   string          `db:"state_abbr"`
	ParentName  *string         `db:"parent_name"`
	GeoJSON     json.RawMessage `db:"geojson"`
}

// GetCountyBoundary returns a single county boundary as GeoJSON
func (h *Handler) GetCountyBoundary(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid county ID", http.StatusBadRequest)
		return
	}

	query := h.Registry.SQL("GetCountyBoundary")

	var row boundaryRow
	err = h.Pool.QueryRow(ctx, query, id).Scan(&row.ID, &row.Name, &row.StateAbbr, &row.Code, &row.GeoJSON)
	if err != nil {
		http.Error(w, "County not found or has no boundary", http.StatusNotFound)
		return
	}

	h.trackBoundaryEvent(ctx, analytics.BoundaryTypeCounty, id, time.Since(start).Milliseconds())

	resp := BoundaryResponse{
		ID:        row.ID,
		Name:      row.Name,
		Type:      "county",
		StateAbbr: row.StateAbbr,
		GeoJSON:   row.GeoJSON,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	json.NewEncoder(w).Encode(resp)
}

// GetCityBoundary returns a single city boundary as GeoJSON
func (h *Handler) GetCityBoundary(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid city ID", http.StatusBadRequest)
		return
	}

	query := h.Registry.SQL("GetCityBoundary")

	var row boundaryRow
	err = h.Pool.QueryRow(ctx, query, id).Scan(&row.ID, &row.Name, &row.StateAbbr, &row.CountyName, &row.GeoJSON)
	if err != nil {
		http.Error(w, "City not found or has no boundary", http.StatusNotFound)
		return
	}

	h.trackBoundaryEvent(ctx, analytics.BoundaryTypeCity, id, time.Since(start).Milliseconds())

	parentName := ""
	if row.CountyName != nil {
		parentName = *row.CountyName
	}

	resp := BoundaryResponse{
		ID:         row.ID,
		Name:       row.Name,
		Type:       "city",
		StateAbbr:  row.StateAbbr,
		ParentName: parentName,
		GeoJSON:    row.GeoJSON,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	json.NewEncoder(w).Encode(resp)
}

// GetPostalCodeBoundary returns a single postal code boundary as GeoJSON
func (h *Handler) GetPostalCodeBoundary(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	code := chi.URLParam(r, "code")
	if code == "" || len(code) != 5 {
		http.Error(w, "Invalid postal code", http.StatusBadRequest)
		return
	}

	query := h.Registry.SQL("GetPostalCodeBoundary")

	var row boundaryRow
	err := h.Pool.QueryRow(ctx, query, code).Scan(&row.ID, &row.Code, &row.StateAbbr, &row.GeoJSON)
	if err != nil {
		http.Error(w, "Postal code not found or has no boundary", http.StatusNotFound)
		return
	}

	h.trackBoundaryEvent(ctx, analytics.BoundaryTypePostalCode, row.ID, time.Since(start).Milliseconds())

	name := code
	if row.Code != nil {
		name = *row.Code
	}

	resp := BoundaryResponse{
		ID:        row.ID,
		Name:      name,
		Type:      "postal_code",
		StateAbbr: row.StateAbbr,
		GeoJSON:   row.GeoJSON,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	json.NewEncoder(w).Encode(resp)
}

// GetCountyWithCities returns the county boundary and all cities within it as a FeatureCollection
func (h *Handler) GetCountyWithCities(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid county ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := h.Registry.SQL("GetCountyWithCities")

	rows, err := h.Pool.Query(ctx, query, id)
	if err != nil {
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	fc := GeoJSONFeatureCollection{
		Type:     "FeatureCollection",
		Features: []GeoJSONFeature{},
	}

	for rows.Next() {
		var fr featureRow
		if err := rows.Scan(&fr.FeatureType, &fr.ID, &fr.Name, &fr.StateAbbr, &fr.ParentName, &fr.GeoJSON); err != nil {
			continue
		}

		props := map[string]any{
			"id":         fr.ID,
			"name":       fr.Name,
			"type":       fr.FeatureType,
			"state_abbr": fr.StateAbbr,
		}
		if fr.ParentName != nil {
			props["parent_name"] = *fr.ParentName
		}

		fc.Features = append(fc.Features, GeoJSONFeature{
			Type:       "Feature",
			Properties: props,
			Geometry:   fr.GeoJSON,
		})
	}

	if len(fc.Features) == 0 {
		http.Error(w, "County not found or has no boundaries", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=1800") // 30 min cache for collections
	json.NewEncoder(w).Encode(fc)
}

// GetCountyWithSubdivisions returns the county boundary and all subdivisions within it
func (h *Handler) GetCountyWithSubdivisions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid county ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := h.Registry.SQL("GetCountyWithSubdivisions")

	rows, err := h.Pool.Query(ctx, query, id)
	if err != nil {
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	fc := GeoJSONFeatureCollection{
		Type:     "FeatureCollection",
		Features: []GeoJSONFeature{},
	}

	for rows.Next() {
		var fr featureRow
		if err := rows.Scan(&fr.FeatureType, &fr.ID, &fr.Name, &fr.StateAbbr, &fr.ParentName, &fr.GeoJSON); err != nil {
			continue
		}

		props := map[string]any{
			"id":         fr.ID,
			"name":       fr.Name,
			"type":       fr.FeatureType,
			"state_abbr": fr.StateAbbr,
		}
		if fr.ParentName != nil {
			props["parent_name"] = *fr.ParentName
		}

		fc.Features = append(fc.Features, GeoJSONFeature{
			Type:       "Feature",
			Properties: props,
			Geometry:   fr.GeoJSON,
		})
	}

	if len(fc.Features) == 0 {
		http.Error(w, "County not found or has no boundaries", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=1800")
	json.NewEncoder(w).Encode(fc)
}

// GetCityWithSubdivisions returns the city boundary and all subdivisions within it
func (h *Handler) GetCityWithSubdivisions(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid city ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := h.Registry.SQL("GetCityWithSubdivisions")

	rows, err := h.Pool.Query(ctx, query, id)
	if err != nil {
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	fc := GeoJSONFeatureCollection{
		Type:     "FeatureCollection",
		Features: []GeoJSONFeature{},
	}

	for rows.Next() {
		var fr featureRow
		if err := rows.Scan(&fr.FeatureType, &fr.ID, &fr.Name, &fr.StateAbbr, &fr.ParentName, &fr.GeoJSON); err != nil {
			continue
		}

		props := map[string]any{
			"id":         fr.ID,
			"name":       fr.Name,
			"type":       fr.FeatureType,
			"state_abbr": fr.StateAbbr,
		}
		if fr.ParentName != nil {
			props["parent_name"] = *fr.ParentName
		}

		fc.Features = append(fc.Features, GeoJSONFeature{
			Type:       "Feature",
			Properties: props,
			Geometry:   fr.GeoJSON,
		})
	}

	if len(fc.Features) == 0 {
		http.Error(w, "City not found or has no boundaries", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=1800")
	json.NewEncoder(w).Encode(fc)
}

// GetStateWithCounties returns the state boundary and all counties within it as a FeatureCollection
func (h *Handler) GetStateWithCounties(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid state ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	query := h.Registry.SQL("GetStateWithCounties")

	rows, err := h.Pool.Query(ctx, query, id)
	if err != nil {
		http.Error(w, "Query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	fc := GeoJSONFeatureCollection{
		Type:     "FeatureCollection",
		Features: []GeoJSONFeature{},
	}

	for rows.Next() {
		var fr featureRow
		if err := rows.Scan(&fr.FeatureType, &fr.ID, &fr.Name, &fr.StateAbbr, &fr.ParentName, &fr.GeoJSON); err != nil {
			http.Error(w, "Failed to process boundary data", http.StatusInternalServerError)
			return
		}

		props := map[string]any{
			"id":         fr.ID,
			"name":       fr.Name,
			"type":       fr.FeatureType,
			"state_abbr": fr.StateAbbr,
		}
		if fr.ParentName != nil {
			props["parent_name"] = *fr.ParentName
		}

		fc.Features = append(fc.Features, GeoJSONFeature{
			Type:       "Feature",
			Properties: props,
			Geometry:   fr.GeoJSON,
		})
	}

	if len(fc.Features) == 0 {
		http.Error(w, "State not found or has no boundaries", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=1800")
	json.NewEncoder(w).Encode(fc)
}
