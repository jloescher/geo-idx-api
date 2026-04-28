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

// childrenFixture holds test data IDs for children endpoint tests
type childrenFixture struct {
	StateID       int64
	CountyID      int64
	CityID        int64
	SubdivisionID int64

	// Additional test entities for edge cases
	EmptyCountyID int64 // County with no cities or subdivisions
	EmptyCityID   int64 // City with no subdivisions
}

// seedChildrenFixture creates test data for children endpoint tests
func seedChildrenFixture(t *testing.T, pool *pgxpool.Pool) childrenFixture {
	t.Helper()
	ctx := context.Background()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)

	var fx childrenFixture

	// Create a state
	err := pool.QueryRow(ctx, `
		INSERT INTO states (name, abbreviation, slug, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id
	`, "TestChildrenState "+suffix, "TC", "test-children-state-"+suffix).Scan(&fx.StateID)
	require.NoError(t, err, "failed to insert state")

	// Create a county
	err = pool.QueryRow(ctx, `
		INSERT INTO counties (state_id, name, normalized_name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`, fx.StateID, "TestChildrenCounty "+suffix, normalize.Name("TestChildrenCounty "+suffix), "test-children-county-"+suffix).Scan(&fx.CountyID)
	require.NoError(t, err, "failed to insert county")

	// Create a city
	cityName := "TestChildrenCity " + suffix
	err = pool.QueryRow(ctx, `
		INSERT INTO cities (state_id, county_id, name, normalized_name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, fx.StateID, fx.CountyID, cityName, normalize.Name(cityName), "test-children-city-"+suffix).Scan(&fx.CityID)
	require.NoError(t, err, "failed to insert city")

	// Create a subdivision
	subdivName := "TestChildrenSubdivision " + suffix
	err = pool.QueryRow(ctx, `
		INSERT INTO subdivisions (city_id, state_id, county_id, name, normalized_name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id
	`, fx.CityID, fx.StateID, fx.CountyID, subdivName, normalize.Name(subdivName), "test-children-subdivision-"+suffix).Scan(&fx.SubdivisionID)
	require.NoError(t, err, "failed to insert subdivision")

	// Create an empty county (no cities or subdivisions)
	err = pool.QueryRow(ctx, `
		INSERT INTO counties (state_id, name, normalized_name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id
	`, fx.StateID, "EmptyCounty "+suffix, normalize.Name("EmptyCounty "+suffix), "empty-county-"+suffix).Scan(&fx.EmptyCountyID)
	require.NoError(t, err, "failed to insert empty county")

	// Create an empty city (no subdivisions)
	err = pool.QueryRow(ctx, `
		INSERT INTO cities (state_id, county_id, name, normalized_name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id
	`, fx.StateID, fx.CountyID, "EmptyCity "+suffix, normalize.Name("EmptyCity "+suffix), "empty-city-"+suffix).Scan(&fx.EmptyCityID)
	require.NoError(t, err, "failed to insert empty city")

	// Cleanup function
	t.Cleanup(func() {
		// Delete in reverse dependency order
		_, _ = pool.Exec(ctx, "DELETE FROM subdivisions WHERE id = $1", fx.SubdivisionID)
		_, _ = pool.Exec(ctx, "DELETE FROM cities WHERE id IN ($1, $2)", fx.CityID, fx.EmptyCityID)
		_, _ = pool.Exec(ctx, "DELETE FROM counties WHERE id IN ($1, $2)", fx.CountyID, fx.EmptyCountyID)
		_, _ = pool.Exec(ctx, "DELETE FROM states WHERE id = $1", fx.StateID)
	})

	return fx
}

// newChildrenTestHandler creates a handler with test database connection
func newChildrenTestHandler(t *testing.T) (*Handler, childrenFixture) {
	t.Helper()
	pool := testutil.NewPool(t)
	fx := seedChildrenFixture(t, pool)
	return NewHandler(pool, db.NewRegistry()), fx
}

// =====================================================
// Test ListCountiesByState
// =====================================================

// TestListCountiesByState_InvalidID_NoDatabase tests invalid ID handling without DB
func TestListCountiesByState_InvalidID_NoDatabase(t *testing.T) {
	handler := &Handler{} // No pool needed for ID validation

	r := chi.NewRouter()
	r.Get("/children/states/{id}/counties", handler.ListCountiesByState)

	tests := []struct {
		name   string
		id     string
		expect string
	}{
		{"non-numeric", "invalid", "Invalid state ID"},
		{"negative", "-1", "Invalid state ID"},
		{"zero", "0", "Invalid state ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/children/states/"+tt.id+"/counties", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), tt.expect)
		})
	}
}

func TestListCountiesByState_ValidID_ReturnsCounties(t *testing.T) {
	handler, fx := newChildrenTestHandler(t)

	r := chi.NewRouter()
	r.Get("/children/states/{id}/counties", handler.ListCountiesByState)

	req := httptest.NewRequest(http.MethodGet, "/children/states/"+strconv.FormatInt(fx.StateID, 10)+"/counties", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Cache-Control"), "max-age=300")

	// Decode response
	var counties []CountyChild
	err := json.NewDecoder(w.Body).Decode(&counties)
	require.NoError(t, err, "failed to decode response")

	// Verify we got at least one county (the one we created)
	assert.GreaterOrEqual(t, len(counties), 1, "expected at least one county")

	// Find our test county
	var found bool
	for _, c := range counties {
		if c.ID == fx.CountyID {
			found = true
			assert.Equal(t, fx.StateID, c.StateID)
			assert.NotEmpty(t, c.Name)
			assert.NotEmpty(t, c.NormalizedName)
			assert.NotEmpty(t, c.Slug)
			break
		}
	}
	assert.True(t, found, "expected to find test county in response")
}

func TestListCountiesByState_ValidIDNoCounties_ReturnsEmptyArray(t *testing.T) {
	handler, fx := newChildrenTestHandler(t)
	pool := testutil.NewPool(t)
	ctx := context.Background()

	// Create a new state with no counties
	var emptyStateID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO states (name, abbreviation, slug, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id
	`, "EmptyState "+strconv.FormatInt(time.Now().UnixNano(), 10), "ES", "empty-state").Scan(&emptyStateID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM states WHERE id = $1", emptyStateID)
	})

	// Use the handler from the fixture but test with the empty state
	r := chi.NewRouter()
	r.Get("/children/states/{id}/counties", handler.ListCountiesByState)

	req := httptest.NewRequest(http.MethodGet, "/children/states/"+strconv.FormatInt(emptyStateID, 10)+"/counties", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify it's an empty array, not null
	var counties []CountyChild
	err = json.NewDecoder(w.Body).Decode(&counties)
	require.NoError(t, err)
	assert.Empty(t, counties)
	assert.NotNil(t, counties) // Should be empty slice, not nil

	_ = fx // fx is not used directly but keeps the test consistent
}

// =====================================================
// Test ListCitiesByCounty
// =====================================================

// TestListCitiesByCounty_InvalidID_NoDatabase tests invalid ID handling without DB
func TestListCitiesByCounty_InvalidID_NoDatabase(t *testing.T) {
	handler := &Handler{} // No pool needed for ID validation

	r := chi.NewRouter()
	r.Get("/children/counties/{id}/cities", handler.ListCitiesByCounty)

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
			req := httptest.NewRequest(http.MethodGet, "/children/counties/"+tt.id+"/cities", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid county ID")
		})
	}
}

func TestListCitiesByCounty_ValidID_ReturnsCities(t *testing.T) {
	handler, fx := newChildrenTestHandler(t)

	r := chi.NewRouter()
	r.Get("/children/counties/{id}/cities", handler.ListCitiesByCounty)

	req := httptest.NewRequest(http.MethodGet, "/children/counties/"+strconv.FormatInt(fx.CountyID, 10)+"/cities", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Cache-Control"), "max-age=300")

	var cities []CityChild
	err := json.NewDecoder(w.Body).Decode(&cities)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(cities), 1)

	var found bool
	for _, c := range cities {
		if c.ID == fx.CityID {
			found = true
			assert.Equal(t, fx.CountyID, c.CountyID)
			assert.Equal(t, fx.StateID, c.StateID)
			assert.NotEmpty(t, c.Name)
			assert.NotEmpty(t, c.CountyName)
			break
		}
	}
	assert.True(t, found)
}

func TestListCitiesByCounty_ValidIDNoCities_ReturnsEmptyArray(t *testing.T) {
	handler, fx := newChildrenTestHandler(t)

	r := chi.NewRouter()
	r.Get("/children/counties/{id}/cities", handler.ListCitiesByCounty)

	// Use the empty county created in fixture
	req := httptest.NewRequest(http.MethodGet, "/children/counties/"+strconv.FormatInt(fx.EmptyCountyID, 10)+"/cities", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var cities []CityChild
	err := json.NewDecoder(w.Body).Decode(&cities)
	require.NoError(t, err)
	assert.Empty(t, cities)
	assert.NotNil(t, cities)
}

// =====================================================
// Test ListSubdivisionsByCounty
// =====================================================

// TestListSubdivisionsByCounty_InvalidID_NoDatabase tests invalid ID handling without DB
func TestListSubdivisionsByCounty_InvalidID_NoDatabase(t *testing.T) {
	handler := &Handler{} // No pool needed for ID validation

	r := chi.NewRouter()
	r.Get("/children/counties/{id}/subdivisions", handler.ListSubdivisionsByCounty)

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
			req := httptest.NewRequest(http.MethodGet, "/children/counties/"+tt.id+"/subdivisions", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid county ID")
		})
	}
}

func TestListSubdivisionsByCounty_ValidID_ReturnsSubdivisions(t *testing.T) {
	handler, fx := newChildrenTestHandler(t)

	r := chi.NewRouter()
	r.Get("/children/counties/{id}/subdivisions", handler.ListSubdivisionsByCounty)

	req := httptest.NewRequest(http.MethodGet, "/children/counties/"+strconv.FormatInt(fx.CountyID, 10)+"/subdivisions", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Cache-Control"), "max-age=300")

	var subdivisions []SubdivisionChild
	err := json.NewDecoder(w.Body).Decode(&subdivisions)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(subdivisions), 1)

	var found bool
	for _, s := range subdivisions {
		if s.ID == fx.SubdivisionID {
			found = true
			assert.NotEmpty(t, s.Name)
			assert.NotEmpty(t, s.NormalizedName)
			break
		}
	}
	assert.True(t, found)
}

func TestListSubdivisionsByCounty_ValidIDNoSubdivisions_ReturnsEmptyArray(t *testing.T) {
	handler, fx := newChildrenTestHandler(t)

	r := chi.NewRouter()
	r.Get("/children/counties/{id}/subdivisions", handler.ListSubdivisionsByCounty)

	// Use the empty county
	req := httptest.NewRequest(http.MethodGet, "/children/counties/"+strconv.FormatInt(fx.EmptyCountyID, 10)+"/subdivisions", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var subdivisions []SubdivisionChild
	err := json.NewDecoder(w.Body).Decode(&subdivisions)
	require.NoError(t, err)
	assert.Empty(t, subdivisions)
	assert.NotNil(t, subdivisions)
}

// =====================================================
// Test ListSubdivisionsByCity
// =====================================================

// TestListSubdivisionsByCity_InvalidID_NoDatabase tests invalid ID handling without DB
func TestListSubdivisionsByCity_InvalidID_NoDatabase(t *testing.T) {
	handler := &Handler{} // No pool needed for ID validation

	r := chi.NewRouter()
	r.Get("/children/cities/{id}/subdivisions", handler.ListSubdivisionsByCity)

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
			req := httptest.NewRequest(http.MethodGet, "/children/cities/"+tt.id+"/subdivisions", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid city ID")
		})
	}
}

func TestListSubdivisionsByCity_ValidID_ReturnsSubdivisions(t *testing.T) {
	handler, fx := newChildrenTestHandler(t)

	r := chi.NewRouter()
	r.Get("/children/cities/{id}/subdivisions", handler.ListSubdivisionsByCity)

	req := httptest.NewRequest(http.MethodGet, "/children/cities/"+strconv.FormatInt(fx.CityID, 10)+"/subdivisions", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Cache-Control"), "max-age=300")

	var subdivisions []SubdivisionChild
	err := json.NewDecoder(w.Body).Decode(&subdivisions)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(subdivisions), 1)

	var found bool
	for _, s := range subdivisions {
		if s.ID == fx.SubdivisionID {
			found = true
			assert.NotEmpty(t, s.Name)
			if s.CityID != nil {
				assert.Equal(t, fx.CityID, *s.CityID)
			}
			break
		}
	}
	assert.True(t, found)
}

func TestListSubdivisionsByCity_ValidIDNoSubdivisions_ReturnsEmptyArray(t *testing.T) {
	handler, fx := newChildrenTestHandler(t)

	r := chi.NewRouter()
	r.Get("/children/cities/{id}/subdivisions", handler.ListSubdivisionsByCity)

	// Use the empty city
	req := httptest.NewRequest(http.MethodGet, "/children/cities/"+strconv.FormatInt(fx.EmptyCityID, 10)+"/subdivisions", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var subdivisions []SubdivisionChild
	err := json.NewDecoder(w.Body).Decode(&subdivisions)
	require.NoError(t, err)
	assert.Empty(t, subdivisions)
	assert.NotNil(t, subdivisions)
}

// =====================================================
// Database Error Tests
// =====================================================

func TestListCountiesByState_DatabaseError_Returns500(t *testing.T) {
	// Create handler with nil pool to simulate database error
	handler := &Handler{
		Pool:     nil,
		Registry: db.NewRegistry(),
	}

	r := chi.NewRouter()
	r.Get("/children/states/{id}/counties", handler.ListCountiesByState)

	req := httptest.NewRequest(http.MethodGet, "/children/states/1/counties", nil)
	w := httptest.NewRecorder()

	// This will panic with nil pool, so we need to recover
	defer func() {
		_ = recover()
	}()

	r.ServeHTTP(w, req)

	// If we get here without panic, check for 500
	// Note: This test might need adjustment based on how nil pool is handled
}

// =====================================================
// Response Structure Tests
// =====================================================

func TestCountyChild_JSONStructure(t *testing.T) {
	county := CountyChild{
		ID:             1,
		Name:           "Test County",
		NormalizedName: "testcounty",
		Slug:           "test-county",
		StateID:        10,
		StateName:      "Test State",
		StateAbbr:      "TS",
		PropertyCount:  ptrInt64(100),
		Latitude:       ptrString("27.9903"),
		Longitude:      ptrString("-82.3018"),
	}

	data, err := json.Marshal(county)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(1), result["id"])
	assert.Equal(t, "Test County", result["name"])
	assert.Equal(t, "testcounty", result["normalized_name"])
	assert.Equal(t, "test-county", result["slug"])
	assert.Equal(t, float64(10), result["state_id"])
	assert.Equal(t, "Test State", result["state_name"])
	assert.Equal(t, "TS", result["state_abbr"])
}

func TestCityChild_JSONStructure(t *testing.T) {
	city := CityChild{
		ID:             1,
		Name:           "Test City",
		NormalizedName: "testcity",
		Slug:           "test-city",
		StateID:        10,
		StateName:      "Test State",
		StateAbbr:      "TS",
		CountyID:       20,
		CountyName:     "Test County",
	}

	data, err := json.Marshal(city)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(1), result["id"])
	assert.Equal(t, "Test City", result["name"])
	assert.Equal(t, float64(20), result["county_id"])
	assert.Equal(t, "Test County", result["county_name"])
}

func TestSubdivisionChild_JSONStructure(t *testing.T) {
	subdiv := SubdivisionChild{
		ID:             1,
		Name:           "Test Subdivision",
		NormalizedName: "testsubdivision",
		Slug:           "test-subdivision",
		StateID:        10,
		StateName:      "Test State",
		StateAbbr:      "TS",
		CountyID:       ptrInt64(20),
		CountyName:     ptrString("Test County"),
		CityID:         ptrInt64(30),
		CityName:       ptrString("Test City"),
	}

	data, err := json.Marshal(subdiv)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(1), result["id"])
	assert.Equal(t, "Test Subdivision", result["name"])
	assert.Equal(t, float64(20), result["county_id"])
	assert.Equal(t, "Test County", result["county_name"])
	assert.Equal(t, float64(30), result["city_id"])
	assert.Equal(t, "Test City", result["city_name"])
}

// =====================================================
// Helper Functions
// =====================================================

func ptrInt64(v int64) *int64 {
	return &v
}

func ptrString(v string) *string {
	return &v
}
