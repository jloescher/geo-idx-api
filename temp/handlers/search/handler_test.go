package search

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	searchsvc "github.com/xotec-solutions/xotec-datalayer/src/internal/search"
)

func TestSearchPayload_ValidJSON(t *testing.T) {
	// Test that valid JSON payloads are parsed correctly
	payload := map[string]any{
		"params": map[string]any{
			"min_price": 200000,
			"max_price": 500000,
			"min_beds":  3,
		},
		"location_filters": map[string]any{
			"city_ref_ids": []int{101, 102},
		},
		"listing_types": []string{"Residential"},
		"page": map[string]any{
			"limit": 24,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req, err := searchsvc.ParseSearchPayload(body)
	if err != nil {
		t.Errorf("expected valid payload to parse, got error: %v", err)
	}

	if req.Params.MinPrice.Value == nil || *req.Params.MinPrice.Value != 200000 {
		t.Errorf("expected min_price 200000, got %v", req.Params.MinPrice.Value)
	}

	if req.Page.Limit.Value == nil || *req.Page.Limit.Value != 24 {
		t.Errorf("expected limit 24, got %v", req.Page.Limit.Value)
	}
}

func TestHandleSearch_InvalidJSON(t *testing.T) {
	h := &Handler{Store: nil}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid JSON, got %d", w.Code)
	}

	var resp errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	if resp.Error.Code != "validation_error" {
		t.Errorf("expected code validation_error, got %s", resp.Error.Code)
	}
}

func TestHandleSearch_UnknownField(t *testing.T) {
	h := &Handler{Store: nil}

	payload := `{"unknown_field": "value"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for unknown field, got %d", w.Code)
	}
}

func TestHandleSearch_PayloadTooLarge(t *testing.T) {
	h := &Handler{Store: nil}

	// Create payload larger than 2MB limit
	largePayload := make([]byte, 3<<20) // 3MB
	for i := range largePayload {
		largePayload[i] = 'x'
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search", bytes.NewReader(largePayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.HandleSearch(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413 for large payload, got %d", w.Code)
	}
}

func TestHandleListing_RequiresListingID(t *testing.T) {
	h := &Handler{Store: nil}

	r := chi.NewRouter()
	r.Post("/listings", h.HandleListing)

	req := httptest.NewRequest(http.MethodPost, "/listings", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing listing_id, got %d", w.Code)
	}
}

func TestHandleOpenAPI_ReturnsJSON(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	w := httptest.NewRecorder()

	h.HandleOpenAPI(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	// Verify it's valid JSON
	var openapi map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &openapi); err != nil {
		t.Errorf("response is not valid JSON: %v", err)
	}

	if openapi["openapi"] == nil {
		t.Error("expected openapi field in response")
	}
}

func TestHandleDocs_ReturnsHTML(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/docs", nil)
	w := httptest.NewRecorder()

	h.HandleDocs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type text/html, got %s", contentType)
	}
}

func TestSearchRequestJSON_Examples(t *testing.T) {
	// Test examples from OpenAPI spec
	examples := []struct {
		name    string
		payload string
	}{
		{
			name: "first_page",
			payload: `{
				"params": {
					"min_price": 350000,
					"max_price": 750000,
					"min_beds": 3,
					"min_baths": 2,
					"sort": "on_market_date",
					"sort_dir": "desc"
				},
				"location_filters": {
					"city_ref_ids": [101]
				},
				"listing_types": ["Residential"],
				"page": {
					"limit": 24
				}
			}`,
		},
		{
			name: "geo_distance",
			payload: `{
				"params": {
					"geo": {
						"distance": {
							"lat": 27.9506,
							"lng": -82.4572,
							"radius_miles": 10
						}
					}
				},
				"page": {"limit": 50}
			}`,
		},
		{
			name: "geo_bbox",
			payload: `{
				"params": {
					"geo": {
						"bbox": {
							"west": -82.6,
							"south": 27.8,
							"east": -82.3,
							"north": 28.1
						}
					}
				}
			}`,
		},
	}

	for _, tt := range examples {
		t.Run(tt.name, func(t *testing.T) {
			req, err := searchsvc.ParseSearchPayload([]byte(tt.payload))
			if err != nil {
				t.Errorf("failed to parse %s: %v", tt.name, err)
				return
			}

			// Basic sanity check
			if tt.name == "geo_distance" && req.Params.Geo == nil {
				t.Error("expected geo filters to be parsed")
			}
		})
	}
}
