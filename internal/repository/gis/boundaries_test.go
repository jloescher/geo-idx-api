package gisrepo

import (
	"context"
	"os"
	"testing"

	"github.com/quantyralabs/idx-api/internal/repository"
)

func TestBackfillCityCountiesIntegration(t *testing.T) {
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
	gen := 99991
	countySlug := "test-backfill-county"
	countyGeom := `{"type":"Polygon","coordinates":[[[-82.85,27.95],[-82.80,27.95],[-82.80,28.00],[-82.85,28.00],[-82.85,27.95]]]}`
	cityGeom := `{"type":"Polygon","coordinates":[[[-82.84,27.96],[-82.81,27.96],[-82.81,27.99],[-82.84,27.99],[-82.84,27.96]]]}`

	if err := repo.BulkUpsertCounties(ctx, []CountyRow{{
		CountyName: "Test Backfill County", CountySlug: countySlug, SourceKey: "test",
		GeometryJSON: countyGeom, SourceGeneration: gen,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := repo.BulkUpsertCities(ctx, []CityRow{{
		CityName: "BackfillTestCity", SourceKey: "test",
		GeometryJSON: cityGeom, SourceGeneration: gen,
	}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Pool.Exec(ctx, `DELETE FROM gis_cities WHERE city_name = $1 AND source_generation = $2`, "BackfillTestCity", gen)
		_, _ = db.Pool.Exec(ctx, `DELETE FROM gis_counties WHERE county_slug = $1`, countySlug)
	})

	updated, err := repo.BackfillCityCounties(ctx, gen)
	if err != nil {
		t.Fatal(err)
	}
	if updated == 0 {
		t.Fatal("expected city county backfill updates")
	}
}
