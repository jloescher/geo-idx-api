package geo

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) trackSEOEvent(ctx context.Context, seoType string, parentID int64, resultsCount int, durationMs int64) {
	if h.Analytics == nil {
		return
	}
	props := map[string]interface{}{
		"type":        seoType,
		"parent_id":   parentID,
		"count":       resultsCount,
		"duration_ms": durationMs,
	}
	h.Analytics.CaptureWithCorrelation(ctx, "go_api_seo_fetched", props)
}

type CountySEO struct {
	ID                  int64   `json:"id"`
	Name                string  `json:"name"`
	NormalizedName      string  `json:"normalized_name"`
	Slug                string  `json:"slug"`
	StateID             int64   `json:"state_id"`
	StateName           string  `json:"state_name"`
	StateAbbr           string  `json:"state_abbr"`
	PropertyCount       *int64  `json:"property_count,omitempty"`
	ActivePropertyCount *int64  `json:"active_property_count,omitempty"`
	Latitude            *string `json:"latitude,omitempty"`
	Longitude           *string `json:"longitude,omitempty"`
}

type CitySEO struct {
	ID                  int64   `json:"id"`
	Name                string  `json:"name"`
	NormalizedName      string  `json:"normalized_name"`
	Slug                string  `json:"slug"`
	StateID             int64   `json:"state_id"`
	StateName           string  `json:"state_name"`
	StateAbbr           string  `json:"state_abbr"`
	CountyID            int64   `json:"county_id"`
	CountyName          string  `json:"county_name"`
	PropertyCount       *int64  `json:"property_count,omitempty"`
	ActivePropertyCount *int64  `json:"active_property_count,omitempty"`
	Latitude            *string `json:"latitude,omitempty"`
	Longitude           *string `json:"longitude,omitempty"`
}

type SubdivisionSEO struct {
	ID                  int64   `json:"id"`
	Name                string  `json:"name"`
	NormalizedName      string  `json:"normalized_name"`
	Slug                string  `json:"slug"`
	StateID             int64   `json:"state_id"`
	StateName           string  `json:"state_name"`
	StateAbbr           string  `json:"state_abbr"`
	CountyID            *int64  `json:"county_id,omitempty"`
	CountyName          *string `json:"county_name,omitempty"`
	CityID              *int64  `json:"city_id,omitempty"`
	CityName            *string `json:"city_name,omitempty"`
	PropertyCount       *int64  `json:"property_count,omitempty"`
	ActivePropertyCount *int64  `json:"active_property_count,omitempty"`
}

type PostalCodeSEO struct {
	ID                  int64   `json:"id"`
	Code                string  `json:"code"`
	Slug                string  `json:"slug"`
	StateID             *int64  `json:"state_id,omitempty"`
	StateName           *string `json:"state_name,omitempty"`
	StateAbbr           *string `json:"state_abbr,omitempty"`
	PropertyCount       *int64  `json:"property_count,omitempty"`
	ActivePropertyCount *int64  `json:"active_property_count,omitempty"`
	Latitude            *string `json:"latitude,omitempty"`
	Longitude           *string `json:"longitude,omitempty"`
}

func (h *Handler) ListCountiesByStateOrdered(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid state ID", http.StatusBadRequest)
		return
	}

	if h.Pool == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	var counties []CountySEO
	err = pgxscan.Select(ctx, h.Pool, &counties, h.Registry.SQL("ListCountiesByStateOrdered"), id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if counties == nil {
		counties = []CountySEO{}
	}

	h.trackSEOEvent(ctx, "counties_by_state_ordered", id, len(counties), time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := json.NewEncoder(w).Encode(counties); err != nil {
		log.Printf("[seo] failed to encode counties response: %v", err)
	}
}

func (h *Handler) ListCitiesByCountyOrdered(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid county ID", http.StatusBadRequest)
		return
	}

	if h.Pool == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	var cities []CitySEO
	err = pgxscan.Select(ctx, h.Pool, &cities, h.Registry.SQL("ListCitiesByCountyOrdered"), id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if cities == nil {
		cities = []CitySEO{}
	}

	h.trackSEOEvent(ctx, "cities_by_county_ordered", id, len(cities), time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := json.NewEncoder(w).Encode(cities); err != nil {
		log.Printf("[seo] failed to encode cities response: %v", err)
	}
}

func (h *Handler) ListCitiesByStateOrdered(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid state ID", http.StatusBadRequest)
		return
	}

	if h.Pool == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	var cities []CitySEO
	err = pgxscan.Select(ctx, h.Pool, &cities, h.Registry.SQL("ListCitiesByStateOrdered"), id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if cities == nil {
		cities = []CitySEO{}
	}

	h.trackSEOEvent(ctx, "cities_by_state_ordered", id, len(cities), time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := json.NewEncoder(w).Encode(cities); err != nil {
		log.Printf("[seo] failed to encode cities response: %v", err)
	}
}

func (h *Handler) ListSubdivisionsByCountyOrdered(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid county ID", http.StatusBadRequest)
		return
	}

	if h.Pool == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	var subdivisions []SubdivisionSEO
	err = pgxscan.Select(ctx, h.Pool, &subdivisions, h.Registry.SQL("ListSubdivisionsByCountyOrdered"), id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if subdivisions == nil {
		subdivisions = []SubdivisionSEO{}
	}

	h.trackSEOEvent(ctx, "subdivisions_by_county_ordered", id, len(subdivisions), time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := json.NewEncoder(w).Encode(subdivisions); err != nil {
		log.Printf("[seo] failed to encode subdivisions response: %v", err)
	}
}

func (h *Handler) ListSubdivisionsByCityOrdered(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid city ID", http.StatusBadRequest)
		return
	}

	if h.Pool == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	var subdivisions []SubdivisionSEO
	err = pgxscan.Select(ctx, h.Pool, &subdivisions, h.Registry.SQL("ListSubdivisionsByCityOrdered"), id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if subdivisions == nil {
		subdivisions = []SubdivisionSEO{}
	}

	h.trackSEOEvent(ctx, "subdivisions_by_city_ordered", id, len(subdivisions), time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := json.NewEncoder(w).Encode(subdivisions); err != nil {
		log.Printf("[seo] failed to encode subdivisions response: %v", err)
	}
}

func (h *Handler) ListPostalCodesByStateOrdered(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid state ID", http.StatusBadRequest)
		return
	}

	if h.Pool == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	var postalCodes []PostalCodeSEO
	err = pgxscan.Select(ctx, h.Pool, &postalCodes, h.Registry.SQL("ListPostalCodesByStateOrdered"), id)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if postalCodes == nil {
		postalCodes = []PostalCodeSEO{}
	}

	h.trackSEOEvent(ctx, "postal_codes_by_state_ordered", id, len(postalCodes), time.Since(start).Milliseconds())

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := json.NewEncoder(w).Encode(postalCodes); err != nil {
		log.Printf("[seo] failed to encode postal codes response: %v", err)
	}
}
