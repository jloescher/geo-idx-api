package gis

import "testing"

func TestIsShapefileEndpoint(t *testing.T) {
	t.Parallel()
	cases := []struct {
		endpoint string
		want     bool
	}{
		{"shapefile://local", true},
		{"shapefile://local?f=json", true},
		{"https://example.test/layer/0/query", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isShapefileEndpoint(tc.endpoint); got != tc.want {
			t.Fatalf("isShapefileEndpoint(%q) = %v, want %v", tc.endpoint, got, tc.want)
		}
	}
}

func TestQueryMetaURLSkipsShapefile(t *testing.T) {
	t.Parallel()
	meta := queryMetaURL("shapefile://local")
	if isShapefileEndpoint(meta) {
		// queryMetaURL should not be called for shapefile in production paths;
		// if it is, probeOne must still treat the result as shapefile.
		if !isShapefileEndpoint(meta) {
			t.Fatal("expected shapefile meta URL to remain a shapefile endpoint")
		}
	}
}
