package gisrepo

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/quantyralabs/idx-api/internal/repository"
)

func TestQueryParcelsByBBoxIntegration(t *testing.T) {
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

	repo := New(db)
	parcelGeom := `{"type":"Polygon","coordinates":[[[-82.82,27.96],[-82.81,27.96],[-82.81,27.97],[-82.82,27.97],[-82.82,27.96]]]}`
	rows := []ParcelRow{{
		ParcelID:         "integration-test-1",
		SourceKey:        "pinellas_enterprise_parcels",
		County:           "pinellas",
		GeometryJSON:     parcelGeom,
		Properties:       json.RawMessage(`{}`),
		SourceGeneration: 1,
	}}

	if err := repo.BulkUpsertParcels(ctx, rows, 500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Pool.Exec(ctx, `DELETE FROM gis_parcels WHERE parcel_id = $1`, "integration-test-1")
	})

	has, err := repo.HasParcelsInBBox(ctx, -82.83, 27.95, -82.79, 27.98, []string{"pinellas"})
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("expected parcel in bbox")
	}

	feats, err := repo.QueryParcelsByBBox(ctx, -82.83, 27.95, -82.79, 27.98, []string{"pinellas"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(feats) == 0 {
		t.Fatal("expected features")
	}
}
