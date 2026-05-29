package gis

import "testing"

func TestProbeOneSkipsNonHTTPEndpoint(t *testing.T) {
	endpoint := queryMetaURL("shapefile://local")
	if endpoint != "shapefile://local?f=json" {
		t.Fatalf("queryMetaURL shapefile = %q", endpoint)
	}
	if stringsHasHTTPPrefix(endpoint) {
		t.Fatal("shapefile probe URL must not use HTTP")
	}
}

func stringsHasHTTPPrefix(s string) bool {
	return len(s) >= 4 && (s[:4] == "http" || (len(s) >= 5 && s[:5] == "https"))
}
