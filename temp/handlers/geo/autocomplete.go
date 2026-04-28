package geo

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/analytics"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/normalize"
)

const (
	defaultAutocompleteLimit = 15
	maxAutocompleteLimit     = 50
	defaultListLimit         = 500
	minTextQueryLen          = 2
	minPostalQueryLen        = 2
	geoSimilarityThreshold   = 0.30
)

type Handler struct {
	Pool      *pgxpool.Pool
	Registry  *db.Registry
	Analytics *analytics.Client
}

func NewHandler(pool *pgxpool.Pool, registry *db.Registry) *Handler {
	return &Handler{Pool: pool, Registry: registry}
}

// WithAnalytics sets the analytics client for the handler.
func (h *Handler) WithAnalytics(client *analytics.Client) *Handler {
	h.Analytics = client
	return h
}

// trackAutocompleteEvent sends the go_api_autocomplete_served analytics event.
func (h *Handler) trackAutocompleteEvent(ctx context.Context, autocompleteType string, queryLen int, resultsCount int, durationMs int64) {
	if h.Analytics == nil {
		return
	}
	props := analytics.AutocompleteProperties(autocompleteType, queryLen, resultsCount, durationMs)
	h.Analytics.CaptureWithCorrelation(ctx, analytics.EventAPIAutocompleteServed, props)
}

type autocompleteItem struct {
	ID             int64   `db:"id" json:"id"`
	Name           string  `db:"name" json:"name"`
	ParentName     string  `db:"parent_name" json:"parent_name,omitempty"`
	NormalizedName string  `db:"normalized_name" json:"normalized_name"`
	StateID        *int64  `db:"state_id" json:"state_id,omitempty"`
	CountyID       *int64  `db:"county_id" json:"county_id,omitempty"`
	CityID         *int64  `db:"city_id" json:"city_id,omitempty"`
	CityIDs        []int64 `json:"city_ids,omitempty"`
	SubdivisionID  *int64  `db:"subdivision_id" json:"subdivision_id,omitempty"`
	Type           string  `db:"geo_type" json:"type,omitempty"`
}

type autocompleteResponse struct {
	Query   string             `json:"query"`
	Limit   int                `json:"limit"`
	Results []autocompleteItem `json:"results"`
}

func (h *Handler) AutocompleteStates(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	query, limit, ok := parseTextQuery(w, r)
	if !ok {
		return
	}
	rawQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeStates, len(rawQuery), 0, time.Since(start).Milliseconds())
		writeJSON(w, http.StatusOK, autocompleteResponse{Query: rawQuery, Limit: limit, Results: []autocompleteItem{}})
		return
	}

	var results []autocompleteItem
	err := pgxscan.Select(ctx, h.Pool, &results, h.Registry.SQL("autocomplete_states"), query, geoSimilarityThreshold, limit)
	if err != nil {
		http.Error(w, "failed to fetch states", http.StatusInternalServerError)
		return
	}

	h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeStates, len(query), len(results), time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, autocompleteResponse{Query: query, Limit: limit, Results: results})
}

func (h *Handler) AutocompleteCounties(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	rawQuery := strings.TrimSpace(r.URL.Query().Get("q"))

	// List mode: no query text, return all counties filtered by state_id
	if rawQuery == "" {
		stateID, ok := parseOptionalID(w, r, "state_id")
		if !ok {
			return
		}
		if stateID == nil {
			writeJSON(w, http.StatusOK, []autocompleteItem{})
			return
		}
		var results []autocompleteItem
		err := pgxscan.Select(ctx, h.Pool, &results, h.Registry.SQL("list_counties_by_state"), *stateID, defaultListLimit)
		if err != nil {
			log.Printf("[autocomplete] list_counties_by_state error: state_id=%d err=%v", *stateID, err)
			http.Error(w, "failed to fetch counties", http.StatusInternalServerError)
			return
		}
		h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeCounties, 0, len(results), time.Since(start).Milliseconds())
		writeJSON(w, http.StatusOK, results)
		return
	}

	// Autocomplete mode: search by text
	query, limit, ok := parseTextQuery(w, r)
	if !ok {
		return
	}
	stateID, ok := parseOptionalID(w, r, "state_id")
	if !ok {
		return
	}
	if query == "" {
		h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeCounties, len(rawQuery), 0, time.Since(start).Milliseconds())
		writeJSON(w, http.StatusOK, autocompleteResponse{Query: rawQuery, Limit: limit, Results: []autocompleteItem{}})
		return
	}

	var results []autocompleteItem
	err := pgxscan.Select(ctx, h.Pool, &results, h.Registry.SQL("autocomplete_counties"), query, stateID, geoSimilarityThreshold, limit)
	if err != nil {
		http.Error(w, "failed to fetch counties", http.StatusInternalServerError)
		return
	}

	h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeCounties, len(query), len(results), time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, autocompleteResponse{Query: query, Limit: limit, Results: results})
}

func (h *Handler) AutocompleteCities(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	rawQuery := strings.TrimSpace(r.URL.Query().Get("q"))

	// List mode: no query text, return all cities filtered by county_id
	if rawQuery == "" {
		countyID, ok := parseOptionalID(w, r, "county_id")
		if !ok {
			return
		}
		if countyID == nil {
			writeJSON(w, http.StatusOK, []autocompleteItem{})
			return
		}
		var results []autocompleteItem
		err := pgxscan.Select(ctx, h.Pool, &results, h.Registry.SQL("list_cities_by_county"), *countyID, defaultListLimit)
		if err != nil {
			log.Printf("[autocomplete] list_cities_by_county error: county_id=%d err=%v", *countyID, err)
			http.Error(w, "failed to fetch cities", http.StatusInternalServerError)
			return
		}
		h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeCities, 0, len(results), time.Since(start).Milliseconds())
		writeJSON(w, http.StatusOK, results)
		return
	}

	// Autocomplete mode: search by text
	query, limit, ok := parseTextQuery(w, r)
	if !ok {
		return
	}
	stateID, ok := parseOptionalID(w, r, "state_id")
	if !ok {
		return
	}
	countyID, ok := parseOptionalID(w, r, "county_id")
	if !ok {
		return
	}
	if query == "" {
		h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeCities, len(rawQuery), 0, time.Since(start).Milliseconds())
		writeJSON(w, http.StatusOK, autocompleteResponse{Query: rawQuery, Limit: limit, Results: []autocompleteItem{}})
		return
	}

	var results []autocompleteItem
	err := pgxscan.Select(ctx, h.Pool, &results, h.Registry.SQL("autocomplete_cities"), query, stateID, countyID, geoSimilarityThreshold, limit)
	if err != nil {
		http.Error(w, "failed to fetch cities", http.StatusInternalServerError)
		return
	}

	h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeCities, len(query), len(results), time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, autocompleteResponse{Query: query, Limit: limit, Results: results})
}

// AutocompleteSubdivisions returns individual subdivisions matching the query.
func (h *Handler) AutocompleteSubdivisions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	query, limit, ok := parseTextQuery(w, r)
	if !ok {
		return
	}
	stateID, ok := parseOptionalID(w, r, "state_id")
	if !ok {
		return
	}
	countyID, ok := parseOptionalID(w, r, "county_id")
	if !ok {
		return
	}
	cityID, ok := parseOptionalID(w, r, "city_id")
	if !ok {
		return
	}
	rawQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeSubdivisions, len(rawQuery), 0, time.Since(start).Milliseconds())
		writeJSON(w, http.StatusOK, autocompleteResponse{Query: rawQuery, Limit: limit, Results: []autocompleteItem{}})
		return
	}

	var results []autocompleteItem
	err := pgxscan.Select(ctx, h.Pool, &results, h.Registry.SQL("autocomplete_subdivisions"),
		query, stateID, countyID, cityID, geoSimilarityThreshold, limit)
	if err != nil {
		http.Error(w, "failed to fetch subdivisions", http.StatusInternalServerError)
		return
	}

	for i := range results {
		results[i].Type = "subdivision"
	}

	h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeSubdivisions, len(query), len(results), time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, autocompleteResponse{Query: query, Limit: limit, Results: results})
}

func (h *Handler) AutocompletePostalCodes(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	query, limit, ok := parsePostalQuery(w, r)
	if !ok {
		return
	}
	stateID, ok := parseOptionalID(w, r, "state_id")
	if !ok {
		return
	}
	rawQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypePostalCodes, len(rawQuery), 0, time.Since(start).Milliseconds())
		writeJSON(w, http.StatusOK, autocompleteResponse{Query: rawQuery, Limit: limit, Results: []autocompleteItem{}})
		return
	}

	var results []autocompleteItem
	err := pgxscan.Select(ctx, h.Pool, &results, h.Registry.SQL("autocomplete_postal_codes"), query, geoSimilarityThreshold, stateID, limit)
	if err != nil {
		http.Error(w, "failed to fetch postal codes", http.StatusInternalServerError)
		return
	}

	h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypePostalCodes, len(query), len(results), time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, autocompleteResponse{Query: query, Limit: limit, Results: results})
}

func (h *Handler) AutocompleteGeography(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	textQuery, postalQuery, limit, ok := parseUnifiedQuery(w, r)
	if !ok {
		return
	}
	rawQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	if textQuery == "" && postalQuery == "" {
		h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeUnified, len(rawQuery), 0, time.Since(start).Milliseconds())
		writeJSON(w, http.StatusOK, autocompleteResponse{Query: rawQuery, Limit: limit, Results: []autocompleteItem{}})
		return
	}
	stateID, ok := parseOptionalID(w, r, "state_id")
	if !ok {
		return
	}
	countyID, ok := parseOptionalID(w, r, "county_id")
	if !ok {
		return
	}
	cityID, ok := parseOptionalID(w, r, "city_id")
	if !ok {
		return
	}

	var candidates []unifiedCandidate
	if textQuery != "" {
		var states []autocompleteItem
		if err := pgxscan.Select(ctx, h.Pool, &states, h.Registry.SQL("autocomplete_states"), textQuery, geoSimilarityThreshold, limit); err != nil {
			http.Error(w, "failed to fetch states", http.StatusInternalServerError)
			return
		}
		addUnifiedCandidates(&candidates, states, "state", textQuery)

		var counties []autocompleteItem
		if err := pgxscan.Select(ctx, h.Pool, &counties, h.Registry.SQL("autocomplete_counties"), textQuery, stateID, geoSimilarityThreshold, limit); err != nil {
			http.Error(w, "failed to fetch counties", http.StatusInternalServerError)
			return
		}
		addUnifiedCandidates(&candidates, counties, "county", textQuery)

		var cities []autocompleteItem
		if err := pgxscan.Select(ctx, h.Pool, &cities, h.Registry.SQL("autocomplete_cities"), textQuery, stateID, countyID, geoSimilarityThreshold, limit); err != nil {
			http.Error(w, "failed to fetch cities", http.StatusInternalServerError)
			return
		}
		addUnifiedCandidates(&candidates, cities, "city", textQuery)

		var subdivisions []autocompleteItem
		if err := pgxscan.Select(ctx, h.Pool, &subdivisions, h.Registry.SQL("autocomplete_subdivisions"), textQuery, stateID, countyID, cityID, geoSimilarityThreshold, limit); err != nil {
			http.Error(w, "failed to fetch subdivisions", http.StatusInternalServerError)
			return
		}
		addUnifiedCandidates(&candidates, subdivisions, "subdivision", textQuery)
	}

	if postalQuery != "" {
		var postals []autocompleteItem
		if err := pgxscan.Select(ctx, h.Pool, &postals, h.Registry.SQL("autocomplete_postal_codes"), postalQuery, geoSimilarityThreshold, stateID, limit); err != nil {
			http.Error(w, "failed to fetch postal codes", http.StatusInternalServerError)
			return
		}
		addUnifiedCandidates(&candidates, postals, "postal_code", postalQuery)
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].prefix != candidates[j].prefix {
			return candidates[i].prefix
		}
		if candidates[i].Type != candidates[j].Type {
			return candidates[i].Type < candidates[j].Type
		}
		return candidates[i].Name < candidates[j].Name
	})

	results := make([]autocompleteItem, 0, minInt(limit, len(candidates)))
	for i := 0; i < len(candidates) && i < limit; i++ {
		results = append(results, candidates[i].autocompleteItem)
	}

	h.trackAutocompleteEvent(ctx, analytics.AutocompleteTypeUnified, len(rawQuery), len(results), time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, autocompleteResponse{Query: rawQuery, Limit: limit, Results: results})
}

func parseTextQuery(w http.ResponseWriter, r *http.Request) (string, int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("q"))
	if raw == "" {
		http.Error(w, "missing query", http.StatusBadRequest)
		return "", 0, false
	}
	normalized := normalize.Name(raw)
	limit, err := parseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, "invalid limit", http.StatusBadRequest)
		return "", 0, false
	}
	if len(normalized) < minTextQueryLen {
		return "", limit, true
	}
	return normalized, limit, true
}

func parseUnifiedQuery(w http.ResponseWriter, r *http.Request) (string, string, int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("q"))
	if raw == "" {
		http.Error(w, "missing query", http.StatusBadRequest)
		return "", "", 0, false
	}
	normalized := normalize.Name(raw)
	postal := normalizePostal(raw)
	limit, err := parseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, "invalid limit", http.StatusBadRequest)
		return "", "", 0, false
	}
	if len(normalized) < minTextQueryLen {
		normalized = ""
	}
	if len(postal) < minPostalQueryLen {
		postal = ""
	}
	return normalized, postal, limit, true
}

type unifiedCandidate struct {
	autocompleteItem
	prefix bool
}

func addUnifiedCandidates(out *[]unifiedCandidate, items []autocompleteItem, geoType string, query string) {
	for _, item := range items {
		item.Type = geoType
		normalized := strings.TrimSpace(item.NormalizedName)
		prefix := normalized != "" && strings.HasPrefix(normalized, query)
		*out = append(*out, unifiedCandidate{autocompleteItem: item, prefix: prefix})
	}
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func parsePostalQuery(w http.ResponseWriter, r *http.Request) (string, int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("q"))
	if raw == "" {
		http.Error(w, "missing query", http.StatusBadRequest)
		return "", 0, false
	}
	normalized := normalizePostal(raw)
	limit, err := parseLimit(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, "invalid limit", http.StatusBadRequest)
		return "", 0, false
	}
	if len(normalized) < minPostalQueryLen {
		return "", limit, true
	}
	return normalized, limit, true
}

func parseOptionalID(w http.ResponseWriter, r *http.Request, key string) (*int64, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, true
	}
	val, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		http.Error(w, "invalid "+key, http.StatusBadRequest)
		return nil, false
	}
	return &val, true
}

func parseLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultAutocompleteLimit, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 0, errInvalidLimit
	}
	if limit > maxAutocompleteLimit {
		return maxAutocompleteLimit, nil
	}
	return limit, nil
}

var errInvalidLimit = errors.New("invalid limit")

func normalizePostal(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
