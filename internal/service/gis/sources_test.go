package gis

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestCountyBBox(t *testing.T) {
	p := countyBBox("pinellas")
	if p.West == 0 || p.East == 0 {
		t.Fatalf("pinellas bbox=%+v", p)
	}
	h := countyBBox("hillsborough")
	if h.West == 0 || h.East == 0 {
		t.Fatalf("hillsborough bbox=%+v", h)
	}
}

func TestParcelSyncTargetsDefault(t *testing.T) {
	svc := NewParcelSyncService(config.Config{}, nil, nil, nil)
	targets := svc.parcelSyncTargets("")
	if len(targets) != 3 {
		t.Fatalf("expected 3 sync targets (statewide + pinellas + hillsborough), got %d", len(targets))
	}
	keys := map[string]bool{}
	for _, tgt := range targets {
		if tgt.QueryURL == "" {
			t.Fatalf("empty query url for %s", tgt.SourceKey)
		}
		keys[tgt.SourceKey] = true
	}
	if !keys[FloridaStatewideCadastralKey] || !keys["pinellas_enterprise_parcels"] || !keys["hillsborough_hc_parcels"] {
		t.Fatalf("missing expected sources: %+v", keys)
	}
}

func TestParcelSourceURLs(t *testing.T) {
	if HillsboroughParcelsQueryURL == "" {
		t.Fatal("hillsborough parcel URL must be set")
	}
}
