package geo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
)

// =====================================================
// Invalid ID Tests (No Database Required)
// =====================================================

func TestListCountiesByStateOrdered_InvalidID_Returns400(t *testing.T) {
	handler := &Handler{}

	r := chi.NewRouter()
	r.Get("/seo/states/{id}/counties", handler.ListCountiesByStateOrdered)

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
			req := httptest.NewRequest(http.MethodGet, "/seo/states/"+tt.id+"/counties", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), tt.expect)
		})
	}
}

func TestListCitiesByCountyOrdered_InvalidID_Returns400(t *testing.T) {
	handler := &Handler{}

	r := chi.NewRouter()
	r.Get("/seo/counties/{id}/cities", handler.ListCitiesByCountyOrdered)

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
			req := httptest.NewRequest(http.MethodGet, "/seo/counties/"+tt.id+"/cities", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid county ID")
		})
	}
}

func TestListCitiesByStateOrdered_InvalidID_Returns400(t *testing.T) {
	handler := &Handler{}

	r := chi.NewRouter()
	r.Get("/seo/states/{id}/cities", handler.ListCitiesByStateOrdered)

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
			req := httptest.NewRequest(http.MethodGet, "/seo/states/"+tt.id+"/cities", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid state ID")
		})
	}
}

func TestListSubdivisionsByCountyOrdered_InvalidID_Returns400(t *testing.T) {
	handler := &Handler{}

	r := chi.NewRouter()
	r.Get("/seo/counties/{id}/subdivisions", handler.ListSubdivisionsByCountyOrdered)

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
			req := httptest.NewRequest(http.MethodGet, "/seo/counties/"+tt.id+"/subdivisions", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid county ID")
		})
	}
}

func TestListSubdivisionsByCityOrdered_InvalidID_Returns400(t *testing.T) {
	handler := &Handler{}

	r := chi.NewRouter()
	r.Get("/seo/cities/{id}/subdivisions", handler.ListSubdivisionsByCityOrdered)

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
			req := httptest.NewRequest(http.MethodGet, "/seo/cities/"+tt.id+"/subdivisions", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid city ID")
		})
	}
}

func TestListPostalCodesByStateOrdered_InvalidID_Returns400(t *testing.T) {
	handler := &Handler{}

	r := chi.NewRouter()
	r.Get("/seo/states/{id}/postal-codes", handler.ListPostalCodesByStateOrdered)

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
			req := httptest.NewRequest(http.MethodGet, "/seo/states/"+tt.id+"/postal-codes", nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), "Invalid state ID")
		})
	}
}

// =====================================================
// Nil Pool Tests
// =====================================================

func TestListCountiesByStateOrdered_NilPool_Returns500(t *testing.T) {
	handler := &Handler{
		Pool:     nil,
		Registry: db.NewRegistry(),
	}

	r := chi.NewRouter()
	r.Get("/seo/states/{id}/counties", handler.ListCountiesByStateOrdered)

	req := httptest.NewRequest(http.MethodGet, "/seo/states/1/counties", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Database not configured")
}

func TestListCitiesByCountyOrdered_NilPool_Returns500(t *testing.T) {
	handler := &Handler{
		Pool:     nil,
		Registry: db.NewRegistry(),
	}

	r := chi.NewRouter()
	r.Get("/seo/counties/{id}/cities", handler.ListCitiesByCountyOrdered)

	req := httptest.NewRequest(http.MethodGet, "/seo/counties/1/cities", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Database not configured")
}

func TestListCitiesByStateOrdered_NilPool_Returns500(t *testing.T) {
	handler := &Handler{
		Pool:     nil,
		Registry: db.NewRegistry(),
	}

	r := chi.NewRouter()
	r.Get("/seo/states/{id}/cities", handler.ListCitiesByStateOrdered)

	req := httptest.NewRequest(http.MethodGet, "/seo/states/1/cities", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Database not configured")
}

func TestListSubdivisionsByCountyOrdered_NilPool_Returns500(t *testing.T) {
	handler := &Handler{
		Pool:     nil,
		Registry: db.NewRegistry(),
	}

	r := chi.NewRouter()
	r.Get("/seo/counties/{id}/subdivisions", handler.ListSubdivisionsByCountyOrdered)

	req := httptest.NewRequest(http.MethodGet, "/seo/counties/1/subdivisions", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Database not configured")
}

func TestListSubdivisionsByCityOrdered_NilPool_Returns500(t *testing.T) {
	handler := &Handler{
		Pool:     nil,
		Registry: db.NewRegistry(),
	}

	r := chi.NewRouter()
	r.Get("/seo/cities/{id}/subdivisions", handler.ListSubdivisionsByCityOrdered)

	req := httptest.NewRequest(http.MethodGet, "/seo/cities/1/subdivisions", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Database not configured")
}

func TestListPostalCodesByStateOrdered_NilPool_Returns500(t *testing.T) {
	handler := &Handler{
		Pool:     nil,
		Registry: db.NewRegistry(),
	}

	r := chi.NewRouter()
	r.Get("/seo/states/{id}/postal-codes", handler.ListPostalCodesByStateOrdered)

	req := httptest.NewRequest(http.MethodGet, "/seo/states/1/postal-codes", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Database not configured")
}

// =====================================================
// Response Structure Tests
// =====================================================

func TestCountySEO_JSONStructure(t *testing.T) {
	county := CountySEO{
		ID:                  1,
		Name:                "Test County",
		NormalizedName:      "testcounty",
		Slug:                "test-county",
		StateID:             10,
		StateName:           "Test State",
		StateAbbr:           "TS",
		PropertyCount:       ptrInt64(100),
		ActivePropertyCount: ptrInt64(80),
		Latitude:            ptrString("27.9903"),
		Longitude:           ptrString("-82.3018"),
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
	assert.Equal(t, float64(100), result["property_count"])
	assert.Equal(t, float64(80), result["active_property_count"])
}

func TestCitySEO_JSONStructure(t *testing.T) {
	city := CitySEO{
		ID:                  1,
		Name:                "Test City",
		NormalizedName:      "testcity",
		Slug:                "test-city",
		StateID:             10,
		StateName:           "Test State",
		StateAbbr:           "TS",
		CountyID:            20,
		CountyName:          "Test County",
		PropertyCount:       ptrInt64(50),
		ActivePropertyCount: ptrInt64(40),
		Latitude:            ptrString("27.9903"),
		Longitude:           ptrString("-82.3018"),
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
	assert.Equal(t, float64(50), result["property_count"])
}

func TestSubdivisionSEO_JSONStructure(t *testing.T) {
	subdiv := SubdivisionSEO{
		ID:                  1,
		Name:                "Test Subdivision",
		NormalizedName:      "testsubdivision",
		Slug:                "test-subdivision",
		StateID:             10,
		StateName:           "Test State",
		StateAbbr:           "TS",
		CountyID:            ptrInt64(20),
		CountyName:          ptrString("Test County"),
		CityID:              ptrInt64(30),
		CityName:            ptrString("Test City"),
		PropertyCount:       ptrInt64(25),
		ActivePropertyCount: ptrInt64(20),
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
	assert.Equal(t, float64(25), result["property_count"])
}

func TestPostalCodeSEO_JSONStructure(t *testing.T) {
	pc := PostalCodeSEO{
		ID:                  1,
		Code:                "33601",
		Slug:                "33601",
		StateID:             ptrInt64(10),
		StateName:           ptrString("Test State"),
		StateAbbr:           ptrString("TS"),
		PropertyCount:       ptrInt64(100),
		ActivePropertyCount: ptrInt64(80),
		Latitude:            ptrString("27.9903"),
		Longitude:           ptrString("-82.3018"),
	}

	data, err := json.Marshal(pc)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(1), result["id"])
	assert.Equal(t, "33601", result["code"])
	assert.Equal(t, float64(10), result["state_id"])
	assert.Equal(t, float64(100), result["property_count"])
}

// =====================================================
// OmitEmpty Tests
// =====================================================

func TestCountySEO_OmitEmpty(t *testing.T) {
	county := CountySEO{
		ID:             1,
		Name:           "Test County",
		NormalizedName: "testcounty",
		Slug:           "test-county",
		StateID:        10,
		StateName:      "Test State",
		StateAbbr:      "TS",
	}

	data, err := json.Marshal(county)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(1), result["id"])
	_, hasPropertyCount := result["property_count"]
	assert.False(t, hasPropertyCount, "property_count should be omitted when nil")
	_, hasLatitude := result["latitude"]
	assert.False(t, hasLatitude, "latitude should be omitted when nil")
}

func TestCitySEO_OmitEmpty(t *testing.T) {
	city := CitySEO{
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

	_, hasPropertyCount := result["property_count"]
	assert.False(t, hasPropertyCount, "property_count should be omitted when nil")
}

func TestSubdivisionSEO_OmitEmpty(t *testing.T) {
	subdiv := SubdivisionSEO{
		ID:             1,
		Name:           "Test Subdivision",
		NormalizedName: "testsubdivision",
		Slug:           "test-subdivision",
		StateID:        10,
		StateName:      "Test State",
		StateAbbr:      "TS",
	}

	data, err := json.Marshal(subdiv)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	_, hasCountyID := result["county_id"]
	assert.False(t, hasCountyID, "county_id should be omitted when nil")
	_, hasCityID := result["city_id"]
	assert.False(t, hasCityID, "city_id should be omitted when nil")
	_, hasPropertyCount := result["property_count"]
	assert.False(t, hasPropertyCount, "property_count should be omitted when nil")
}

func TestPostalCodeSEO_OmitEmpty(t *testing.T) {
	pc := PostalCodeSEO{
		ID:   1,
		Code: "33601",
		Slug: "33601",
	}

	data, err := json.Marshal(pc)
	require.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	_, hasStateID := result["state_id"]
	assert.False(t, hasStateID, "state_id should be omitted when nil")
	_, hasPropertyCount := result["property_count"]
	assert.False(t, hasPropertyCount, "property_count should be omitted when nil")
}
