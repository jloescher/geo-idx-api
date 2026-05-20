package mls

import (
	"encoding/json"
	"testing"
)

func TestMergeMirrorListingFlatCustomFields(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"ListingKey": "x",
		"ListPrice":  100,
	})
	custom, _ := json.Marshal(map[string]any{
		"_sp_Extra": "y",
		"ListPrice": 999,
	})
	merged := MergeMirrorListing(raw, ExpandedPayload{}, custom)
	var out map[string]any
	if err := json.Unmarshal(merged, &out); err != nil {
		t.Fatal(err)
	}
	if out["_sp_Extra"] != "y" {
		t.Fatalf("_sp_Extra = %#v", out["_sp_Extra"])
	}
	if out["ListPrice"] != float64(100) {
		t.Fatalf("raw_data wins on collision: ListPrice = %#v", out["ListPrice"])
	}
	if _, has := out["custom_fields"]; has {
		t.Fatal("must not emit nested custom_fields key")
	}
}

func TestNormalizeBridgeExpandKeys(t *testing.T) {
	row := map[string]any{
		"Rooms":      []any{map[string]any{"RoomType": "Bedroom"}},
		"UnitTypes":  []any{},
		"OpenHouses": []any{},
	}
	NormalizeBridgeExpandKeys(row)
	if row["Room"] == nil {
		t.Fatal("expected Room alias from Rooms")
	}
}

func TestBuildCustomFieldsIncludesUnmappedScalars(t *testing.T) {
	row := map[string]any{
		"ListingKey":     "k",
		"StandardStatus": "Active",
		"ListPrice":      1,
		"STELLAR_Foo":    "bar",
		"Media":          []any{},
	}
	custom := BuildCustomFields(row, MirrorProviderBridge, []string{"Media"})
	var m map[string]any
	if err := json.Unmarshal(custom, &m); err != nil {
		t.Fatal(err)
	}
	if m["STELLAR_Foo"] != "bar" {
		t.Fatalf("custom = %#v", m)
	}
	if _, has := m["ListingKey"]; has {
		t.Fatal("mapped scalar should not be in custom_fields")
	}
}
