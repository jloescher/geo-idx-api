package mls_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func TestExtractMediaJSONAndStrip(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"ListingKey":     "k1",
		"StandardStatus": "Active",
		"ListPrice":      100,
		"Media": []map[string]any{
			{"MediaKey": "m1", "MediaURL": "https://example/1.jpg"},
		},
	})
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		t.Fatal(err)
	}
	media, ok := mls.ExtractMediaJSON(row)
	if !ok {
		t.Fatal("expected media present")
	}
	stripped := mls.StripMediaFromRaw(raw)
	var check map[string]any
	if err := json.Unmarshal(stripped, &check); err != nil {
		t.Fatal(err)
	}
	if _, has := check["Media"]; has {
		t.Fatal("Media should be stripped from raw_data")
	}
	merged := mls.MergeListingJSON(stripped, media)
	if err := json.Unmarshal(merged, &check); err != nil {
		t.Fatal(err)
	}
	mediaArr, ok := check["Media"].([]any)
	if !ok || len(mediaArr) != 1 {
		t.Fatalf("merged Media = %#v", check["Media"])
	}
}

func TestBuildListingRecordBeachesSeparatesMedia(t *testing.T) {
	// beaches_50_listings.json: replication page with $expand=Media,Unit,Room,OpenHouse.
	// Media[] uses Spark CDN URLs; Room/Unit/OpenHouse remain on the property row (not split).
	row, raw := loadFixtureListing(t, fixturePath("spark/beaches_50_listings.json"), 0)
	resolver := mls.NewResoFieldResolver()
	rec, action := mls.BuildListingRecord("beaches", mls.MirrorProviderSpark, row, raw, resolver, nil)
	if action != mls.RowActionUpsert {
		t.Fatalf("action %s", action)
	}
	if !rec.HasMedia {
		t.Fatal("expected beaches fixture row with expanded Media")
	}

	var mediaItems []map[string]any
	if err := json.Unmarshal(rec.Media, &mediaItems); err != nil {
		t.Fatal(err)
	}
	if len(mediaItems) < 2 {
		t.Fatalf("expected multiple photos, got %d", len(mediaItems))
	}
	first := mediaItems[0]
	mediaURL, _ := first["MediaURL"].(string)
	if mediaURL == "" || !strings.Contains(mediaURL, "sparkplatform.com") {
		t.Fatalf("expected Spark MediaURL, got %q", mediaURL)
	}
	if first["MediaKey"] == nil || first["MediaKey"] == "" {
		t.Fatal("expected MediaKey on beaches media item")
	}
	if first["Order"] == nil {
		t.Fatal("expected Order on beaches media item")
	}

	var stripped map[string]any
	if err := json.Unmarshal(rec.RawData, &stripped); err != nil {
		t.Fatal(err)
	}
	if _, has := stripped["Media"]; has {
		t.Fatal("raw_data must not contain Media")
	}
	if _, has := stripped["Room"]; has {
		t.Fatal("raw_data must not contain Room")
	}
	if !rec.HasRoom {
		t.Fatal("expected Room in room column")
	}

	merged := mls.MergeMirrorListing(rec.RawData, mls.ExpandedPayload{
		Media: rec.Media, HasMedia: rec.HasMedia,
		Room: rec.Room, HasRoom: rec.HasRoom,
	}, rec.CustomFields)
	var mergedRow map[string]any
	if err := json.Unmarshal(merged, &mergedRow); err != nil {
		t.Fatal(err)
	}
	mergedMedia, ok := mergedRow["Media"].([]any)
	if !ok || len(mergedMedia) != len(mediaItems) {
		t.Fatalf("merged Media count = %d, want %d", len(mergedMedia), len(mediaItems))
	}
}

func TestBuildListingRecordSeparatesMedia(t *testing.T) {
	row, raw := loadFixtureListing(t, fixturePath("bridge_interactive/stellar_50_listings.json"), 1)
	resolver := mls.NewResoFieldResolver()
	rec, action := mls.BuildListingRecord("stellar", mls.MirrorProviderBridge, row, raw, resolver, nil)
	if action != mls.RowActionUpsert {
		t.Fatalf("action %s", action)
	}
	if !rec.HasMedia {
		t.Fatal("expected fixture row with Media")
	}
	if len(rec.Media) == 0 {
		t.Fatal("expected media json")
	}
	var stripped map[string]any
	if err := json.Unmarshal(rec.RawData, &stripped); err != nil {
		t.Fatal(err)
	}
	if _, has := stripped["Media"]; has {
		t.Fatal("raw_data must not contain Media")
	}
}
