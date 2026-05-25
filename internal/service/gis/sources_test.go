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
	if len(targets) != 21 {
		t.Fatalf("expected 21 enabled MLS targets, got %d", len(targets))
	}
	for _, tgt := range targets {
		if tgt.QueryURL == "" {
			t.Fatalf("empty query url for %s", tgt.SourceKey)
		}
	}
}

func TestParcelSourceURLs(t *testing.T) {
	if HillsboroughParcelsQueryURL == "" {
		t.Fatal("hillsborough parcel URL must be set")
	}
}
