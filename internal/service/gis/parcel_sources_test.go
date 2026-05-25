package gis

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestParcelSourceCatalogCoverage(t *testing.T) {
	catalog := ParcelSourceCatalog()
	if len(catalog) != 22 {
		t.Fatalf("catalog len=%d, want 22 (21 enabled + Osceola stub)", len(catalog))
	}
	enabled := 0
	slugs := map[string]bool{}
	for _, s := range catalog {
		if s.Enabled {
			enabled++
		}
		if s.CountySlug != "" {
			slugs[s.CountySlug] = true
		}
		if s.SourceKey == "" || s.QueryURL == "" {
			t.Fatalf("incomplete spec: %+v", s)
		}
	}
	if enabled != 21 {
		t.Fatalf("enabled=%d, want 21 (22 MLS counties minus Osceola)", enabled)
	}
	if len(slugs) != 22 {
		t.Fatalf("unique county slugs=%d, want 22", len(slugs))
	}
}

func TestSourcesForBBoxPinellas(t *testing.T) {
	bbox := BBox{West: -82.85, South: 27.95, East: -82.75, North: 28.05}
	sources := SourcesForBBox(bbox)
	if len(sources) == 0 {
		t.Fatal("expected pinellas source")
	}
	found := false
	for _, s := range sources {
		if s.CountySlug == "pinellas" {
			found = true
		}
	}
	if !found {
		t.Fatal("pinellas not in bbox sources")
	}
}

func TestArcGISClientPaginateMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("where") != "1=1" {
			t.Fatalf("where=%q", r.URL.Query().Get("where"))
		}
		if r.URL.Query().Get("geometry") != "" {
			t.Fatal("paginate mode should not send geometry")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":"FeatureCollection","features":[]}`))
	}))
	defer srv.Close()

	client := NewArcGISClient(config.GISConfig{SyncPageSize: 500})
	spec := ParcelSourceSpec{
		QueryURL:       srv.URL,
		SyncMode:       SyncModePaginate,
		ResponseFormat: FormatGeoJSON,
	}
	_, err := client.FetchParcelPage(spec, BBox{}, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
}

func TestArcGISClientWhereFilterMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("where") != "CNTYNAME='Martin'" {
			t.Fatalf("where=%q", r.URL.Query().Get("where"))
		}
		if r.URL.Query().Get("geometry") == "" {
			t.Fatal("where_filter mode should send geometry")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":"FeatureCollection","features":[]}`))
	}))
	defer srv.Close()

	client := NewArcGISClient(config.GISConfig{SyncPageSize: 500})
	spec := ParcelSourceSpec{
		QueryURL:       srv.URL,
		SyncMode:       SyncModeWhereFilter,
		ArcGISWhere:    "CNTYNAME='Martin'",
		ResponseFormat: FormatGeoJSON,
	}
	bbox := countyBBox("martin")
	_, err := client.FetchParcelPage(spec, bbox, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
}
