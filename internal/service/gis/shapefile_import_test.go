package gis

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

func writeTestZip(t *testing.T, path string, entries map[string][]byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for name, data := range entries {
		hdr := &zip.FileHeader{Name: name, Method: zip.Store}
		wr, err := w.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := wr.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestFindShapefileInZip_nestedPath(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "nested.zip")
	writeTestZip(t, zipPath, map[string][]byte{
		"workspace/scripts/pascoshp/pasco_parcels.shp": []byte("shp"),
		"workspace/scripts/pascoshp/pasco_parcels.dbf": []byte("dbf"),
		"readme.txt": []byte("ignore"),
	})
	got, err := findShapefileInZip(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "workspace/scripts/pascoshp/pasco_parcels.shp"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFindShapefileInZip_prefersShallowRoot(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "multi.zip")
	writeTestZip(t, zipPath, map[string][]byte{
		"Parcels.shp":                    make([]byte, 100),
		"deep/nested/other_parcels.shp": make([]byte, 9999),
	})
	got, err := findShapefileInZip(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Parcels.shp" {
		t.Fatalf("got %q want Parcels.shp", got)
	}
}

func TestShapefileOGRSource(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "county.zip")
	writeTestZip(t, zipPath, map[string][]byte{
		"folder/layer.shp": []byte("shp"),
	})
	src, err := shapefileOGRSource(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "/vsizip/" + zipPath + "/folder/layer.shp"
	if src != want {
		t.Fatalf("got %q want %q", src, want)
	}
	shpPath := filepath.Join(dir, "flat.shp")
	if err := os.WriteFile(shpPath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	src, err = shapefileOGRSource(shpPath)
	if err != nil || src != shpPath {
		t.Fatalf("direct shp: got %q err=%v", src, err)
	}
}

func TestSaveUploadStream_enforcesMaxBytes(t *testing.T) {
	dir := t.TempDir()
	_, _, err := SaveUploadStream(dir, "pasco", "upload.zip", bytes.NewReader([]byte("0123456789")), 5)
	if err == nil || !strings.Contains(err.Error(), "max size") {
		t.Fatalf("expected max size error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "pasco", "upload.zip")); !os.IsNotExist(err) {
		t.Fatal("expected partial upload removed")
	}
}

func TestFindShapefileInZip_pascoFixture(t *testing.T) {
	zipPath := filepath.Join("..", "..", "..", "temp", "pasco_parcels.zip")
	if _, err := os.Stat(zipPath); err != nil {
		t.Skip("temp/pasco_parcels.zip not present")
	}
	got, err := findShapefileInZip(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "workspace/scripts/pascoshp/pasco_parcels.shp"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	src, err := shapefileOGRSource(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(src, want) {
		t.Fatalf("ogr source %q should end with %q", src, want)
	}
}
