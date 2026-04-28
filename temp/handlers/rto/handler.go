package rto

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/analytics"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/mls"
)

const maxRTOBodyBytes = 1 << 16 // 64 KB

// Handler handles RTO estimate API requests.
type Handler struct {
	Pool      *pgxpool.Pool
	Registry  *db.Registry
	Analytics *analytics.Client
}

// NewHandler creates a new RTO handler.
func NewHandler(pool *pgxpool.Pool, registry *db.Registry) *Handler {
	return &Handler{Pool: pool, Registry: registry}
}

// WithAnalytics sets the analytics client for the handler.
func (h *Handler) WithAnalytics(client *analytics.Client) *Handler {
	h.Analytics = client
	return h
}

// HandleEstimate handles POST /api/v1/rto/estimate.
func (h *Handler) HandleEstimate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body := http.MaxBytesReader(w, r.Body, maxRTOBodyBytes)
	defer body.Close()

	payload, err := io.ReadAll(body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "Request body too large"})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	var req EstimateRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON: " + err.Error()})
		return
	}

	// Resolve list price: from request body or DB lookup.
	var listPrice float64
	var taxRatePct *float64
	listingID := strings.TrimSpace(req.ListingID)

	if req.ListPrice != nil && *req.ListPrice > 0 {
		listPrice = *req.ListPrice
	} else if listingID != "" {
		// DB lookup
		dbListingID := mls.AddPrefix(listingID)
		var dbListPrice *float64
		var dbTaxAnnual *float64
		sql := h.Registry.SQL(db.QueryName("RTOResolveListing"))
		err := h.Pool.QueryRow(ctx, sql, dbListingID).Scan(&dbListPrice, &dbTaxAnnual)
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Listing not found"})
			return
		}
		if err != nil {
			log.Printf("rto: resolve listing failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to resolve listing"})
			return
		}
		if dbListPrice == nil || *dbListPrice <= 0 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Listing has no list price"})
			return
		}
		listPrice = *dbListPrice

		// Derive tax rate from annual taxes if available.
		if dbTaxAnnual != nil && *dbTaxAnnual > 0 && listPrice > 0 {
			rate := (*dbTaxAnnual / listPrice) * 100
			taxRatePct = &rate
		}
	} else {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Either listing_id or list_price is required"})
		return
	}

	ri := resolveInputs(req, listPrice, taxRatePct)
	models := computeAllModels(ri)

	writeJSON(w, http.StatusOK, EstimateResponse{
		ListingID:    listingID,
		ListPrice:    listPrice,
		ModelVersion: modelVersion,
		Models:       models,
		Disclaimer:   disclaimer,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
