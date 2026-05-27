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

	ins, del, err := repo.ExpandCityCountyPairs(ctx, gen)
	if err != nil {
		t.Fatal(err)
	}
	if ins+del == 0 {
		t.Fatal("expected city county expand changes")
	}
	var countyCount int
	if err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT county) FROM gis_cities
		WHERE city_name = $1 AND source_generation = $2 AND county IS NOT NULL
	`, "BackfillTestCity", gen).Scan(&countyCount); err != nil {
		t.Fatal(err)
	}
	if countyCount != 1 {
		t.Fatalf("single-county city: got %d county rows", countyCount)
	}
}

func TestExpandCityCountyPairsMultiCountyIntegration(t *testing.T) {
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
	gen := 99992
	leftSlug := "test-expand-left"
	rightSlug := "test-expand-right"
	// Two adjacent county boxes; city polygon spans both.
	leftCounty := `{"type":"Polygon","coordinates":[[[-83.0,28.0],[-82.5,28.0],[-82.5,28.5],[-83.0,28.5],[-83.0,28.0]]]}`
	rightCounty := `{"type":"Polygon","coordinates":[[[-82.5,28.0],[-82.0,28.0],[-82.0,28.5],[-82.5,28.5],[-82.5,28.0]]]}`
	cityGeom := `{"type":"Polygon","coordinates":[[[-82.55,28.1],[-82.45,28.1],[-82.45,28.4],[-82.55,28.4],[-82.55,28.1]]]}`

	if err := repo.BulkUpsertCounties(ctx, []CountyRow{
		{CountyName: "Expand Left", CountySlug: leftSlug, SourceKey: "test", GeometryJSON: leftCounty, SourceGeneration: gen},
		{CountyName: "Expand Right", CountySlug: rightSlug, SourceKey: "test", GeometryJSON: rightCounty, SourceGeneration: gen},
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.BulkUpsertCities(ctx, []CityRow{{
		CityName: "ExpandMultiCity", SourceKey: "test",
		GeometryJSON: cityGeom, SourceGeneration: gen,
	}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Pool.Exec(ctx, `DELETE FROM gis_cities WHERE city_name = $1`, "ExpandMultiCity")
		_, _ = db.Pool.Exec(ctx, `DELETE FROM gis_counties WHERE county_slug IN ($1, $2)`, leftSlug, rightSlug)
	})

	if _, _, err := repo.ExpandCityCountyPairs(ctx, gen); err != nil {
		t.Fatal(err)
	}
	var countyCount int
	if err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT county) FROM gis_cities
		WHERE city_name = $1 AND source_generation = $2 AND county IS NOT NULL
	`, "ExpandMultiCity", gen).Scan(&countyCount); err != nil {
		t.Fatal(err)
	}
	if countyCount < 2 {
		t.Fatalf("expected at least 2 county rows for multi-county city, got %d", countyCount)
	}
}
