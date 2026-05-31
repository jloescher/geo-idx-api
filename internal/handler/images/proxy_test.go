package images

import "testing"

func TestFindMediaURL_stellarMediaKey(t *testing.T) {
	listingKey := "5d6013f5cf0fdf36a62557a4dac50567"
	photoID := listingKey + "-m1"
	url := "https://d37ukvrrv3in12.cloudfront.net/listings/stellar/" + listingKey + "-1/originalphoto.jpg"
	items := []map[string]any{
		{
			"MediaKey": photoID,
			"MediaURL": url,
		},
	}
	got := findMediaURL(items, photoID)
	if got != url {
		t.Fatalf("findMediaURL = %q, want %q", got, url)
	}
}

func TestFindMediaURL_photoIdField(t *testing.T) {
	url := "https://cdn.example/photo.jpg"
	items := []map[string]any{
		{
			"PhotoId":  "773806724_1",
			"MediaURL": url,
		},
	}
	got := findMediaURL(items, "773806724_1")
	if got != url {
		t.Fatalf("findMediaURL = %q, want %q", got, url)
	}
}

func TestFindMediaURL_numericMediaKey(t *testing.T) {
	url := "https://cdn.example/numeric.jpg"
	items := []map[string]any{
		{
			"MediaKey": float64(42),
			"MediaURL": url,
		},
	}
	got := findMediaURL(items, "42")
	if got != url {
		t.Fatalf("findMediaURL = %q, want %q", got, url)
	}
}

func TestFindMediaURL_noMatch(t *testing.T) {
	items := []map[string]any{
		{"MediaKey": "other-m1", "MediaURL": "https://cdn.example/x.jpg"},
	}
	if got := findMediaURL(items, "missing"); got != "" {
		t.Fatalf("findMediaURL = %q, want empty", got)
	}
}

func TestFindMediaURL_skipsEmptyMediaURL(t *testing.T) {
	items := []map[string]any{
		{"MediaKey": "abc-m1", "MediaURL": ""},
		{"MediaKey": "abc-m1", "MediaURL": "https://cdn.example/ok.jpg"},
	}
	got := findMediaURL(items, "abc-m1")
	if got != "https://cdn.example/ok.jpg" {
		t.Fatalf("findMediaURL = %q", got)
	}
}

func TestMediaItemMatchesPhotoID(t *testing.T) {
	item := map[string]any{"MediaKey": "k-m1", "PhotoId": ""}
	if !mediaItemMatchesPhotoID(item, "k-m1") {
		t.Fatal("expected MediaKey match")
	}
	if mediaItemMatchesPhotoID(item, "other") {
		t.Fatal("expected no match")
	}
}
