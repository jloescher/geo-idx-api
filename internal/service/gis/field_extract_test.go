package gis

import (
	"encoding/json"
	"testing"
)

func TestExtractParcelRowStatewide(t *testing.T) {
	feat := ArcGISFeature{
		Geometry: json.RawMessage(`{"type":"Polygon","coordinates":[[[-82.8,27.9],[-82.79,27.9],[-82.79,27.91],[-82.8,27.91],[-82.8,27.9]]]}`),
		Properties: map[string]any{
			"PARCELID":  "P-123",
			"PHY_ADDR1": "100 Main St",
			"OWN_NAME":  "Jane Doe",
			"JV":        250000.0,
		},
	}
	row, err := ExtractParcelRow(feat, "florida_statewide_cadastral", "pinellas", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if row.ParcelID != "P-123" {
		t.Fatalf("parcel_id=%q", row.ParcelID)
	}
	if row.County != "pinellas" {
		t.Fatalf("county=%q", row.County)
	}
	if row.SiteAddress == nil || *row.SiteAddress != "100 Main St" {
		t.Fatalf("site_address=%v", row.SiteAddress)
	}
	if row.JustValue == nil || *row.JustValue != 250000 {
		t.Fatalf("just_value=%v", row.JustValue)
	}
}

func TestExtractCityRow(t *testing.T) {
	feat := ArcGISFeature{
		Geometry: json.RawMessage(`{"type":"Polygon","coordinates":[[[-82.8,27.9],[-82.79,27.9],[-82.79,27.91],[-82.8,27.91],[-82.8,27.9]]]}`),
		Properties: map[string]any{
			"NAME":   "Clearwater",
			"COUNTY": "Pinellas",
		},
	}
	row, err := ExtractCityRow(feat, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if row.CityName != "Clearwater" {
		t.Fatalf("city_name=%q", row.CityName)
	}
}

func TestExtractCountyRow(t *testing.T) {
	feat := ArcGISFeature{
		Geometry: json.RawMessage(`{"type":"Polygon","coordinates":[[[-82.8,27.9],[-82.79,27.9],[-82.79,27.91],[-82.8,27.91],[-82.8,27.9]]]}`),
		Properties: map[string]any{
			"NAME": "Pinellas",
			"FIPS": "12103",
		},
	}
	row, err := ExtractCountyRow(feat, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if row.CountyName != "Pinellas" {
		t.Fatalf("county_name=%q", row.CountyName)
	}
	if row.FIPSCode == nil || *row.FIPSCode != "12103" {
		t.Fatalf("fips=%v", row.FIPSCode)
	}
}

func TestExtractZipRow(t *testing.T) {
	feat := ArcGISFeature{
		Geometry: json.RawMessage(`{"type":"Polygon","coordinates":[[[-82.8,27.9],[-82.79,27.9],[-82.79,27.91],[-82.8,27.91],[-82.8,27.9]]]}`),
		Properties: map[string]any{
			"ZIP": "33755",
		},
	}
	row, err := ExtractZipRow(feat, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if row.ZipCode != "33755" {
		t.Fatalf("zip=%q", row.ZipCode)
	}
}

func TestExtractParcelRowMissingGeometry(t *testing.T) {
	_, err := ExtractParcelRow(ArcGISFeature{Properties: map[string]any{"PARCELID": "x"}}, "k", "pinellas", 1, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
