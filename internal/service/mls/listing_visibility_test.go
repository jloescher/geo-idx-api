package mls

import (
	"encoding/json"
	"testing"
)

func TestIsListingPublicCompliant(t *testing.T) {
	falseVal := false
	trueVal := true

	if !IsListingPublicCompliant(ListingIDXFlags{DatasetSlug: "beaches"}) {
		t.Fatal("expected compliant when flags nil")
	}
	if IsListingPublicCompliant(ListingIDXFlags{
		DatasetSlug:                    "beaches",
		InternetEntireListingDisplayYN: &falseVal,
	}) {
		t.Fatal("expected non-compliant when entire listing display false")
	}
	if IsListingPublicCompliant(ListingIDXFlags{
		DatasetSlug:          "stellar",
		IDXParticipationYN:   &falseVal,
	}) {
		t.Fatal("expected stellar non-compliant when IDX participation false")
	}
	if !IsListingPublicCompliant(ListingIDXFlags{
		DatasetSlug:          "stellar",
		IDXParticipationYN:   &trueVal,
	}) {
		t.Fatal("expected stellar compliant when IDX participation true")
	}
}

func TestFilterMediaForPublicSearch(t *testing.T) {
	root := map[string]any{
		"Media": []any{
			map[string]any{"MediaKey": "1", "Permission": []any{"Private"}},
			map[string]any{"MediaKey": "2", "Permission": []any{"Public"}},
		},
	}
	flags := ListingIDXFlags{}
	if !ApplyPublicListingVisibility(root, flags, VisibilityPublicSearch) {
		t.Fatal("listing should remain")
	}
	media, ok := root["Media"].([]any)
	if !ok || len(media) != 1 {
		t.Fatalf("expected one public media item, got %#v", root["Media"])
	}
}

func TestApplyPublicListingVisibilityJSONDropsListing(t *testing.T) {
	falseVal := false
	body, ok := ApplyPublicListingVisibilityJSON(
		json.RawMessage(`{"ListingKey":"x","InternetEntireListingDisplayYN":false}`),
		ListingIDXFlags{InternetEntireListingDisplayYN: &falseVal},
		VisibilityPublicSearch,
	)
	if ok || len(body) != 0 {
		t.Fatalf("expected drop, ok=%v len=%d", ok, len(body))
	}
}
