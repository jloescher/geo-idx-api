package gis

import (
	"encoding/json"
	"testing"
)

func TestCountySlugFromCONOOsceola(t *testing.T) {
	slug, ok := CountySlugFromCONO(59)
	if !ok || slug != "osceola" {
		t.Fatalf("CO_NO 59 → %q ok=%v", slug, ok)
	}
}

func TestCountySlugFromCONOPinellas(t *testing.T) {
	slug, ok := CountySlugFromCONO(52)
	if !ok || slug != "pinellas" {
		t.Fatalf("CO_NO 52 → %q ok=%v", slug, ok)
	}
}

func TestIsMLSPilotCounty(t *testing.T) {
	if !IsMLSPilotCounty("osceola") {
		t.Fatal("osceola should be MLS pilot")
	}
	if IsMLSPilotCounty("duval") {
		t.Fatal("duval is not MLS pilot")
	}
}

func TestCONOFromProperties(t *testing.T) {
	coNo, ok := CONOFromProperties(map[string]any{"CO_NO": float64(59)})
	if !ok || coNo != 59 {
		t.Fatalf("CO_NO=%d ok=%v", coNo, ok)
	}
}

func TestExtractParcelRowStatewideOsceola(t *testing.T) {
	feat := ArcGISFeature{
		Geometry: json.RawMessage(`{"type":"Polygon","coordinates":[[[-81.2,28.1],[-81.19,28.1],[-81.19,28.11],[-81.2,28.11],[-81.2,28.1]]]}`),
		Properties: map[string]any{
			"PARCEL_ID_": "OSC-123",
			"CO_NO":      float64(59),
			"PHY_ADDR1":  "100 Main St",
			"OWN_NAME":   "Jane Doe",
			"JV":         250000.0,
		},
	}
	coNo, ok := CONOFromProperties(feat.Properties)
	if !ok {
		t.Fatal("missing CO_NO")
	}
	slug, ok := CountySlugFromCONO(coNo)
	if !ok || slug != "osceola" {
		t.Fatalf("county=%q", slug)
	}
	row, err := ExtractParcelRow(feat, FloridaStatewideCadastralKey, slug, 1, nil, []string{"PARCEL_ID_"})
	if err != nil {
		t.Fatal(err)
	}
	if row.County != "osceola" {
		t.Fatalf("county=%q", row.County)
	}
	if row.ParcelID != "OSC-123" {
		t.Fatalf("parcel_id=%q", row.ParcelID)
	}
}
