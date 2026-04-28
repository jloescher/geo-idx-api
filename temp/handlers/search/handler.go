package search

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/analytics"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/config"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/cryptoquotes"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/fxrates"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/http/handlers/rto"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/mls"
	searchsvc "github.com/xotec-solutions/xotec-datalayer/src/internal/search"
)

const maxSearchBodyBytes = 2 << 20 // 2MB

// Handler serves the search API.
type Handler struct {
	Store          *searchsvc.Store
	Analytics      *analytics.Client
	CryptoSvc      *cryptoquotes.Service
	FXSvc          *fxrates.Service
	CryptoAssets   []cryptoquotes.Asset
	SupportedFiats []string
}

func NewHandler(pool *pgxpool.Pool, registry *db.Registry, cfg config.Config) *Handler {
	return &Handler{Store: searchsvc.NewStore(pool, registry, cfg.MediaCDNProdHost)}
}

// WithAnalytics sets the analytics client for the handler.
func (h *Handler) WithAnalytics(client *analytics.Client) *Handler {
	h.Analytics = client
	return h
}

// WithCurrencyServices sets the digital asset and FX services for the handler.
func (h *Handler) WithCurrencyServices(
	cryptoSvc *cryptoquotes.Service,
	fxSvc *fxrates.Service,
	cryptoAssets []cryptoquotes.Asset,
	supportedFiats []string,
) *Handler {
	h.CryptoSvc = cryptoSvc
	h.FXSvc = fxSvc
	h.CryptoAssets = cryptoAssets
	h.SupportedFiats = supportedFiats
	return h
}

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type searchResponse struct {
	Status         string                    `json:"status"`
	Message        string                    `json:"message,omitempty"`
	Results        []searchsvc.ListingResult `json:"results"`
	HasMore        bool                      `json:"has_more"`
	NextCursor     *string                   `json:"next_cursor"`
	Count          int64                     `json:"count"`
	Stats          searchsvc.SearchStats     `json:"stats"`
	CurrencyQuotes map[string]float64        `json:"currency_quotes,omitempty"`
}

type listingResponse struct {
	Status  string                    `json:"status"`
	Message string                    `json:"message,omitempty"`
	Listing *searchsvc.ListingDetails `json:"listing,omitempty"`
}

// listingsBasicResponse is returned when requesting multiple listings via listing_ids array.
type listingsBasicResponse struct {
	Status   string                   `json:"status"`
	Message  string                   `json:"message,omitempty"`
	Listings []searchsvc.ListingBasic `json:"listings"`
}

const maxListingIDsPerRequest = 50

// HandleListing handles POST /api/v1/listings.
// Supports both single listing_id and array of listing_ids:
//   - Single: {"listing_id": "MFR1234567"} → returns full ListingDetails
//   - Array: {"listing_ids": ["MFR1234567", "MFR7654321"]} → returns basic ListingBasic data
func (h *Handler) HandleListing(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	body := http.MaxBytesReader(w, r.Body, maxSearchBodyBytes)
	defer body.Close()

	payload, err := io.ReadAll(body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body too large", nil)
			return
		}
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request", map[string]string{"request": "invalid body"})
		return
	}

	// Try to parse as array request first
	var arrayReq struct {
		ListingIDs []string `json:"listing_ids"`
	}
	if err := json.Unmarshal(payload, &arrayReq); err == nil && len(arrayReq.ListingIDs) > 0 {
		h.handleListingIDs(w, r, ctx, arrayReq.ListingIDs, start)
		return
	}

	// Fall back to single listing request
	var req struct {
		ListingID string `json:"listing_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request", map[string]string{"request": "invalid payload"})
		return
	}

	listingID := mls.AddPrefix(strings.TrimSpace(req.ListingID))
	if listingID == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request", map[string]string{"listing_id": "required"})
		return
	}

	details, found, err := h.Store.GetListingDetails(ctx, listingID)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		requestID := requestIDFromHeaders(r)
		log.Printf("HandleListing: listing_id=%s request_id=%s error=%v", listingID, requestID, err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		return
	}
	if !found {
		h.trackListingEvent(ctx, listingID, false, durationMs)
		writeJSON(w, http.StatusOK, listingResponse{
			Status:  "no_matches",
			Message: "No matches found",
			Listing: nil,
		})
		return
	}

	h.trackListingEvent(ctx, listingID, true, durationMs)
	writeJSON(w, http.StatusOK, listingResponse{
		Status:  "ok",
		Listing: &details,
	})
}

// handleListingIDs processes batch listing requests with listing_ids array.
func (h *Handler) handleListingIDs(w http.ResponseWriter, r *http.Request, ctx context.Context, rawIDs []string, start time.Time) {
	// Validate and normalize IDs
	if len(rawIDs) == 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request", map[string]string{"listing_ids": "must not be empty"})
		return
	}
	if len(rawIDs) > maxListingIDsPerRequest {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request", map[string]string{"listing_ids": "maximum 50 IDs per request"})
		return
	}

	// Normalize IDs with prefix
	listingIDs := make([]string, 0, len(rawIDs))
	for _, id := range rawIDs {
		normalized := mls.AddPrefix(strings.TrimSpace(id))
		if normalized != "" {
			listingIDs = append(listingIDs, normalized)
		}
	}

	if len(listingIDs) == 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request", map[string]string{"listing_ids": "no valid listing IDs"})
		return
	}

	listings, err := h.Store.GetListingBasics(ctx, listingIDs)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		requestID := requestIDFromHeaders(r)
		log.Printf("HandleListing: listing_ids batch request_id=%s count=%d error=%v", requestID, len(listingIDs), err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		return
	}

	// Track batch event (use first ID for tracking, mark as batch)
	if len(listings) > 0 && h.Analytics != nil {
		h.trackListingBatchEvent(ctx, listingIDs, len(listings), durationMs)
	}

	if len(listings) == 0 {
		writeJSON(w, http.StatusOK, listingsBasicResponse{
			Status:   "no_matches",
			Message:  "No matches found",
			Listings: []searchsvc.ListingBasic{},
		})
		return
	}

	writeJSON(w, http.StatusOK, listingsBasicResponse{
		Status:   "ok",
		Listings: listings,
	})
}

// HandleSearch handles POST /api/v1/search.
func (h *Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	body := http.MaxBytesReader(w, r.Body, maxSearchBodyBytes)
	defer body.Close()

	payload, err := io.ReadAll(body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body too large", nil)
			return
		}
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request", map[string]string{"request": "invalid body"})
		return
	}

	req, err := searchsvc.ParseSearchPayload(payload)
	if err != nil {
		writeValidationError(w, err)
		return
	}

	effectiveLimit := searchsvc.EffectiveLimit(req.Page)
	if effectiveLimit == 0 {
		h.trackSearchEvent(ctx, req, 0, time.Since(start).Milliseconds())
		writeJSON(w, http.StatusOK, searchResponse{
			Status:  "no_matches",
			Message: "No matches found",
			Results: []searchsvc.ListingResult{},
			HasMore: false,
			Count:   0,
			Stats:   searchsvc.SearchStats{},
		})
		return
	}

	result, err := h.Store.Search(ctx, req, effectiveLimit)
	if err != nil {
		var vErr *searchsvc.ValidationError
		if errors.As(err, &vErr) {
			writeError(w, http.StatusBadRequest, "validation_error", "Invalid request", vErr.Fields)
			return
		}
		log.Printf("HandleSearch: store.Search failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		return
	}

	// Enrich results with RTO estimates when requested.
	if req.IncludeRTO {
		for i := range result.Results {
			r := &result.Results[i]
			if r.ListPrice != nil {
				hybrid := rto.ComputeHybridForSearch(*r.ListPrice, r.TaxAnnualAmount)
				r.RTOEstimate = &searchsvc.RTOEstimate{
					EstimatedMonthly:       hybrid.EstimatedMonthly,
					EstimatedPurchasePrice: hybrid.EstimatedPurchasePrice,
				}
			}
		}
	}

	durationMs := time.Since(start).Milliseconds()

	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=120")

	if result.Stats.TotalCount == 0 {
		h.trackSearchEvent(ctx, req, 0, durationMs)
		resp := searchResponse{
			Status:     "no_matches",
			Message:    "No matches found",
			Results:    []searchsvc.ListingResult{},
			HasMore:    false,
			NextCursor: nil,
			Count:      0,
			Stats:      result.Stats,
		}
		// Include currency quotes if requested
		if len(req.CurrencyTickers) > 0 {
			resp.CurrencyQuotes = h.getCurrencyQuotes(ctx, req.CurrencyTickers)
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	h.trackSearchEvent(ctx, req, int(result.Stats.TotalCount), durationMs)
	resp := searchResponse{
		Status:     "ok",
		Results:    result.Results,
		HasMore:    result.HasMore,
		NextCursor: result.NextCursor,
		Count:      result.Stats.TotalCount,
		Stats:      result.Stats,
	}
	// Include currency quotes if requested
	if len(req.CurrencyTickers) > 0 {
		resp.CurrencyQuotes = h.getCurrencyQuotes(ctx, req.CurrencyTickers)
	}
	writeJSON(w, http.StatusOK, resp)
}

// searchCountResponse represents the response for the count endpoint.
type searchCountResponse struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// HandleSearchCount handles POST /api/v1/search/count.
// It accepts the same payload as /api/v1/search but returns only the count of matching properties.
func (h *Handler) HandleSearchCount(w http.ResponseWriter, r *http.Request) {
	// start := time.Now() // Metrics if needed
	ctx := r.Context()

	body := http.MaxBytesReader(w, r.Body, maxSearchBodyBytes)
	defer body.Close()

	payload, err := io.ReadAll(body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body too large", nil)
			return
		}
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request", map[string]string{"request": "invalid body"})
		return
	}

	req, err := searchsvc.ParseSearchPayload(payload)
	if err != nil {
		writeValidationError(w, err)
		return
	}

	count, err := h.Store.Count(ctx, req)
	if err != nil {
		var vErr *searchsvc.ValidationError
		if errors.As(err, &vErr) {
			writeError(w, http.StatusBadRequest, "validation_error", "Invalid request", vErr.Fields)
			return
		}
		// Log the actual error for debugging
		log.Printf("HandleSearchCount: store.Count failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=120")
	writeJSON(w, http.StatusOK, searchCountResponse{
		Status: "ok",
		Count:  count,
	})
}

// trackSearchEvent sends the go_api_search_executed analytics event.
func (h *Handler) trackSearchEvent(ctx context.Context, req searchsvc.SearchRequest, resultsCount int, durationMs int64) {
	if h.Analytics == nil {
		return
	}

	// Determine geo_type from request
	geoType := analytics.GeoTypeNone
	if req.Params.Geo != nil && req.Params.Geo.Polygon != nil {
		geoType = analytics.GeoTypePolygon
	} else if len(req.LocationFilters.CityRefIDs.Values) > 0 {
		geoType = analytics.GeoTypeCity
	} else if len(req.LocationFilters.PostalCodeRefIDs.Values) > 0 {
		geoType = analytics.GeoTypeZip
	} else if len(req.LocationFilters.CountyRefIDs.Values) > 0 {
		geoType = analytics.GeoTypeCounty
	}

	// Count filters applied
	filtersCount := countFilters(req)

	props := analytics.SearchProperties("search", filtersCount, resultsCount, geoType, durationMs)
	h.Analytics.CaptureWithCorrelation(ctx, analytics.EventAPISearchExecuted, props)
}

// countFilters counts the number of filters applied in the request.
func countFilters(req searchsvc.SearchRequest) int {
	count := 0
	p := req.Params
	l := req.LocationFilters

	// Property filters
	if len(req.ListingTypes) > 0 {
		count++
	}
	if len(p.Statuses) > 0 {
		count++
	}
	if len(p.PropertySubType) > 0 {
		count++
	}

	// Price range
	if p.MinPrice.Value != nil || p.MaxPrice.Value != nil {
		count++
	}
	// Beds range
	if p.MinBeds.Value != nil || p.MaxBeds.Value != nil {
		count++
	}
	// Baths range
	if p.MinBaths.Value != nil || p.MaxBaths.Value != nil {
		count++
	}
	// Sqft range
	if p.MinSqft.Value != nil || p.MaxSqft.Value != nil {
		count++
	}
	// Year built range
	if p.MinYearBuilt.Value != nil || p.MaxYearBuilt.Value != nil {
		count++
	}
	// Lot size range
	if p.MinLotSizeAcres.Value != nil || p.MaxLotSizeAcres.Value != nil {
		count++
	}
	// DOM range
	if p.MinDOM.Value != nil || p.MaxDOM.Value != nil {
		count++
	}

	// Boolean filters
	if p.PoolPrivate.Value != nil {
		count++
	}
	if p.Waterfront.Value != nil {
		count++
	}
	if p.Dock.Value != nil {
		count++
	}
	if p.NewConstruction.Value != nil {
		count++
	}

	// Location filters
	if len(l.StateRefIDs.Values) > 0 {
		count++
	}
	if len(l.CountyRefIDs.Values) > 0 {
		count++
	}
	if len(l.CityRefIDs.Values) > 0 {
		count++
	}
	if len(l.SubdivisionRefIDs.Values) > 0 {
		count++
	}
	if len(l.PostalCodeRefIDs.Values) > 0 {
		count++
	}
	if len(l.ElementarySchoolRefIDs.Values) > 0 {
		count++
	}
	if len(l.MiddleSchoolRefIDs.Values) > 0 {
		count++
	}
	if len(l.HighSchoolRefIDs.Values) > 0 {
		count++
	}

	// Geo filters
	if p.Geo != nil {
		if p.Geo.Distance != nil {
			count++
		}
		if p.Geo.BBox != nil {
			count++
		}
		if p.Geo.Polygon != nil {
			count++
		}
	}

	return count
}

func requestIDFromHeaders(r *http.Request) string {
	requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
	if requestID == "" {
		requestID = strings.TrimSpace(r.Header.Get("X-Correlation-Id"))
	}
	return requestID
}

// trackListingEvent sends the go_api_listing_fetched analytics event.
func (h *Handler) trackListingEvent(ctx context.Context, listingID string, found bool, durationMs int64) {
	if h.Analytics == nil {
		return
	}

	cacheStatus := "miss" // For now, always miss since we don't cache listing details
	props := analytics.ListingProperties(listingID, cacheStatus, durationMs).
		Set("found", found)
	h.Analytics.CaptureWithCorrelation(ctx, analytics.EventAPIListingFetched, props)
}

// trackListingBatchEvent sends analytics event for batch listing requests.
func (h *Handler) trackListingBatchEvent(ctx context.Context, listingIDs []string, foundCount int, durationMs int64) {
	if h.Analytics == nil {
		return
	}

	// Use first listing ID for correlation, but track batch size
	firstID := ""
	if len(listingIDs) > 0 {
		firstID = listingIDs[0]
	}

	props := analytics.ListingProperties(firstID, "miss", durationMs).
		Set("found_count", foundCount).
		Set("requested_count", len(listingIDs)).
		Set("batch", true)
	h.Analytics.CaptureWithCorrelation(ctx, analytics.EventAPIListingFetched, props)
}

func writeValidationError(w http.ResponseWriter, err error) {
	var vErr *searchsvc.ValidationError
	if errors.As(err, &vErr) {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request", vErr.Fields)
		return
	}
	writeError(w, http.StatusBadRequest, "validation_error", "Invalid request", map[string]string{"request": "invalid payload"})
}

func writeError(w http.ResponseWriter, status int, code string, message string, fields map[string]string) {
	writeJSON(w, status, errorResponse{Error: errorBody{Code: code, Message: message, Fields: fields}})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// =====================================================
// SCHOOL ZONE SEARCH ENDPOINTS
// =====================================================

// SchoolZonePropertyResult represents a property within a school zone
type SchoolZonePropertyResult struct {
	ID              int64   `json:"id" db:"id"`
	ListingID       string  `json:"listing_id" db:"listing_id"`
	Slug            *string `json:"slug,omitempty" db:"slug"`
	ListPrice       *int64  `json:"list_price,omitempty" db:"list_price"`
	BedroomsTotal   *int    `json:"bedrooms_total,omitempty" db:"bedrooms_total"`
	BathroomsTotal  *int    `json:"bathrooms_total,omitempty" db:"bathrooms_total"`
	LivingArea      *int    `json:"living_area,omitempty" db:"living_area"`
	City            *string `json:"city,omitempty" db:"city"`
	SubdivisionName *string `json:"subdivision_name,omitempty" db:"subdivision_name"`
}

// SchoolZoneSearchResponse wraps the response for search by school zone
type SchoolZoneSearchResponse struct {
	Status     string                     `json:"status"`
	Message    string                     `json:"message,omitempty"`
	ZoneID     int64                      `json:"zone_id"`
	ZoneName   string                     `json:"zone_name,omitempty"`
	ZoneType   string                     `json:"zone_type,omitempty"`
	SchoolName string                     `json:"school_name,omitempty"`
	Results    []SchoolZonePropertyResult `json:"results"`
	Total      int64                      `json:"total"`
	Limit      int                        `json:"limit"`
	Offset     int                        `json:"offset"`
}

// HandleSearchBySchoolZone handles GET /api/v1/search/by-school-zone
// Query params: zone_id (required), limit (optional, default 50), offset (optional, default 0)
func (h *Handler) HandleSearchBySchoolZone(w http.ResponseWriter, r *http.Request) {
	zoneIDStr := r.URL.Query().Get("zone_id")
	if zoneIDStr == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "zone_id is required", map[string]string{"zone_id": "required"})
		return
	}

	zoneID, err := strconv.ParseInt(zoneIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid zone_id", map[string]string{"zone_id": "must be a valid integer"})
		return
	}

	// Parse limit and offset
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	ctx := r.Context()

	// First get the zone info for the response
	zoneInfoQuery := h.Store.Registry.SQL("GetSchoolZoneBoundaryGeoJSON")
	var zoneName, zoneType string
	var schoolName *string
	var ignoreID int64
	var ignoreSchoolID *int64
	var ignoreCounty string
	var ignoreGeom []byte
	var ignoreArea, ignoreLng, ignoreLat *float64

	err = h.Store.Pool.QueryRow(ctx, zoneInfoQuery, zoneID).Scan(
		&ignoreID, &zoneName, &zoneType, &ignoreSchoolID, &schoolName,
		&ignoreCounty, &ignoreGeom, &ignoreArea, &ignoreLng, &ignoreLat,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not_found", "School zone not found", nil)
			return
		}
		log.Printf("HandleSearchBySchoolZone: zone lookup failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		return
	}

	// Get total count
	countQuery := h.Store.Registry.SQL("CountPropertiesBySchoolZone")
	var total int64
	err = h.Store.Pool.QueryRow(ctx, countQuery, zoneID).Scan(&total)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to count properties", nil)
		return
	}

	// Get properties
	searchQuery := h.Store.Registry.SQL("SearchPropertiesBySchoolZone")
	rows, err := h.Store.Pool.Query(ctx, searchQuery, zoneID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Search failed", nil)
		return
	}
	defer rows.Close()

	results := []SchoolZonePropertyResult{}
	for rows.Next() {
		var p SchoolZonePropertyResult
		if err := rows.Scan(&p.ID, &p.ListingID, &p.Slug, &p.ListPrice, &p.BedroomsTotal, &p.BathroomsTotal, &p.LivingArea, &p.City, &p.SubdivisionName); err != nil {
			log.Printf("HandleSearchBySchoolZone: scan failed: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
			return
		}
		p.ListingID = mls.StripPrefix(p.ListingID)
		results = append(results, p)
	}
	if err := rows.Err(); err != nil {
		log.Printf("HandleSearchBySchoolZone: rows iteration failed: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		return
	}

	resp := SchoolZoneSearchResponse{
		Status:   "ok",
		ZoneID:   zoneID,
		ZoneName: zoneName,
		ZoneType: zoneType,
		Results:  results,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}
	if schoolName != nil {
		resp.SchoolName = *schoolName
	}
	if total == 0 {
		resp.Status = "no_matches"
		resp.Message = "No active properties found in this school zone"
	}

	w.Header().Set("Cache-Control", "public, max-age=300") // 5 min cache
	writeJSON(w, http.StatusOK, resp)
}

// getCurrencyQuotes fetches prices for the requested currency tickers.
// Supports both digital asset symbols (BTC, ETH, SOL, BNB, XRP) and fiat currencies (USD, EUR).
func (h *Handler) getCurrencyQuotes(ctx context.Context, tickers []string) map[string]float64 {
	if len(tickers) == 0 || (h.CryptoSvc == nil && h.FXSvc == nil) {
		return nil
	}

	quotes := make(map[string]float64)

	// Get digital asset quotes if service is available
	var cryptoPayload *cryptoquotes.CachePayload
	if h.CryptoSvc != nil {
		var err error
		cryptoPayload, err = h.CryptoSvc.GetQuotes(ctx)
		if err != nil {
			log.Printf("search handler: failed to get digital asset quotes: %v", err)
		}
	}

	// Get FX rates if FX service is available
	var fxPayload *fxrates.FXCachePayload
	if h.FXSvc != nil {
		var err error
		fxPayload, err = h.FXSvc.GetRates(ctx)
		if err != nil {
			log.Printf("search handler: failed to get FX rates: %v", err)
		}
	}

	// Build symbol map for quick lookup
	cryptoSymbolMap := make(map[string]cryptoquotes.Asset)
	for _, asset := range h.CryptoAssets {
		cryptoSymbolMap[strings.ToUpper(asset.Symbol)] = asset
	}

	// Process each requested ticker
	for _, ticker := range tickers {
		tickerUpper := strings.ToUpper(strings.TrimSpace(ticker))
		if tickerUpper == "" {
			continue
		}

		// Check if it's a digital asset symbol
		if asset, ok := cryptoSymbolMap[tickerUpper]; ok && cryptoPayload != nil {
			// Find the quote for this asset
			for _, quote := range cryptoPayload.Quotes {
				if quote.ID == asset.ID {
					quotes[tickerUpper] = quote.PriceUSD
					break
				}
			}
		} else if tickerUpper == "USD" {
			// USD is the base currency, always 1.0
			quotes[tickerUpper] = 1.0
		} else if fxPayload != nil {
			// Check if it's a fiat currency
			if rate, ok := fxPayload.Rates.Rates[tickerUpper]; ok {
				quotes[tickerUpper] = rate
			}
		}
	}

	return quotes
}
