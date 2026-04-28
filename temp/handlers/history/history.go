package history

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/analytics"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/db/repository"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/mls"
)

// Handler handles history API requests
type Handler struct {
	Pool      *pgxpool.Pool
	Registry  *db.Registry
	Analytics *analytics.Client
}

// NewHandler creates a new history handler
func NewHandler(pool *pgxpool.Pool, registry *db.Registry) *Handler {
	return &Handler{Pool: pool, Registry: registry}
}

// WithAnalytics sets the analytics client for the handler.
func (h *Handler) WithAnalytics(client *analytics.Client) *Handler {
	h.Analytics = client
	return h
}

// trackHistoryEvent sends the go_api_history_fetched analytics event.
func (h *Handler) trackHistoryEvent(ctx context.Context, historyType string, geoType string, geoID int64, durationMs int64) {
	if h.Analytics == nil {
		return
	}
	props := analytics.HistoryProperties(historyType, geoType, geoID, durationMs)
	h.Analytics.CaptureWithCorrelation(ctx, analytics.EventAPIHistoryFetched, props)
}

// Response types for JSON output

type priceHistoryResponse struct {
	ListingID string                          `json:"listing_id"`
	Records   []repository.PriceHistoryRecord `json:"records"`
}

type statusHistoryResponse struct {
	ListingID string                           `json:"listing_id"`
	Records   []repository.StatusHistoryRecord `json:"records"`
}

type listingTimelineResponse struct {
	ListingID string                            `json:"listing_id"`
	Events    []repository.ListingTimelineEvent `json:"events"`
}

type priceTrendsResponse struct {
	GeoType   string                       `json:"geo_type"`
	GeoID     int64                        `json:"geo_id"`
	StartDate string                       `json:"start_date"`
	EndDate   string                       `json:"end_date"`
	Interval  string                       `json:"interval"`
	Trends    []repository.PriceTrendPoint `json:"trends"`
}

type salesTrendsResponse struct {
	GeoType   string                       `json:"geo_type"`
	GeoID     int64                        `json:"geo_id"`
	StartDate string                       `json:"start_date"`
	EndDate   string                       `json:"end_date"`
	Interval  string                       `json:"interval"`
	Trends    []repository.SalesTrendPoint `json:"trends"`
}

type statusTransitionsResponse struct {
	GeoType     string                             `json:"geo_type"`
	GeoID       int64                              `json:"geo_id"`
	StartDate   string                             `json:"start_date"`
	EndDate     string                             `json:"end_date"`
	Interval    string                             `json:"interval"`
	Transitions []repository.StatusTransitionPoint `json:"transitions"`
}

type listingCountsResponse struct {
	GeoType   string                         `json:"geo_type"`
	GeoID     int64                          `json:"geo_id"`
	StartDate string                         `json:"start_date"`
	EndDate   string                         `json:"end_date"`
	Interval  string                         `json:"interval"`
	Counts    []repository.ListingCountPoint `json:"counts"`
}

type domTrendsResponse struct {
	GeoType   string                     `json:"geo_type"`
	GeoID     int64                      `json:"geo_id"`
	StartDate string                     `json:"start_date"`
	EndDate   string                     `json:"end_date"`
	Interval  string                     `json:"interval"`
	Trends    []repository.DOMTrendPoint `json:"trends"`
}

type historyStatsResponse struct {
	Stats repository.HistoryStatsSummary `json:"stats"`
}

type priceChangeSummaryResponse struct {
	GeoType   string                        `json:"geo_type"`
	GeoID     int64                         `json:"geo_id"`
	StartDate string                        `json:"start_date"`
	EndDate   string                        `json:"end_date"`
	Summary   repository.PriceChangeSummary `json:"summary"`
}

type marketSnapshotFilters struct {
	PropertyType    *string `json:"property_type"`
	PropertySubType *string `json:"property_sub_type"`
}

type marketSnapshotResponse struct {
	GeoType  string                    `json:"geo_type"`
	GeoID    int64                     `json:"geo_id"`
	Filters  marketSnapshotFilters     `json:"filters"`
	Snapshot repository.MarketSnapshot `json:"snapshot"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func parseGeoTrendParams(r *http.Request) (repository.GeoTrendParams, error) {
	params := repository.GeoTrendParams{}

	// Required: geo_type
	params.GeoType = r.URL.Query().Get("geo_type")
	if params.GeoType == "" {
		params.GeoType = "state" // default
	}
	if params.GeoType != "state" && params.GeoType != "county" && params.GeoType != "city" && params.GeoType != "subdivision" && params.GeoType != "postal_code" {
		return params, &paramError{"geo_type must be one of: state, county, city, subdivision, postal_code"}
	}

	// Required: geo_id
	geoIDStr := r.URL.Query().Get("geo_id")
	if geoIDStr == "" {
		return params, &paramError{"geo_id is required"}
	}
	geoID, err := strconv.ParseInt(geoIDStr, 10, 64)
	if err != nil {
		return params, &paramError{"geo_id must be a valid integer"}
	}
	params.GeoID = geoID

	// Optional: start_date (default 30 days ago)
	startDateStr := r.URL.Query().Get("start_date")
	if startDateStr == "" {
		params.StartDate = time.Now().AddDate(0, 0, -30)
	} else {
		t, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return params, &paramError{"start_date must be in YYYY-MM-DD format"}
		}
		params.StartDate = t
	}

	// Optional: end_date (default now)
	endDateStr := r.URL.Query().Get("end_date")
	if endDateStr == "" {
		params.EndDate = time.Now()
	} else {
		t, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return params, &paramError{"end_date must be in YYYY-MM-DD format"}
		}
		params.EndDate = t.Add(24 * time.Hour) // Include the end date
	}

	// Optional: interval (default 'day')
	params.Interval = r.URL.Query().Get("interval")
	if params.Interval == "" {
		params.Interval = "day"
	}
	if params.Interval != "day" && params.Interval != "week" && params.Interval != "month" {
		return params, &paramError{"interval must be one of: day, week, month"}
	}

	return params, nil
}

type paramError struct {
	message string
}

func (e *paramError) Error() string {
	return e.message
}

// Handler methods

// GetPriceHistoryByListing returns price history for a specific listing
// GET /api/v1/history/listing/{listing_id}/prices
func (h *Handler) GetPriceHistoryByListing(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	listingID := mls.AddPrefix(chi.URLParam(r, "listing_id"))
	if listingID == "" {
		writeError(w, http.StatusBadRequest, "listing_id is required")
		return
	}

	records, err := repository.GetPriceHistoryByListing(ctx, h.Pool, listingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch price history")
		return
	}

	if records == nil {
		records = []repository.PriceHistoryRecord{}
	}

	h.trackHistoryEvent(ctx, analytics.HistoryTypePrice, "listing", 0, time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, priceHistoryResponse{
		ListingID: mls.StripPrefix(listingID),
		Records:   records,
	})
}

// GetStatusHistoryByListing returns status history for a specific listing
// GET /api/v1/history/listing/{listing_id}/status
func (h *Handler) GetStatusHistoryByListing(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	listingID := mls.AddPrefix(chi.URLParam(r, "listing_id"))
	if listingID == "" {
		writeError(w, http.StatusBadRequest, "listing_id is required")
		return
	}

	records, err := repository.GetStatusHistoryByListing(ctx, h.Pool, listingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch status history")
		return
	}

	if records == nil {
		records = []repository.StatusHistoryRecord{}
	}

	h.trackHistoryEvent(ctx, analytics.HistoryTypeStatus, "listing", 0, time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, statusHistoryResponse{
		ListingID: mls.StripPrefix(listingID),
		Records:   records,
	})
}

// GetListingTimeline returns full timeline for a specific listing
// GET /api/v1/history/listing/{listing_id}/timeline
func (h *Handler) GetListingTimeline(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	listingID := mls.AddPrefix(chi.URLParam(r, "listing_id"))
	if listingID == "" {
		writeError(w, http.StatusBadRequest, "listing_id is required")
		return
	}

	events, err := repository.GetListingTimeline(ctx, h.Pool, listingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch listing timeline")
		return
	}

	if events == nil {
		events = []repository.ListingTimelineEvent{}
	}

	h.trackHistoryEvent(ctx, analytics.HistoryTypeTimeline, "listing", 0, time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, listingTimelineResponse{
		ListingID: mls.StripPrefix(listingID),
		Events:    events,
	})
}

// GetPriceTrends returns price trends by geography
// GET /api/v1/history/trends/prices?geo_type=city&geo_id=123&start_date=2024-01-01&end_date=2024-12-31&interval=month
func (h *Handler) GetPriceTrends(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	params, err := parseGeoTrendParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	trends, err := repository.GetPriceTrendsByGeo(ctx, h.Pool, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch price trends")
		return
	}

	if trends == nil {
		trends = []repository.PriceTrendPoint{}
	}

	h.trackHistoryEvent(ctx, analytics.HistoryTypeTrends, params.GeoType, params.GeoID, time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, priceTrendsResponse{
		GeoType:   params.GeoType,
		GeoID:     params.GeoID,
		StartDate: params.StartDate.Format("2006-01-02"),
		EndDate:   params.EndDate.Format("2006-01-02"),
		Interval:  params.Interval,
		Trends:    trends,
	})
}

// GetSalesTrends returns sales (close price) trends by geography
// GET /api/v1/history/trends/sales?geo_type=city&geo_id=123&start_date=2024-01-01&end_date=2024-12-31&interval=month
func (h *Handler) GetSalesTrends(w http.ResponseWriter, r *http.Request) {
	params, err := parseGeoTrendParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	trends, err := repository.GetSalesTrendsByGeo(r.Context(), h.Pool, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch sales trends")
		return
	}

	if trends == nil {
		trends = []repository.SalesTrendPoint{}
	}

	writeJSON(w, http.StatusOK, salesTrendsResponse{
		GeoType:   params.GeoType,
		GeoID:     params.GeoID,
		StartDate: params.StartDate.Format("2006-01-02"),
		EndDate:   params.EndDate.Format("2006-01-02"),
		Interval:  params.Interval,
		Trends:    trends,
	})
}

// GetStatusTransitions returns status transition counts by geography
// GET /api/v1/history/trends/status?geo_type=city&geo_id=123&start_date=2024-01-01&end_date=2024-12-31&interval=month
func (h *Handler) GetStatusTransitions(w http.ResponseWriter, r *http.Request) {
	params, err := parseGeoTrendParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	transitions, err := repository.GetStatusTransitionsByGeo(r.Context(), h.Pool, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch status transitions")
		return
	}

	if transitions == nil {
		transitions = []repository.StatusTransitionPoint{}
	}

	writeJSON(w, http.StatusOK, statusTransitionsResponse{
		GeoType:     params.GeoType,
		GeoID:       params.GeoID,
		StartDate:   params.StartDate.Format("2006-01-02"),
		EndDate:     params.EndDate.Format("2006-01-02"),
		Interval:    params.Interval,
		Transitions: transitions,
	})
}

// GetNewListingCounts returns new listing counts by geography
// GET /api/v1/history/trends/listings?geo_type=city&geo_id=123&start_date=2024-01-01&end_date=2024-12-31&interval=month
func (h *Handler) GetNewListingCounts(w http.ResponseWriter, r *http.Request) {
	params, err := parseGeoTrendParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	counts, err := repository.GetListingCountsByGeo(r.Context(), h.Pool, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch listing counts")
		return
	}

	if counts == nil {
		counts = []repository.ListingCountPoint{}
	}

	writeJSON(w, http.StatusOK, listingCountsResponse{
		GeoType:   params.GeoType,
		GeoID:     params.GeoID,
		StartDate: params.StartDate.Format("2006-01-02"),
		EndDate:   params.EndDate.Format("2006-01-02"),
		Interval:  params.Interval,
		Counts:    counts,
	})
}

// GetDaysOnMarketTrends returns days on market trends by geography
// GET /api/v1/history/trends/dom?geo_type=city&geo_id=123&start_date=2024-01-01&end_date=2024-12-31&interval=month
func (h *Handler) GetDaysOnMarketTrends(w http.ResponseWriter, r *http.Request) {
	params, err := parseGeoTrendParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	trends, err := repository.GetDaysOnMarketTrendsByGeo(r.Context(), h.Pool, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch DOM trends")
		return
	}

	if trends == nil {
		trends = []repository.DOMTrendPoint{}
	}

	writeJSON(w, http.StatusOK, domTrendsResponse{
		GeoType:   params.GeoType,
		GeoID:     params.GeoID,
		StartDate: params.StartDate.Format("2006-01-02"),
		EndDate:   params.EndDate.Format("2006-01-02"),
		Interval:  params.Interval,
		Trends:    trends,
	})
}

// GetHistoryStats returns summary statistics for history tables
// GET /api/v1/history/stats
func (h *Handler) GetHistoryStats(w http.ResponseWriter, r *http.Request) {
	stats, err := repository.GetHistoryStatsSummary(r.Context(), h.Pool)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch history stats")
		return
	}

	writeJSON(w, http.StatusOK, historyStatsResponse{
		Stats: stats,
	})
}

// GetPriceChangeSummary returns price change summary statistics by geography
// GET /api/v1/history/summary/prices?geo_type=city&geo_id=123&start_date=2024-01-01&end_date=2024-12-31
func (h *Handler) GetPriceChangeSummary(w http.ResponseWriter, r *http.Request) {
	params, err := parseGeoTrendParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	summary, err := repository.GetPriceChangeSummaryByGeo(r.Context(), h.Pool, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch price change summary")
		return
	}

	writeJSON(w, http.StatusOK, priceChangeSummaryResponse{
		GeoType:   params.GeoType,
		GeoID:     params.GeoID,
		StartDate: params.StartDate.Format("2006-01-02"),
		EndDate:   params.EndDate.Format("2006-01-02"),
		Summary:   summary,
	})
}

func parseMarketSnapshotParams(r *http.Request) (repository.MarketSnapshotParams, error) {
	params := repository.MarketSnapshotParams{}

	// Required: geo_type
	params.GeoType = r.URL.Query().Get("geo_type")
	if params.GeoType == "" {
		return params, &paramError{"geo_type is required"}
	}
	validGeoTypes := map[string]bool{
		"state": true, "county": true, "city": true,
		"subdivision": true, "postal_code": true,
	}
	if !validGeoTypes[params.GeoType] {
		return params, &paramError{"geo_type must be one of: state, county, city, subdivision, postal_code"}
	}

	// Required: geo_id
	geoIDStr := r.URL.Query().Get("geo_id")
	if geoIDStr == "" {
		return params, &paramError{"geo_id is required"}
	}
	geoID, err := strconv.ParseInt(geoIDStr, 10, 64)
	if err != nil {
		return params, &paramError{"geo_id must be a valid integer"}
	}
	params.GeoID = geoID

	// 30-day window for new listings count
	params.NewListedSince = time.Now().AddDate(0, 0, -30)

	// Closed sales lookback window (default 90 days, max 365)
	closedDays := 90
	if cd := r.URL.Query().Get("closed_days"); cd != "" {
		d, err := strconv.Atoi(cd)
		if err != nil || d < 1 || d > 365 {
			return params, &paramError{"closed_days must be between 1 and 365"}
		}
		closedDays = d
	}
	params.ClosedSince = time.Now().AddDate(0, 0, -closedDays)

	// Optional: property_type
	if pt := r.URL.Query().Get("property_type"); pt != "" {
		params.PropertyType = &pt
	}

	// Optional: property_sub_type
	if pst := r.URL.Query().Get("property_sub_type"); pst != "" {
		params.PropertySubType = &pst
	}

	return params, nil
}

// GetMarketSnapshot returns a point-in-time market summary
// GET /api/v1/market/snapshot?geo_type=city&geo_id=730&property_type=Residential&closed_days=90
func (h *Handler) GetMarketSnapshot(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	params, err := parseMarketSnapshotParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var snapshot repository.MarketSnapshot
	var closedSnap repository.ClosedSalesSnapshot
	var pendingCount int

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		s, err := repository.GetMarketSnapshot(gCtx, h.Pool, params)
		if err != nil {
			return err
		}
		snapshot = s
		return nil
	})

	g.Go(func() error {
		s, err := repository.GetClosedSalesSnapshot(gCtx, h.Pool, params)
		if err != nil {
			return err
		}
		closedSnap = s
		return nil
	})

	g.Go(func() error {
		c, err := repository.GetPendingCount(gCtx, h.Pool, params)
		if err != nil {
			return err
		}
		pendingCount = c
		return nil
	})

	if err := g.Wait(); err != nil {
		log.Printf("market snapshot: query failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch market snapshot")
		return
	}

	snapshot.PendingCount = pendingCount
	snapshot.ClosedCount = closedSnap.ClosedCount
	snapshot.MedianClosePrice = closedSnap.MedianClosePrice
	snapshot.AvgClosePrice = closedSnap.AvgClosePrice
	snapshot.AvgDaysToClose = closedSnap.AvgDaysToClose

	h.trackHistoryEvent(ctx, analytics.HistoryTypeSnapshot, params.GeoType, params.GeoID, time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, marketSnapshotResponse{
		GeoType: params.GeoType,
		GeoID:   params.GeoID,
		Filters: marketSnapshotFilters{
			PropertyType:    params.PropertyType,
			PropertySubType: params.PropertySubType,
		},
		Snapshot: snapshot,
	})
}
