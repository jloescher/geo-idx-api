package gis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureImportFileReadableMissing(t *testing.T) {
	err := ensureImportFileReadable(filepath.Join(t.TempDir(), "missing.zip"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "upload file not found") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "GIS_IMPORT_PATH") {
		t.Fatalf("expected volume hint: %v", err)
	}
}

func TestEnsureImportFileReadableOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Parcels.zip")
	if err := os.WriteFile(path, []byte("test"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := ensureImportFileReadable(path); err != nil {
		t.Fatal(err)
	}
}
