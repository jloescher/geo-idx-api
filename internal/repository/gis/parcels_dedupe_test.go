package gisrepo

import "testing"

func TestDedupeParcelRowsKeepsLastDuplicate(t *testing.T) {
	rows := []ParcelRow{
		{ParcelID: "P1", SourceKey: "florida_statewide_cadastral", County: "old"},
		{ParcelID: "P2", SourceKey: "florida_statewide_cadastral", County: "b"},
		{ParcelID: "P1", SourceKey: "florida_statewide_cadastral", County: "new"},
	}
	got := dedupeParcelRows(rows)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	byID := map[string]string{}
	for _, r := range got {
		byID[r.ParcelID] = r.County
	}
	if byID["P1"] != "new" {
		t.Fatalf("P1 county = %q, want new", byID["P1"])
	}
	if byID["P2"] != "b" {
		t.Fatalf("P2 county = %q, want b", byID["P2"])
	}
}

func TestDedupeParcelRowsSameKeyDifferentSource(t *testing.T) {
	rows := []ParcelRow{
		{ParcelID: "P1", SourceKey: "source_a"},
		{ParcelID: "P1", SourceKey: "source_b"},
	}
	got := dedupeParcelRows(rows)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (distinct source_key)", len(got))
	}
}
