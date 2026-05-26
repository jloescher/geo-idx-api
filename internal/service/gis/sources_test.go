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
	if len(targets) != 2 {
		t.Fatalf("expected 2 sync targets (statewide + hillsborough), got %d", len(targets))
	}
	keys := map[string]bool{}
	for _, tgt := range targets {
		if tgt.QueryURL == "" {
			t.Fatalf("empty query url for %s", tgt.SourceKey)
		}
		keys[tgt.SourceKey] = true
	}
	if !keys[FloridaStatewideCadastralKey] || !keys["hillsborough_hc_parcels"] {
		t.Fatalf("missing expected sources: %+v", keys)
	}
	if keys["pinellas_enterprise_parcels"] {
		t.Fatal("pinellas sync should require GIS_SYNC_PINELLAS_ENTERPRISE=true")
	}

	cfg := config.Config{GIS: config.GISConfig{SyncPinellasEnterprise: true}}
	svc2 := NewParcelSyncService(cfg, nil, nil, nil)
	targets2 := svc2.parcelSyncTargets("")
	if len(targets2) != 3 {
		t.Fatalf("with pinellas opt-in expected 3 targets, got %d", len(targets2))
	}
}

func TestParcelSourceURLs(t *testing.T) {
	if HillsboroughParcelsQueryURL == "" {
		t.Fatal("hillsborough parcel URL must be set")
	}
}
