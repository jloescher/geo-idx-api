package gis

import (
	"encoding/json"
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestParseFeatureCollection(t *testing.T) {
	body := []byte(`{
		"type":"FeatureCollection",
		"exceededTransferLimit": true,
		"features":[
			{"type":"Feature","geometry":{"type":"Point","coordinates":[-82.8,27.9]},"properties":{"PARCELID":"A1"}}
		]
	}`)
	page, err := ParseFeatureCollection(body)
	if err != nil {
		t.Fatal(err)
	}
	if !page.ExceededTransferLimit {
		t.Fatal("expected exceededTransferLimit")
	}
	if len(page.Features) != 1 {
		t.Fatalf("features=%d", len(page.Features))
	}
}

func TestParseQueryType(t *testing.T) {
	cases := map[string]QueryType{
		"":       QueryTypeParcel,
		"parcel": QueryTypeParcel,
		"city":   QueryTypeCity,
		"county": QueryTypeCounty,
		"zip":    QueryTypeZip,
		"postal": QueryTypeZip,
	}
	for in, want := range cases {
		if got := ParseQueryType(in); got != want {
			t.Fatalf("ParseQueryType(%q)=%q want %q", in, got, want)
		}
	}
}

func TestParseLimitCap(t *testing.T) {
	cfg := config.GISConfig{MaxFeatures: 100}
	if got := ParseLimit("999", cfg); got != 100 {
		t.Fatalf("limit=%d", got)
	}
	if got := ParseLimit("", cfg); got != 100 {
		t.Fatalf("default limit=%d", got)
	}
}

func TestFeaturesToGeoJSONEmpty(t *testing.T) {
	out, err := featuresToGeoJSON(nil)
	if err != nil {
		t.Fatal(err)
	}
	var fc map[string]any
	if err := json.Unmarshal(out, &fc); err != nil {
		t.Fatal(err)
	}
	if fc["type"] != "FeatureCollection" {
		t.Fatalf("type=%v", fc["type"])
	}
}
