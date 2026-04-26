package search

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationError(t *testing.T) {
	t.Parallel()

	err := newValidationError()
	err.Fields["test"] = "error"

	assert.Equal(t, "invalid request", err.Error())
	assert.Contains(t, err.Fields, "test")
}

func TestIntParamUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    *int
		wantErr bool
	}{
		{"number", "42", ptr(42), false},
		{"string number", `"42"`, ptr(42), false},
		{"negative number", "-10", ptr(-10), false},
		{"negative string", `"-10"`, ptr(-10), false},
		{"null", "null", nil, false},
		{"empty string", `""`, nil, false},
		{"float whole", "42.0", ptr(42), false},
		{"float decimal", "42.5", nil, true},
		{"invalid string", `"abc"`, nil, true},
		{"invalid type", `{"key":"val"}`, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p IntParam
			err := json.Unmarshal([]byte(tt.input), &p)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, p.Value)
		})
	}
}

func TestIntParamMethods(t *testing.T) {
	t.Parallel()

	t.Run("IntOr with value", func(t *testing.T) {
		p := IntParam{Value: ptr(10)}
		assert.Equal(t, 10, p.IntOr(5))
	})

	t.Run("IntOr without value", func(t *testing.T) {
		p := IntParam{}
		assert.Equal(t, 5, p.IntOr(5))
	})

	t.Run("Int with value", func(t *testing.T) {
		p := IntParam{Value: ptr(10)}
		val, ok := p.Int()
		assert.True(t, ok)
		assert.Equal(t, 10, val)
	})

	t.Run("Int without value", func(t *testing.T) {
		p := IntParam{}
		val, ok := p.Int()
		assert.False(t, ok)
		assert.Equal(t, 0, val)
	})
}

func TestFloatParamUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    *float64
		wantErr bool
	}{
		{"number", "42.5", ptrFloat(42.5), false},
		{"integer", "42", ptrFloat(42), false},
		{"string number", `"42.5"`, ptrFloat(42.5), false},
		{"negative", "-10.5", ptrFloat(-10.5), false},
		{"null", "null", nil, false},
		{"empty string", `""`, nil, false},
		{"invalid string", `"abc"`, nil, true},
		{"invalid type", `{"key":"val"}`, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p FloatParam
			err := json.Unmarshal([]byte(tt.input), &p)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want == nil {
				assert.Nil(t, p.Value)
			} else {
				require.NotNil(t, p.Value)
				assert.InDelta(t, *tt.want, *p.Value, 0.001)
			}
		})
	}
}

func TestFloatParamMethods(t *testing.T) {
	t.Parallel()

	t.Run("Float64 with value", func(t *testing.T) {
		p := FloatParam{Value: ptrFloat(10.5)}
		val, ok := p.Float64()
		assert.True(t, ok)
		assert.InDelta(t, 10.5, val, 0.001)
	})

	t.Run("Float64 without value", func(t *testing.T) {
		p := FloatParam{}
		val, ok := p.Float64()
		assert.False(t, ok)
		assert.Equal(t, float64(0), val)
	})
}

func TestBoolParamUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    *bool
		wantErr bool
	}{
		{"true literal", "true", ptrBool(true), false},
		{"false literal", "false", ptrBool(false), false},
		{"string true", `"true"`, ptrBool(true), false},
		{"string false", `"false"`, ptrBool(false), false},
		{"string t", `"t"`, ptrBool(true), false},
		{"string f", `"f"`, ptrBool(false), false},
		{"string yes", `"yes"`, ptrBool(true), false},
		{"string no", `"no"`, ptrBool(false), false},
		{"string y", `"y"`, ptrBool(true), false},
		{"string n", `"n"`, ptrBool(false), false},
		{"string 1", `"1"`, ptrBool(true), false},
		{"string 0", `"0"`, ptrBool(false), false},
		{"number 1", "1", ptrBool(true), false},
		{"number 0", "0", ptrBool(false), false},
		{"number 5", "5", ptrBool(true), false},
		{"null", "null", nil, false},
		{"invalid string", `"maybe"`, nil, true},
		{"invalid type", `{"key":"val"}`, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p BoolParam
			err := json.Unmarshal([]byte(tt.input), &p)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, p.Value)
		})
	}
}

func TestBoolParamMethods(t *testing.T) {
	t.Parallel()

	t.Run("BoolOr with value", func(t *testing.T) {
		p := BoolParam{Value: ptrBool(true)}
		assert.True(t, p.BoolOr(false))
	})

	t.Run("BoolOr without value", func(t *testing.T) {
		p := BoolParam{}
		assert.True(t, p.BoolOr(true))
	})

	t.Run("Bool with value", func(t *testing.T) {
		p := BoolParam{Value: ptrBool(false)}
		val, ok := p.Bool()
		assert.True(t, ok)
		assert.False(t, val)
	})

	t.Run("Bool without value", func(t *testing.T) {
		p := BoolParam{}
		val, ok := p.Bool()
		assert.False(t, ok)
		assert.False(t, val)
	})
}

func TestIDListUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    []int64
		wantErr bool
	}{
		{"empty array", "[]", []int64{}, false},
		{"numbers", "[1, 2, 3]", []int64{1, 2, 3}, false},
		{"strings", `["1", "2", "3"]`, []int64{1, 2, 3}, false},
		{"mixed", `[1, "2", 3]`, []int64{1, 2, 3}, false},
		{"null", "null", nil, false},
		{"with empty strings", `["1", "", "3"]`, []int64{1, 3}, false},
		{"invalid string", `["abc"]`, nil, true},
		{"invalid type", `[{"id":1}]`, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var l IDList
			err := json.Unmarshal([]byte(tt.input), &l)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, l.Values)
		})
	}
}

func TestGeoFiltersDistance(t *testing.T) {
	t.Parallel()

	t.Run("HasDistance with valid filter", func(t *testing.T) {
		g := GeoFilters{
			Distance: &DistanceFilter{
				Lat:         FloatParam{Value: ptrFloat(27.0)},
				Lng:         FloatParam{Value: ptrFloat(-82.0)},
				RadiusMiles: FloatParam{Value: ptrFloat(5.0)},
			},
		}
		assert.True(t, g.HasDistance())
	})

	t.Run("HasDistance without filter", func(t *testing.T) {
		g := GeoFilters{}
		assert.False(t, g.HasDistance())
	})

	t.Run("HasDistance with incomplete filter", func(t *testing.T) {
		g := GeoFilters{
			Distance: &DistanceFilter{
				Lat: FloatParam{Value: ptrFloat(27.0)},
			},
		}
		assert.False(t, g.HasDistance())
	})

	t.Run("DistanceValues", func(t *testing.T) {
		g := GeoFilters{
			Distance: &DistanceFilter{
				Lat:         FloatParam{Value: ptrFloat(27.0)},
				Lng:         FloatParam{Value: ptrFloat(-82.0)},
				RadiusMiles: FloatParam{Value: ptrFloat(5.0)},
			},
		}
		lat, lng, rad, ok := g.DistanceValues()
		assert.True(t, ok)
		assert.InDelta(t, 27.0, lat, 0.001)
		assert.InDelta(t, -82.0, lng, 0.001)
		assert.InDelta(t, 5.0, rad, 0.001)
	})

	t.Run("DistanceCenter", func(t *testing.T) {
		g := GeoFilters{
			Distance: &DistanceFilter{
				Lat: FloatParam{Value: ptrFloat(27.0)},
				Lng: FloatParam{Value: ptrFloat(-82.0)},
			},
		}
		lat, lng, ok := g.DistanceCenter()
		assert.True(t, ok)
		assert.InDelta(t, 27.0, lat, 0.001)
		assert.InDelta(t, -82.0, lng, 0.001)
	})

	t.Run("DistanceCenter nil filter", func(t *testing.T) {
		g := GeoFilters{}
		lat, lng, ok := g.DistanceCenter()
		assert.False(t, ok)
		assert.Equal(t, float64(0), lat)
		assert.Equal(t, float64(0), lng)
	})

	t.Run("DistanceValues nil filter", func(t *testing.T) {
		g := GeoFilters{}
		lat, lng, rad, ok := g.DistanceValues()
		assert.False(t, ok)
		assert.Equal(t, float64(0), lat)
		assert.Equal(t, float64(0), lng)
		assert.Equal(t, float64(0), rad)
	})
}

func TestGeoFiltersBBox(t *testing.T) {
	t.Parallel()

	t.Run("BBoxValues valid", func(t *testing.T) {
		g := GeoFilters{
			BBox: &BBoxFilter{
				West:  FloatParam{Value: ptrFloat(-82.5)},
				South: FloatParam{Value: ptrFloat(27.0)},
				East:  FloatParam{Value: ptrFloat(-82.0)},
				North: FloatParam{Value: ptrFloat(28.0)},
			},
		}
		west, south, east, north, ok := g.BBoxValues()
		assert.True(t, ok)
		assert.InDelta(t, -82.5, west, 0.001)
		assert.InDelta(t, 27.0, south, 0.001)
		assert.InDelta(t, -82.0, east, 0.001)
		assert.InDelta(t, 28.0, north, 0.001)
	})

	t.Run("BBoxValues incomplete", func(t *testing.T) {
		g := GeoFilters{
			BBox: &BBoxFilter{
				West: FloatParam{Value: ptrFloat(-82.5)},
			},
		}
		_, _, _, _, ok := g.BBoxValues()
		assert.False(t, ok)
	})

	t.Run("BBoxValues nil", func(t *testing.T) {
		g := GeoFilters{}
		_, _, _, _, ok := g.BBoxValues()
		assert.False(t, ok)
	})
}

func TestGeoFiltersPolygon(t *testing.T) {
	t.Parallel()

	t.Run("PolygonWKT valid", func(t *testing.T) {
		g := GeoFilters{
			Polygon: &PolygonFilter{
				Points: []GeoPoint{
					{Lat: FloatParam{Value: ptrFloat(27.0)}, Lng: FloatParam{Value: ptrFloat(-82.0)}},
					{Lat: FloatParam{Value: ptrFloat(27.5)}, Lng: FloatParam{Value: ptrFloat(-82.0)}},
					{Lat: FloatParam{Value: ptrFloat(27.5)}, Lng: FloatParam{Value: ptrFloat(-82.5)}},
				},
			},
		}
		wkt, ok := g.PolygonWKT()
		assert.True(t, ok)
		assert.Contains(t, wkt, "POLYGON((")
	})

	t.Run("PolygonWKT closed ring", func(t *testing.T) {
		g := GeoFilters{
			Polygon: &PolygonFilter{
				Points: []GeoPoint{
					{Lat: FloatParam{Value: ptrFloat(27.0)}, Lng: FloatParam{Value: ptrFloat(-82.0)}},
					{Lat: FloatParam{Value: ptrFloat(27.5)}, Lng: FloatParam{Value: ptrFloat(-82.0)}},
					{Lat: FloatParam{Value: ptrFloat(27.5)}, Lng: FloatParam{Value: ptrFloat(-82.5)}},
					{Lat: FloatParam{Value: ptrFloat(27.0)}, Lng: FloatParam{Value: ptrFloat(-82.0)}},
				},
			},
		}
		wkt, ok := g.PolygonWKT()
		assert.True(t, ok)
		assert.Contains(t, wkt, "POLYGON((")
	})

	t.Run("PolygonWKT too few points", func(t *testing.T) {
		g := GeoFilters{
			Polygon: &PolygonFilter{
				Points: []GeoPoint{
					{Lat: FloatParam{Value: ptrFloat(27.0)}, Lng: FloatParam{Value: ptrFloat(-82.0)}},
				},
			},
		}
		_, ok := g.PolygonWKT()
		assert.False(t, ok)
	})

	t.Run("PolygonWKT nil", func(t *testing.T) {
		g := GeoFilters{}
		_, ok := g.PolygonWKT()
		assert.False(t, ok)
	})

	t.Run("PolygonWKT invalid point", func(t *testing.T) {
		g := GeoFilters{
			Polygon: &PolygonFilter{
				Points: []GeoPoint{
					{Lat: FloatParam{Value: ptrFloat(27.0)}, Lng: FloatParam{Value: ptrFloat(-82.0)}},
					{Lat: FloatParam{}, Lng: FloatParam{Value: ptrFloat(-82.0)}},
					{Lat: FloatParam{Value: ptrFloat(27.5)}, Lng: FloatParam{Value: ptrFloat(-82.5)}},
				},
			},
		}
		_, ok := g.PolygonWKT()
		assert.False(t, ok)
	})
}

func TestSearchParamsMethods(t *testing.T) {
	t.Parallel()

	t.Run("SortKey", func(t *testing.T) {
		p := SearchParams{Sort: " LIST_PRICE "}
		assert.Equal(t, "list_price", p.SortKey())
	})

	t.Run("SortKey empty", func(t *testing.T) {
		p := SearchParams{}
		assert.Empty(t, p.SortKey())
	})

	t.Run("SortDirection", func(t *testing.T) {
		p := SearchParams{SortDir: " DESC "}
		assert.Equal(t, "desc", p.SortDirection())
	})

	t.Run("SortDirection empty", func(t *testing.T) {
		p := SearchParams{}
		assert.Empty(t, p.SortDirection())
	})

	t.Run("ActiveOnlyOrDefault with value", func(t *testing.T) {
		p := SearchParams{ActiveOnly: BoolParam{Value: ptrBool(false)}}
		assert.False(t, p.ActiveOnlyOrDefault())
	})

	t.Run("ActiveOnlyOrDefault without value", func(t *testing.T) {
		p := SearchParams{}
		assert.True(t, p.ActiveOnlyOrDefault())
	})

	t.Run("NormalizeSort", func(t *testing.T) {
		p := SearchParams{Sort: " LIST_PRICE "}
		assert.Equal(t, "list_price", p.NormalizeSort("on_market_date"))
	})

	t.Run("NormalizeSort empty", func(t *testing.T) {
		p := SearchParams{}
		assert.Equal(t, "on_market_date", p.NormalizeSort("on_market_date"))
	})

	t.Run("NormalizeSortDir", func(t *testing.T) {
		p := SearchParams{SortDir: " ASC "}
		assert.Equal(t, "asc", p.NormalizeSortDir("desc"))
	})

	t.Run("NormalizeSortDir empty", func(t *testing.T) {
		p := SearchParams{}
		assert.Equal(t, "desc", p.NormalizeSortDir("desc"))
	})
}

func TestCursorSortTime(t *testing.T) {
	t.Parallel()

	t.Run("valid time", func(t *testing.T) {
		p := SearchParams{}
		input := "2024-01-15T10:30:00.000000000Z"
		result, err := p.CursorSortTime(input)
		require.NoError(t, err)
		assert.Equal(t, 2024, result.Year())
		assert.Equal(t, time.January, result.Month())
		assert.Equal(t, 15, result.Day())
	})

	t.Run("invalid type", func(t *testing.T) {
		p := SearchParams{}
		_, err := p.CursorSortTime(12345)
		assert.Error(t, err)
	})

	t.Run("invalid format", func(t *testing.T) {
		p := SearchParams{}
		_, err := p.CursorSortTime("not-a-time")
		assert.Error(t, err)
	})
}

func TestFocusAreaNormalize(t *testing.T) {
	t.Parallel()

	t.Run("valid focus area with ID", func(t *testing.T) {
		id := int64(123)
		fa := &FocusArea{Type: "city", ID: &id}
		kind, resultID, ok := fa.normalize()
		assert.True(t, ok)
		assert.Equal(t, "city", kind)
		assert.Equal(t, int64(123), resultID)
	})

	t.Run("valid focus area with RefID", func(t *testing.T) {
		refID := int64(456)
		fa := &FocusArea{Type: "county", RefID: &refID}
		kind, resultID, ok := fa.normalize()
		assert.True(t, ok)
		assert.Equal(t, "county", kind)
		assert.Equal(t, int64(456), resultID)
	})

	t.Run("ID takes precedence over RefID", func(t *testing.T) {
		id := int64(123)
		refID := int64(456)
		fa := &FocusArea{Type: "city", ID: &id, RefID: &refID}
		_, resultID, ok := fa.normalize()
		assert.True(t, ok)
		assert.Equal(t, int64(123), resultID)
	})

	t.Run("nil focus area", func(t *testing.T) {
		var fa *FocusArea
		_, _, ok := fa.normalize()
		assert.False(t, ok)
	})

	t.Run("empty type", func(t *testing.T) {
		id := int64(123)
		fa := &FocusArea{Type: "", ID: &id}
		_, _, ok := fa.normalize()
		assert.False(t, ok)
	})

	t.Run("zero ID", func(t *testing.T) {
		id := int64(0)
		fa := &FocusArea{Type: "city", ID: &id}
		_, _, ok := fa.normalize()
		assert.False(t, ok)
	})
}

func TestLocationFiltersNormalizeFocusAreas(t *testing.T) {
	t.Parallel()

	t.Run("all focus area types", func(t *testing.T) {
		stateID := int64(1)
		countyID := int64(2)
		cityID := int64(3)
		subdivisionID := int64(4)
		postalID := int64(5)
		elemID := int64(6)
		middleID := int64(7)
		highID := int64(8)

		lf := LocationFilters{
			FocusAreas: []FocusArea{
				{Type: "state", ID: &stateID},
				{Type: "county", ID: &countyID},
				{Type: "city", ID: &cityID},
				{Type: "subdivision", ID: &subdivisionID},
				{Type: "postal_code", ID: &postalID},
				{Type: "elementary_school", ID: &elemID},
				{Type: "middle_school", ID: &middleID},
				{Type: "high_school", ID: &highID},
			},
		}
		lf.normalizeFocusAreas()

		assert.Equal(t, []int64{1}, lf.StateRefIDs.Values)
		assert.Equal(t, []int64{2}, lf.CountyRefIDs.Values)
		assert.Equal(t, []int64{3}, lf.CityRefIDs.Values)
		assert.Equal(t, []int64{4}, lf.SubdivisionRefIDs.Values)
		assert.Equal(t, []int64{5}, lf.PostalCodeRefIDs.Values)
		assert.Equal(t, []int64{6}, lf.ElementarySchoolRefIDs.Values)
		assert.Equal(t, []int64{7}, lf.MiddleSchoolRefIDs.Values)
		assert.Equal(t, []int64{8}, lf.HighSchoolRefIDs.Values)
	})

	t.Run("alternative type names", func(t *testing.T) {
		postalID := int64(1)
		elemID := int64(2)
		middleID := int64(3)
		highID := int64(4)

		lf := LocationFilters{
			FocusAreas: []FocusArea{
				{Type: "postal", ID: &postalID},
				{Type: "zip", RefID: &postalID},
				{Type: "elementary", ID: &elemID},
				{Type: "middle", ID: &middleID},
				{Type: "junior_high", RefID: &middleID},
				{Type: "high", ID: &highID},
			},
		}
		lf.normalizeFocusAreas()

		assert.Equal(t, []int64{1, 1}, lf.PostalCodeRefIDs.Values)
		assert.Equal(t, []int64{2}, lf.ElementarySchoolRefIDs.Values)
		assert.Equal(t, []int64{3, 3}, lf.MiddleSchoolRefIDs.Values)
		assert.Equal(t, []int64{4}, lf.HighSchoolRefIDs.Values)
	})
}

func TestIsSortAllowed(t *testing.T) {
	t.Parallel()

	allowed := []string{
		"list_price", "LIST_PRICE", " list_price ",
		"on_market_date",
		"year_built",
		"living_area",
		"lot_size_acres",
		"bedrooms_total",
		"bathrooms_total",
		"distance",
	}

	for _, sort := range allowed {
		t.Run("allowed: "+sort, func(t *testing.T) {
			assert.True(t, isSortAllowed(sort))
		})
	}

	notAllowed := []string{"invalid", "created_at", "random"}
	for _, sort := range notAllowed {
		t.Run("not allowed: "+sort, func(t *testing.T) {
			assert.False(t, isSortAllowed(sort))
		})
	}
}

func TestParseSearchPayload_GeoValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid distance filter", func(t *testing.T) {
		payload := []byte(`{"params":{"geo":{"distance":{"lat":27.5,"lng":-82.5,"radius_miles":10}}}}`)
		req, err := ParseSearchPayload(payload)
		require.NoError(t, err)
		require.NotNil(t, req.Params.Geo)
		assert.True(t, req.Params.Geo.HasDistance())
	})

	t.Run("invalid lat range", func(t *testing.T) {
		payload := []byte(`{"params":{"geo":{"distance":{"lat":100,"lng":-82.5,"radius_miles":10}}}}`)
		_, err := ParseSearchPayload(payload)
		assert.Error(t, err)
		vErr, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Contains(t, vErr.Fields, "params.geo.distance.lat")
	})

	t.Run("invalid lng range", func(t *testing.T) {
		payload := []byte(`{"params":{"geo":{"distance":{"lat":27.5,"lng":-200,"radius_miles":10}}}}`)
		_, err := ParseSearchPayload(payload)
		assert.Error(t, err)
		vErr, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Contains(t, vErr.Fields, "params.geo.distance.lng")
	})

	t.Run("missing distance radius", func(t *testing.T) {
		payload := []byte(`{"params":{"geo":{"distance":{"lat":27.5,"lng":-82.5}}}}`)
		_, err := ParseSearchPayload(payload)
		assert.Error(t, err)
		vErr, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Contains(t, vErr.Fields, "params.geo.distance.radius_miles")
	})

	t.Run("zero radius", func(t *testing.T) {
		payload := []byte(`{"params":{"geo":{"distance":{"lat":27.5,"lng":-82.5,"radius_miles":0}}}}`)
		_, err := ParseSearchPayload(payload)
		assert.Error(t, err)
		vErr, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Contains(t, vErr.Fields, "params.geo.distance.radius_miles")
	})

	t.Run("valid bbox filter", func(t *testing.T) {
		payload := []byte(`{"params":{"geo":{"bbox":{"west":-83,"south":27,"east":-82,"north":28}}}}`)
		req, err := ParseSearchPayload(payload)
		require.NoError(t, err)
		require.NotNil(t, req.Params.Geo.BBox)
	})

	t.Run("invalid bbox east <= west", func(t *testing.T) {
		payload := []byte(`{"params":{"geo":{"bbox":{"west":-82,"south":27,"east":-83,"north":28}}}}`)
		_, err := ParseSearchPayload(payload)
		assert.Error(t, err)
		vErr, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Contains(t, vErr.Fields, "params.geo.bbox.east")
	})

	t.Run("invalid bbox north <= south", func(t *testing.T) {
		payload := []byte(`{"params":{"geo":{"bbox":{"west":-83,"south":28,"east":-82,"north":27}}}}`)
		_, err := ParseSearchPayload(payload)
		assert.Error(t, err)
		vErr, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Contains(t, vErr.Fields, "params.geo.bbox.north")
	})

	t.Run("valid polygon filter", func(t *testing.T) {
		payload := []byte(`{"params":{"geo":{"polygon":{"points":[{"lat":27,"lng":-82},{"lat":27.5,"lng":-82},{"lat":27.5,"lng":-82.5}]}}}}`)
		req, err := ParseSearchPayload(payload)
		require.NoError(t, err)
		require.NotNil(t, req.Params.Geo.Polygon)
	})

	t.Run("polygon too few points", func(t *testing.T) {
		payload := []byte(`{"params":{"geo":{"polygon":{"points":[{"lat":27,"lng":-82}]}}}}`)
		_, err := ParseSearchPayload(payload)
		assert.Error(t, err)
		vErr, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Contains(t, vErr.Fields, "params.geo.polygon.points")
	})

	t.Run("polygon invalid point lat", func(t *testing.T) {
		payload := []byte(`{"params":{"geo":{"polygon":{"points":[{"lat":100,"lng":-82},{"lat":27.5,"lng":-82},{"lat":27.5,"lng":-82.5}]}}}}`)
		_, err := ParseSearchPayload(payload)
		assert.Error(t, err)
		vErr, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Contains(t, vErr.Fields, "params.geo.polygon.points[0].lat")
	})

	t.Run("polygon invalid point lng", func(t *testing.T) {
		payload := []byte(`{"params":{"geo":{"polygon":{"points":[{"lat":27,"lng":-200},{"lat":27.5,"lng":-82},{"lat":27.5,"lng":-82.5}]}}}}`)
		_, err := ParseSearchPayload(payload)
		assert.Error(t, err)
		vErr, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Contains(t, vErr.Fields, "params.geo.polygon.points[0].lng")
	})
}

func TestParseSearchPayload_SortValidation(t *testing.T) {
	t.Parallel()

	t.Run("valid sort", func(t *testing.T) {
		payload := []byte(`{"params":{"sort":"list_price","sort_dir":"desc"}}`)
		req, err := ParseSearchPayload(payload)
		require.NoError(t, err)
		assert.Equal(t, "list_price", req.Params.Sort)
		assert.Equal(t, "desc", req.Params.SortDir)
	})

	t.Run("invalid sort", func(t *testing.T) {
		payload := []byte(`{"params":{"sort":"invalid_sort"}}`)
		_, err := ParseSearchPayload(payload)
		assert.Error(t, err)
		vErr, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Contains(t, vErr.Fields, "params.sort")
	})

	t.Run("invalid sort_dir", func(t *testing.T) {
		payload := []byte(`{"params":{"sort":"list_price","sort_dir":"invalid"}}`)
		_, err := ParseSearchPayload(payload)
		assert.Error(t, err)
		vErr, ok := err.(*ValidationError)
		require.True(t, ok)
		assert.Contains(t, vErr.Fields, "params.sort_dir")
	})
}

func TestParseSearchPayload_ContextCombined(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"context":{"params":{"min_price":100000}},"params":{"max_price":500000}}`)
	_, err := ParseSearchPayload(payload)
	assert.Error(t, err)
	vErr, ok := err.(*ValidationError)
	require.True(t, ok)
	assert.Contains(t, vErr.Fields, "context")
}

func TestParseSearchPayload_NegativeOverallLimit(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"page":{"overall_limit":-1}}`)
	_, err := ParseSearchPayload(payload)
	assert.Error(t, err)
	vErr, ok := err.(*ValidationError)
	require.True(t, ok)
	assert.Contains(t, vErr.Fields, "page.overall_limit")
}

func TestParseSearchPayload_NegativeReturned(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"page":{"returned":-1}}`)
	_, err := ParseSearchPayload(payload)
	assert.Error(t, err)
	vErr, ok := err.(*ValidationError)
	require.True(t, ok)
	assert.Contains(t, vErr.Fields, "page.returned")
}

func TestNormalizeStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"normal", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"with spaces", []string{" a ", " b "}, []string{"a", "b"}},
		{"empty strings", []string{"a", "", "c"}, []string{"a", "c"}},
		{"whitespace only", []string{"  ", "\t", "a"}, []string{"a"}},
		{"empty input", []string{}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeStrings(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestIsNullJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  bool
	}{
		{"null", true},
		{"  null  ", true},
		{"", true},
		{"   ", true},
		{"{}", false},
		{"[]", false},
		{`"null"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, isNullJSON([]byte(tt.input)))
		})
	}
}

func TestExtractUnknownField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{`json: unknown field "foo"`, "foo"},
		{`json: unknown field "bar_baz"`, "bar_baz"},
		{`no quotes here`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, extractUnknownField(tt.input))
		})
	}
}

// Helpers
func ptr(i int) *int {
	return &i
}

func ptrFloat(f float64) *float64 {
	return &f
}

func ptrBool(b bool) *bool {
	return &b
}
