package mls_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func fixturePath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "docs", name)
}

func loadFixtureListing(t *testing.T, path string, index int) (map[string]any, json.RawMessage) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if index >= len(doc.Value) {
		t.Fatalf("fixture index %d out of range (%d)", index, len(doc.Value))
	}
	raw := doc.Value[index]
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		t.Fatal(err)
	}
	return row, raw
}

func TestBuildListingRecordStellarFixture(t *testing.T) {
	row, raw := loadFixtureListing(t, fixturePath("bridge_interactive/stellar_50_listings.json"), 0)
	resolver := mls.NewResoFieldResolver()
	rec, action := mls.BuildListingRecord("stellar", mls.MirrorProviderBridge, row, raw, resolver, nil)
	if action != mls.RowActionUpsert {
		t.Fatalf("action %s", action)
	}
	if rec.ListingKey == "" {
		t.Fatal("missing listing key")
	}
	if rec.ListPrice == nil {
		t.Fatal("expected list_price")
	}
	if len(rec.RawData) == 0 {
		t.Fatal("expected raw_data")
	}
}

func TestBuildListingRecordBeachesFloodZone(t *testing.T) {
	b, err := os.ReadFile(fixturePath("spark/beaches_50_listings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	resolver := mls.NewResoFieldResolver()
	for i, raw := range doc.Value {
		var row map[string]any
		if err := json.Unmarshal(raw, &row); err != nil {
			t.Fatal(err)
		}
		if stringValue(row[mls.BeachesSparkFloodZoneField]) == "" {
			continue
		}
		rec, action := mls.BuildListingRecord("beaches", mls.MirrorProviderSpark, row, raw, resolver, nil)
		if action != mls.RowActionUpsert {
			t.Fatalf("index %d action %s", i, action)
		}
		if rec.FloodZoneCode == nil || *rec.FloodZoneCode != "X" {
			t.Fatalf("index %d flood_zone_code %v", i, rec.FloodZoneCode)
		}
		if !rec.LowRiskFloodZoneYN {
			t.Fatalf("index %d low_risk_flood_zone_yn want true", i)
		}
		return
	}
	t.Fatal("no beaches fixture row with flood zone field")
}

func stringValue(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func TestBuildListingRecordBeachesFees(t *testing.T) {
	row, raw := loadFixtureListing(t, fixturePath("spark/beaches_50_listings.json"), 0)
	resolver := mls.NewResoFieldResolver()
	rec, action := mls.BuildListingRecord("beaches", mls.MirrorProviderSpark, row, raw, resolver, nil)
	if action != mls.RowActionUpsert {
		t.Fatalf("action %s", action)
	}
	if rec.EstimatedTotalMonthlyFees == nil || *rec.EstimatedTotalMonthlyFees != 500.22 {
		t.Fatalf("fees %v", rec.EstimatedTotalMonthlyFees)
	}
	if rec.FloodZoneCode != nil {
		t.Fatalf("first beaches row should not set flood in fixture: %v", *rec.FloodZoneCode)
	}
	if rec.City == nil || *rec.City != "Boca Raton" {
		t.Fatalf("city %v", rec.City)
	}
}

func TestBuildListingRecordClosedDeletes(t *testing.T) {
	row := map[string]any{
		"ListingKey":     "stellar:abc",
		"StandardStatus": "Closed",
	}
	raw, _ := json.Marshal(row)
	rec, action := mls.BuildListingRecord("stellar", mls.MirrorProviderBridge, row, raw, mls.NewResoFieldResolver(), nil)
	if action != mls.RowActionDelete {
		t.Fatalf("action %s", action)
	}
	if rec.DatasetSlug != "stellar" || rec.ListingKey != "stellar:abc" {
		t.Fatalf("rec %+v", rec)
	}
}

func TestFixtureUpsertsAllHaveListPrice(t *testing.T) {
	resolver := mls.NewResoFieldResolver()
	for _, spec := range []struct {
		path, dataset string
		provider      mls.MirrorProvider
	}{
		{"bridge_interactive/stellar_50_listings.json", "stellar", mls.MirrorProviderBridge},
		{"spark/beaches_50_listings.json", "beaches", mls.MirrorProviderSpark},
	} {
		b, err := os.ReadFile(fixturePath(spec.path))
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Value []json.RawMessage `json:"value"`
		}
		if err := json.Unmarshal(b, &doc); err != nil {
			t.Fatal(err)
		}
		for i, raw := range doc.Value {
			var row map[string]any
			if err := json.Unmarshal(raw, &row); err != nil {
				t.Fatal(err)
			}
			rec, action := mls.BuildListingRecord(spec.dataset, spec.provider, row, raw, resolver, nil)
			if action != mls.RowActionUpsert {
				continue
			}
			if rec.ListPrice == nil {
				t.Fatalf("%s row %d: missing list_price", spec.path, i)
			}
			apiPrice, ok := mls.ResolveListPrice(row)
			if !ok {
				t.Fatalf("%s row %d: API has no price fields", spec.path, i)
			}
			if *rec.ListPrice != apiPrice {
				t.Fatalf("%s row %d: list_price %v want %v", spec.path, i, *rec.ListPrice, apiPrice)
			}
		}
	}
}

func TestBuildListingRecordBathrooms6602(t *testing.T) {
	row := map[string]any{
		"ListingKey":              "stellar:bad-baths",
		"StandardStatus":          "Active",
		"ListPrice":               250000,
		"BathroomsTotalInteger":   6602,
	}
	raw, _ := json.Marshal(row)
	rec, action := mls.BuildListingRecord("stellar", mls.MirrorProviderBridge, row, raw, mls.NewResoFieldResolver(), nil)
	if action != mls.RowActionUpsert {
		t.Fatalf("action %s", action)
	}
	if rec.BathroomsTotalDecimal != nil {
		t.Fatalf("bathrooms %v", rec.BathroomsTotalDecimal)
	}
	if rec.ListPrice == nil || *rec.ListPrice != 250000 {
		t.Fatalf("list_price %v", rec.ListPrice)
	}
}

func TestBuildListingRecordLivingAreaSqft(t *testing.T) {
	row := map[string]any{
		"ListingKey":     "beaches:gla",
		"StandardStatus": "Active",
		"ListPrice":      500000,
		"LivingArea":     8891,
	}
	raw, _ := json.Marshal(row)
	rec, action := mls.BuildListingRecord("beaches", mls.MirrorProviderSpark, row, raw, mls.NewResoFieldResolver(), nil)
	if action != mls.RowActionUpsert {
		t.Fatalf("action %s", action)
	}
	if rec.LivingArea == nil || *rec.LivingArea != 8891 {
		t.Fatalf("living_area %v", rec.LivingArea)
	}
}

func TestBuildListingRecordSkipsActiveWithoutPrice(t *testing.T) {
	row := map[string]any{
		"ListingKey":     "stellar:no-price",
		"StandardStatus": "Active",
	}
	raw, _ := json.Marshal(row)
	_, action := mls.BuildListingRecord("stellar", mls.MirrorProviderBridge, row, raw, mls.NewResoFieldResolver(), nil)
	if action != mls.RowActionSkip {
		t.Fatalf("action %s", action)
	}
}

func TestBuildListingRecordCoordinatesGeoJSON(t *testing.T) {
	row := map[string]any{
		"ListingKey":     "92db0afbb51861ea702ccfe33390e6f3",
		"StandardStatus": "Active",
		"ListPrice":      100000,
		"Coordinates": map[string]any{
			"coordinates": []any{-82.45, 27.95},
		},
	}
	raw, _ := json.Marshal(row)
	rec, action := mls.BuildListingRecord("stellar", mls.MirrorProviderBridge, row, raw, mls.NewResoFieldResolver(), nil)
	if action != mls.RowActionUpsert {
		t.Fatalf("action %s", action)
	}
	if rec.Latitude == nil || rec.Longitude == nil {
		t.Fatal("expected lat/lng from Coordinates")
	}
	if *rec.Latitude != 27.95 || *rec.Longitude != -82.45 {
		t.Fatalf("lat/lng got %v %v", *rec.Latitude, *rec.Longitude)
	}
}
