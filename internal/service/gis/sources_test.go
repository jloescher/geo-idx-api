package gis

import "testing"

func TestSyncBBoxForCounty(t *testing.T) {
	p := syncBBoxForCounty("pinellas")
	if p.West != pinellasBBox.West || p.East != pinellasBBox.East {
		t.Fatalf("pinellas bbox=%+v", p)
	}
	h := syncBBoxForCounty("hillsborough")
	if h.West != hillsboroughBBox.West {
		t.Fatalf("hillsborough bbox=%+v", h)
	}
}

func TestParcelSourceURLs(t *testing.T) {
	if FloridaStatewideCadastralQueryURL == "" || HillsboroughParcelsQueryURL == "" {
		t.Fatal("parcel URL constants must be set")
	}
	for _, tgt := range parcelSyncTargets() {
		if tgt.Source.QueryURL == "" {
			t.Fatalf("empty query url for %s", tgt.SourceKey)
		}
	}
}
