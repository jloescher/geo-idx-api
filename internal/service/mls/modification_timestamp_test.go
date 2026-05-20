package mls

import (
	"strings"
	"testing"
	"time"
)

func TestResolveModificationTimestampStellarPrefersBridge(t *testing.T) {
	bridge := time.Date(2024, 6, 2, 12, 0, 0, 0, time.UTC)
	mod := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	row := map[string]any{
		"BridgeModificationTimestamp": bridge.Format(time.RFC3339),
		"ModificationTimestamp":       mod.Format(time.RFC3339),
	}
	ts := ResolveModificationTimestamp("stellar", row)
	if ts == nil || !ts.Equal(bridge) {
		t.Fatalf("got %v want bridge ts", ts)
	}
}

func TestResolveModificationTimestampBeachesUsesModification(t *testing.T) {
	mod := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	bridge := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	row := map[string]any{
		"ModificationTimestamp":       mod.Format(time.RFC3339),
		"BridgeModificationTimestamp": bridge.Format(time.RFC3339),
	}
	ts := ResolveModificationTimestamp("beaches", row)
	if ts == nil || !ts.Equal(mod) {
		t.Fatalf("got %v want modification ts", ts)
	}
}

func TestModificationODataField(t *testing.T) {
	if ModificationODataField("stellar") != "BridgeModificationTimestamp" {
		t.Fatal("stellar")
	}
	if ModificationODataField("beaches") != "ModificationTimestamp" {
		t.Fatal("beaches")
	}
}

func TestODataGTFilterBareISO8601(t *testing.T) {
	since := time.Date(2025, 5, 20, 4, 51, 45, 0, time.UTC)
	got := ODataGTFilter("BridgeModificationTimestamp", since)
	want := "BridgeModificationTimestamp gt 2025-05-20T04:51:45Z"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if strings.Contains(got, "datetime'") {
		t.Fatalf("Bridge OData must not use datetime'' wrapper: %q", got)
	}
}
