package gisrepo

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/quantyralabs/idx-api/internal/repository"
)

func TestDeleteParcelSourceFullIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	db, err := repository.NewFromDSN(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	const sourceKey = "integration-purge-test"
	repo := New(db)

	t.Cleanup(func() {
		_, _ = db.Pool.Exec(ctx, `DELETE FROM gis_parcels WHERE source_key = $1`, sourceKey)
		_, _ = db.Pool.Exec(ctx, `DELETE FROM gis_source_states WHERE source_key = $1`, sourceKey)
		_, _ = db.Pool.Exec(ctx, `DELETE FROM gis_parcel_sources WHERE source_key = $1`, sourceKey)
	})

	if err := repo.EnsureParcelSourceCatalog(ctx, []ParcelSourceRow{{
		SourceKey:  sourceKey,
		CountySlug: "pinellas",
		QueryURL:   "https://example.test/arcgis/rest/services/test/FeatureServer/0",
		SyncMode:   "shapefile",
		MLSFeed:    "stellar",
		Enabled:    true,
		Priority:   999,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := repo.EnsureSourceState(ctx, sourceKey); err != nil {
		t.Fatal(err)
	}
	parcelGeom := `{"type":"Polygon","coordinates":[[[-82.82,27.96],[-82.81,27.96],[-82.81,27.97],[-82.82,27.97],[-82.82,27.96]]]}`
	if err := repo.BulkUpsertParcels(ctx, []ParcelRow{{
		ParcelID:         "integration-purge-parcel",
		SourceKey:        sourceKey,
		County:           "pinellas",
		GeometryJSON:     parcelGeom,
		Properties:       json.RawMessage(`{}`),
		SourceGeneration: 1,
	}}, 500); err != nil {
		t.Fatal(err)
	}

	parcelsDeleted, err := repo.DeleteParcelSourceFull(ctx, sourceKey)
	if err != nil {
		t.Fatal(err)
	}
	if parcelsDeleted != 1 {
		t.Fatalf("parcelsDeleted = %d, want 1", parcelsDeleted)
	}

	var parcelCount int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM gis_parcels WHERE source_key = $1`, sourceKey).Scan(&parcelCount); err != nil {
		t.Fatal(err)
	}
	if parcelCount != 0 {
		t.Fatalf("gis_parcels count = %d, want 0", parcelCount)
	}

	var stateCount int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM gis_source_states WHERE source_key = $1`, sourceKey).Scan(&stateCount); err != nil {
		t.Fatal(err)
	}
	if stateCount != 0 {
		t.Fatalf("gis_source_states count = %d, want 0", stateCount)
	}

	var catalogCount int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM gis_parcel_sources WHERE source_key = $1`, sourceKey).Scan(&catalogCount); err != nil {
		t.Fatal(err)
	}
	if catalogCount != 0 {
		t.Fatalf("gis_parcel_sources count = %d, want 0", catalogCount)
	}
}
