package gis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGeoJSONFeature(t *testing.T) {
	line := []byte(`{"type":"Feature","geometry":{"type":"Point","coordinates":[-82.5,27.9]},"properties":{"PARCELID":"123"}}`)
	feat, err := ParseGeoJSONFeature(line)
	if err != nil {
		t.Fatal(err)
	}
	if feat.Properties["PARCELID"] != "123" {
		t.Fatalf("unexpected properties: %v", feat.Properties)
	}
	if len(feat.Geometry) == 0 {
		t.Fatal("expected geometry")
	}
}

func TestStreamGeoJSONSeqFeatures(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.geojsonl")
	content := `{"type":"Feature","geometry":{"type":"Point","coordinates":[0,0]},"properties":{"PARCELID":"a"}}
{"type":"Feature","geometry":{"type":"Point","coordinates":[1,1]},"properties":{"PARCELID":"b"}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	var ids []string
	n, err := streamGeoJSONSeqFeatures(path, func(f ArcGISFeature) error {
		ids = append(ids, f.Properties["PARCELID"].(string))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || len(ids) != 2 {
		t.Fatalf("got count=%d ids=%v", n, ids)
	}
}
