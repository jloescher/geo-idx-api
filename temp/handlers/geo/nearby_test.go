package geo

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/testutil"
)

// --- Unit tests (no DB) ---

func TestGetNearby_MissingType(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Get("/nearby", h.GetNearby)

	req := httptest.NewRequest(http.MethodGet, "/nearby?ref_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetNearby_InvalidType(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Get("/nearby", h.GetNearby)

	req := httptest.NewRequest(http.MethodGet, "/nearby?type=invalid&ref_id=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetNearby_MissingRefID(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Get("/nearby", h.GetNearby)

	req := httptest.NewRequest(http.MethodGet, "/nearby?type=city", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetNearby_InvalidRefID(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Get("/nearby", h.GetNearby)

	req := httptest.NewRequest(http.MethodGet, "/nearby?type=city&ref_id=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPostNearby_InvalidBody(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Post("/nearby", h.PostNearby)

	req := httptest.NewRequest(http.MethodPost, "/nearby", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPostNearby_MissingType(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Post("/nearby", h.PostNearby)

	body, _ := json.Marshal(nearbyPostRequest{RefID: 1})
	req := httptest.NewRequest(http.MethodPost, "/nearby", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPostNearby_InvalidRefID(t *testing.T) {
	h := &Handler{}
	r := chi.NewRouter()
	r.Post("/nearby", h.PostNearby)

	body, _ := json.Marshal(nearbyPostRequest{Type: "city", RefID: -1})
	req := httptest.NewRequest(http.MethodPost, "/nearby", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- Integration tests (require local PostgreSQL) ---

type nearbyFixture struct {
	StateID  int64
	CountyID int64
	CityAID  int64 // city with active properties
	CityBID  int64 // city with no properties
	PropIDs  []int64
}

func seedNearbyFixture(t *testing.T, pool *pgxpool.Pool) nearbyFixture {
	t.Helper()
	ctx := context.Background()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)

	var fx nearbyFixture

	err := pool.QueryRow(ctx, `
		INSERT INTO states (name, slug, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW()) RETURNING id
	`, "NearbyState"+suffix, "nearby-state-"+suffix).Scan(&fx.StateID)
	if err != nil {
		t.Fatalf("insert state: %v", err)
	}

	// County with a location point
	err = pool.QueryRow(ctx, `
		INSERT INTO counties (state_id, name, slug, location, created_at, updated_at)
		VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint(-82.45, 27.95), 4326)::geography, NOW(), NOW()) RETURNING id
	`, fx.StateID, "NearbyCounty"+suffix, "nearby-county-"+suffix).Scan(&fx.CountyID)
	if err != nil {
		t.Fatalf("insert county: %v", err)
	}

	// City A: has a location, will get active properties
	err = pool.QueryRow(ctx, `
		INSERT INTO cities (state_id, county_id, name, normalized_name, slug, is_authoritative, location, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, ST_SetSRID(ST_MakePoint(-82.46, 27.94), 4326)::geography, NOW(), NOW()) RETURNING id
	`, fx.StateID, fx.CountyID, "NearbyAlpha"+suffix, "nearbyalpha"+suffix, "nearby-alpha-"+suffix).Scan(&fx.CityAID)
	if err != nil {
		t.Fatalf("insert city A: %v", err)
	}

	// City B: nearby but no properties, offset ~2km north
	err = pool.QueryRow(ctx, `
		INSERT INTO cities (state_id, county_id, name, normalized_name, slug, is_authoritative, location, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, ST_SetSRID(ST_MakePoint(-82.44, 27.96), 4326)::geography, NOW(), NOW()) RETURNING id
	`, fx.StateID, fx.CountyID, "NearbyBeta"+suffix, "nearbybeta"+suffix, "nearby-beta-"+suffix).Scan(&fx.CityBID)
	if err != nil {
		t.Fatalf("insert city B: %v", err)
	}

	// Insert 2 active Residential properties in City A
	for i := 0; i < 2; i++ {
		listingID := "NEARBY-" + suffix + "-" + strconv.Itoa(i)
		var propID int64
		err = pool.QueryRow(ctx, `
			INSERT INTO properties_core (listing_id, city_ref_id, county_ref_id, state_ref_id, standard_status, property_type, partition_group)
			VALUES ($1, $2, $3, $4, 'Active', 'Residential', 'active') RETURNING id
		`, listingID, fx.CityAID, fx.CountyID, fx.StateID).Scan(&propID)
		if err != nil {
			t.Fatalf("insert property %d: %v", i, err)
		}
		fx.PropIDs = append(fx.PropIDs, propID)
	}

	t.Cleanup(func() {
		for _, pid := range fx.PropIDs {
			_, _ = pool.Exec(ctx, "DELETE FROM properties_core WHERE id = $1 AND partition_group = 'active'", pid)
		}
		_, _ = pool.Exec(ctx, "DELETE FROM cities WHERE id IN ($1, $2)", fx.CityAID, fx.CityBID)
		_, _ = pool.Exec(ctx, "DELETE FROM counties WHERE id = $1", fx.CountyID)
		_, _ = pool.Exec(ctx, "DELETE FROM states WHERE id = $1", fx.StateID)
	})

	return fx
}

func decodeNearbyResponse(t *testing.T, w *httptest.ResponseRecorder) nearbyResponse {
	t.Helper()
	var resp nearbyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func TestGetNearby_Cities(t *testing.T) {
	pool := testutil.NewPool(t)
	fx := seedNearbyFixture(t, pool)
	handler := NewHandler(pool, db.NewRegistry())

	r := chi.NewRouter()
	r.Get("/nearby", handler.GetNearby)

	// Search near City A — should find City B
	url := "/nearby?type=city&ref_id=" + strconv.FormatInt(fx.CityAID, 10) + "&radius_miles=50"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeNearbyResponse(t, w)
	if resp.Type != "city" {
		t.Fatalf("expected type=city, got %s", resp.Type)
	}

	// Find City B in results
	var foundB bool
	for _, item := range resp.Results {
		if item.RefID == fx.CityBID {
			foundB = true
			if item.ListingCount != 0 {
				t.Errorf("CityB listing_count: expected 0, got %d", item.ListingCount)
			}
		}
	}
	if !foundB {
		t.Fatalf("expected City B (id=%d) in results, got %v", fx.CityBID, resp.Results)
	}
}

func TestGetNearby_MinListingCount(t *testing.T) {
	pool := testutil.NewPool(t)
	fx := seedNearbyFixture(t, pool)
	handler := NewHandler(pool, db.NewRegistry())

	r := chi.NewRouter()
	r.Get("/nearby", handler.GetNearby)

	// Search near City A with min_listing_count=1 — City B (0 listings) should be filtered out
	url := "/nearby?type=city&ref_id=" + strconv.FormatInt(fx.CityAID, 10) + "&radius_miles=50&min_listing_count=1"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeNearbyResponse(t, w)
	for _, item := range resp.Results {
		if item.RefID == fx.CityBID {
			t.Fatalf("City B (0 listings) should have been filtered out by min_listing_count=1")
		}
	}
}

func TestPostNearby_WithSearchFilters(t *testing.T) {
	pool := testutil.NewPool(t)
	fx := seedNearbyFixture(t, pool)
	handler := NewHandler(pool, db.NewRegistry())

	r := chi.NewRouter()
	r.Post("/nearby", handler.PostNearby)

	// Search near City A with filters that match our Residential properties
	body, _ := json.Marshal(nearbyPostRequest{
		Type:        "city",
		RefID:       fx.CityAID,
		RadiusMiles: 50,
		SearchFilters: &nearbyFilters{
			ListingTypes: []string{"Residential"},
			Statuses:     []string{"Active"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/nearby", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeNearbyResponse(t, w)
	for _, item := range resp.Results {
		if item.RefID == fx.CityBID && item.ListingCount != 0 {
			t.Errorf("CityB filtered listing_count: expected 0, got %d", item.ListingCount)
		}
	}
}

func TestPostNearby_NoFilters(t *testing.T) {
	pool := testutil.NewPool(t)
	fx := seedNearbyFixture(t, pool)
	handler := NewHandler(pool, db.NewRegistry())

	r := chi.NewRouter()
	r.Post("/nearby", handler.PostNearby)

	body, _ := json.Marshal(nearbyPostRequest{
		Type:        "city",
		RefID:       fx.CityAID,
		RadiusMiles: 50,
	})
	req := httptest.NewRequest(http.MethodPost, "/nearby", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeNearbyResponse(t, w)
	if resp.Type != "city" {
		t.Fatalf("expected type=city, got %s", resp.Type)
	}
}
