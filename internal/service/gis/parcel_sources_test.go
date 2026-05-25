package gis

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestEnabledParcelSourcesForConfig_PinellasOptIn(t *testing.T) {
	off := EnabledParcelSourcesForConfig(config.GISConfig{SyncPinellasEnterprise: false})
	for _, s := range off {
		if s.SourceKey == "pinellas_enterprise_parcels" {
			t.Fatal("pinellas sync should be off when GIS_SYNC_PINELLAS_ENTERPRISE=false")
		}
	}
	on := EnabledParcelSourcesForConfig(config.GISConfig{SyncPinellasEnterprise: true})
	found := false
	for _, s := range on {
		if s.SourceKey == "pinellas_enterprise_parcels" {
			found = true
		}
	}
	if !found {
		t.Fatal("pinellas sync missing when GIS_SYNC_PINELLAS_ENTERPRISE=true")
	}
}

func TestParcelSourceCatalogCoverage(t *testing.T) {
	catalog := ParcelSourceCatalog()
	if len(catalog) != 3 {
		t.Fatalf("catalog len=%d, want 3 (statewide + pinellas + hillsborough)", len(catalog))
	}
	if catalog[0].SourceKey != FloridaStatewideCadastralKey {
		t.Fatalf("primary source=%q", catalog[0].SourceKey)
	}
	for _, s := range catalog {
		if !s.Enabled {
			t.Fatalf("source disabled: %s", s.SourceKey)
		}
		if s.SourceKey == "" || s.QueryURL == "" {
			t.Fatalf("incomplete spec: %+v", s)
		}
	}
}

func TestSourcesForBBoxPinellas(t *testing.T) {
	bbox := BBox{West: -82.85, South: 27.95, East: -82.75, North: 28.05}
	sources := SourcesForBBox(bbox)
	if len(sources) == 0 {
		t.Fatal("expected pinellas failover source")
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

func TestSourcesForBBoxOsceolaNoFailover(t *testing.T) {
	bbox := countyBBox("osceola")
	sources := SourcesForBBox(bbox)
	for _, s := range sources {
		if s.CountySlug == "osceola" {
			t.Fatal("osceola should not have a dedicated failover source")
		}
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
		if r.URL.Query().Get("outSR") != "4326" {
			t.Fatalf("outSR=%q", r.URL.Query().Get("outSR"))
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

func TestArcGISClientBBoxModeOutSR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("outSR") != "4326" {
			t.Fatalf("outSR=%q", r.URL.Query().Get("outSR"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":"FeatureCollection","features":[]}`))
	}))
	defer srv.Close()

	client := NewArcGISClient(config.GISConfig{SyncPageSize: 500})
	spec := ParcelSourceSpec{
		QueryURL:       srv.URL,
		SyncMode:       SyncModeBBox,
		ResponseFormat: FormatGeoJSON,
	}
	bbox := countyBBox("hillsborough")
	_, err := client.FetchParcelPage(spec, bbox, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
}
