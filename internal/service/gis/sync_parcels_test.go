package gis

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestArcGISClientFetchBBoxPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("resultOffset") != "0" {
			t.Fatalf("resultOffset=%q", r.URL.Query().Get("resultOffset"))
		}
		if r.URL.Query().Get("f") != "geojson" {
			t.Fatalf("f=%q", r.URL.Query().Get("f"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "FeatureCollection",
			"features": []map[string]any{
				{
					"type": "Feature",
					"geometry": map[string]any{
						"type":        "Polygon",
						"coordinates": []any{[]any{[]any{-82.8, 27.9}, []any{-82.79, 27.9}, []any{-82.79, 27.91}, []any{-82.8, 27.91}, []any{-82.8, 27.9}}},
					},
					"properties": map[string]any{"PARCELID": "TEST-1"},
				},
			},
		})
	}))
	defer srv.Close()

	cfg := config.GISConfig{SyncPageSize: 2000}
	client := NewArcGISClient(cfg)
	src := Source{QueryURL: srv.URL, CountyCO: "52"}
	body, err := client.FetchBBoxPage(src, BBox{West: -87, South: 24, East: -79, North: 31}, 0, 2000)
	if err != nil {
		t.Fatal(err)
	}
	page, err := ParseFeatureCollection(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Features) != 1 {
		t.Fatalf("features=%d", len(page.Features))
	}
}

func TestArcGISClientFetchLayerPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":     "FeatureCollection",
			"features": []any{},
		})
	}))
	defer srv.Close()

	client := NewArcGISClient(config.GISConfig{SyncPageSize: 500})
	body, err := client.FetchLayerPage(srv.URL, "1=1", 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	page, err := ParseFeatureCollection(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Features) != 0 {
		t.Fatalf("features=%d", len(page.Features))
	}
}
