package sync

import (
	"net/url"
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestBridgePropertyReplicationURL(t *testing.T) {
	s := NewBridgeSync(config.Config{
		Bridge: config.BridgeConfig{
			Host:       "https://api.bridgedataoutput.com",
			PathPrefix: "api/v2",
			ResoRoot:   "reso/odata",
		},
	}, nil)

	got := s.propertyReplicationURL("stellar")
	want := "https://api.bridgedataoutput.com/api/v2/OData/stellar/Property/replication"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBridgeReplicationSkipsExpandWhenFullProperty(t *testing.T) {
	s := NewBridgeSync(config.Config{
		Bridge: config.BridgeConfig{
			SyncFullProperty: true,
			SyncExpand:       "Media,OpenHouses,Rooms,UnitTypes",
		},
	}, nil)
	q := url.Values{}
	s.applySyncExpand(q)
	if q.Get("$expand") != "" {
		t.Fatalf("full property should not set $expand, got %q", q.Get("$expand"))
	}
}

func TestBridgeReplicationQuerySetsBridgeExpand(t *testing.T) {
	s := NewBridgeSync(config.Config{
		Bridge: config.BridgeConfig{
			SyncFullProperty: false,
			SyncExpand:       "Media,OpenHouses,Rooms,UnitTypes",
		},
	}, nil)
	q := url.Values{}
	s.applySyncExpand(q)
	if q.Get("$expand") != "Media,OpenHouses,Rooms,UnitTypes" {
		t.Fatalf("expand = %q", q.Get("$expand"))
	}
}

func TestBridgeSyncSelectListOmitsMediaWhenDisabled(t *testing.T) {
	s := NewBridgeSync(config.Config{
		Bridge: config.BridgeConfig{SyncIncludeMedia: false},
	}, nil)
	sel := s.syncSelectList("stellar")
	if containsField(sel, "Media") {
		t.Fatalf("expected Media omitted, got %q", sel)
	}
}

func containsField(csv, field string) bool {
	for _, part := range splitComma(csv) {
		if part == field {
			return true
		}
	}
	return false
}

func splitComma(s string) []string {
	var out []string
	for _, p := range stringsSplit(s, ',') {
		out = append(out, p)
	}
	return out
}

func stringsSplit(s string, sep rune) []string {
	var parts []string
	start := 0
	for i, c := range s {
		if c == sep {
			parts = append(parts, trim(s[start:i]))
			start = i + 1
		}
	}
	parts = append(parts, trim(s[start:]))
	return parts
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
