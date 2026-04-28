package geo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/normalize"
	"github.com/xotec-solutions/xotec-datalayer/src/internal/testutil"
)

func TestGetCountyBoundary_InvalidID(t *testing.T) {
	h := &Handler{}

	r := chi.NewRouter()
	r.Get("/boundaries/counties/{id}", h.GetCountyBoundary)

	req := httptest.NewRequest(http.MethodGet, "/boundaries/counties/invalid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetCityBoundary_InvalidID(t *testing.T) {
	h := &Handler{}

	r := chi.NewRouter()
	r.Get("/boundaries/cities/{id}", h.GetCityBoundary)

	req := httptest.NewRequest(http.MethodGet, "/boundaries/cities/abc", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetPostalCodeBoundary_InvalidCode(t *testing.T) {
	h := &Handler{}

	r := chi.NewRouter()
	r.Get("/boundaries/postal-codes/{code}", h.GetPostalCodeBoundary)

	tests := []struct {
		name string
		code string
	}{
		{"empty", ""},
		{"too short", "123"},
		{"too long", "123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/boundaries/postal-codes/"+tt.code, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
				t.Errorf("expected status 400 or 404, got %d for code %q", w.Code, tt.code)
			}
		})
	}
}

func TestGeoJSONFeatureCollection_Structure(t *testing.T) {
	fc := GeoJSONFeatureCollection{
		Type: "FeatureCollection",
		Features: []GeoJSONFeature{
			{
				Type: "Feature",
				Properties: map[string]any{
					"id":   int64(1),
					"name": "Test County",
					"type": "county",
				},
				Geometry: json.RawMessage(`{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,1],[0,0]]]}`),
			},
		},
	}

	data, err := json.Marshal(fc)
	if err != nil {
		t.Fatalf("failed to marshal FeatureCollection: %v", err)
	}

	// Verify structure
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if result["type"] != "FeatureCollection" {
		t.Errorf("expected type FeatureCollection, got %v", result["type"])
	}

	features, ok := result["features"].([]any)
	if !ok || len(features) != 1 {
		t.Errorf("expected 1 feature, got %v", result["features"])
	}
}

func TestBoundaryResponse_Structure(t *testing.T) {
	resp := BoundaryResponse{
		ID:         123,
		Name:       "Hillsborough",
		Type:       "county",
		StateAbbr:  "FL",
		ParentName: "",
		GeoJSON:    json.RawMessage(`{"type":"MultiPolygon","coordinates":[[[[0,0],[1,0],[1,1],[0,0]]]]}`),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal BoundaryResponse: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if result["type"] != "county" {
		t.Errorf("expected type county, got %v", result["type"])
	}
	if result["name"] != "Hillsborough" {
		t.Errorf("expected name Hillsborough, got %v", result["name"])
	}
	if result["state_abbr"] != "FL" {
		t.Errorf("expected state_abbr FL, got %v", result["state_abbr"])
	}
}

// =====================================================
// GetStateWithCounties Tests
// =====================================================

// stateCountiesFixture holds test data for state with counties tests
type stateCountiesFixture struct {
	StateID  int64
	CountyID int64
}

// seedStateCountiesFixture creates test data for GetStateWithCounties tests
func seedStateCountiesFixture(t *testing.T, pool *pgxpool.Pool) stateCountiesFixture {
	t.Helper()
	ctx := context.Background()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)

	var fx stateCountiesFixture

	// Create a state
	err := pool.QueryRow(ctx, `
		INSERT INTO states (name, abbreviation, slug, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id
	`, "TestStateBoundary "+suffix, "TB", "test-state-boundary-"+suffix).Scan(&fx.StateID)
	require.NoError(t, err, "failed to insert state")

	// Create a county (won't have geom, so won't be returned in GeoJSON)
	err = pool.QueryRow(ctx, `
		INSERT INTO counties (state_id, name, normalized_name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`, fx.StateID, "TestCountyBoundary "+suffix, normalize.Name("TestCountyBoundary "+suffix), "test-county-boundary-"+suffix).Scan(&fx.CountyID)
	require.NoError(t, err, "failed to insert county")

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM counties WHERE id = $1", fx.CountyID)
		_, _ = pool.Exec(ctx, "DELETE FROM states WHERE id = $1", fx.StateID)
	})

	return fx
}

func newStateBoundaryTestHandler(t *testing.T) (*Handler, stateCountiesFixture) {
	t.Helper()
	pool := testutil.NewPool(t)
	fx := seedStateCountiesFixture(t, pool)
	return NewHandler(pool, db.NewRegistry()), fx
}

// TestGetStateWithCounties_InvalidID_NoDatabase tests invalid ID handling without DB
// Note: Only non-numeric IDs can be rejected without database access.
// Negative and zero IDs are valid integers that require database queries.
func TestGetStateWithCounties_InvalidID_NoDatabase(t *testing.T) {
	handler := &Handler{} // No pool needed for ID validation

	r := chi.NewRouter()
	r.Get("/boundaries/states/{id}/counties", handler.GetStateWithCounties)

	tests := []struct {
		name string
		id   string
	}{
		{"non-numeric", "invalid"},
		{"empty", ""},
		{"special-chars", "abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/boundaries/states/"+tt.id+"/counties", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid state ID")
		})
	}
}

func TestGetStateWithCounties_InvalidID_Returns400(t *testing.T) {
	handler, _ := newStateBoundaryTestHandler(t)

	r := chi.NewRouter()
	r.Get("/boundaries/states/{id}/counties", handler.GetStateWithCounties)

	tests := []struct {
		name string
		id   string
	}{
		{"non-numeric", "invalid"},
		{"negative", "-1"},
		{"zero", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/boundaries/states/"+tt.id+"/counties", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid state ID")
		})
	}
}

func TestGetStateWithCounties_ValidIDNoBoundaries_Returns404(t *testing.T) {
	handler, fx := newStateBoundaryTestHandler(t)

	r := chi.NewRouter()
	r.Get("/boundaries/states/{id}/counties", handler.GetStateWithCounties)

	// State exists but has no TIGER boundary geometry, so should return 404
	req := httptest.NewRequest(http.MethodGet, "/boundaries/states/"+strconv.FormatInt(fx.StateID, 10)+"/counties", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Returns 404 because state has no TIGER boundary data (geom IS NULL)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "State not found or has no boundaries")
}

func TestGetStateWithCounties_NonExistentState_Returns404(t *testing.T) {
	handler, _ := newStateBoundaryTestHandler(t)

	r := chi.NewRouter()
	r.Get("/boundaries/states/{id}/counties", handler.GetStateWithCounties)

	// Use a very high ID that doesn't exist
	req := httptest.NewRequest(http.MethodGet, "/boundaries/states/99999999/counties", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
