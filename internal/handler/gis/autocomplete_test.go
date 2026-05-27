package gis

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

func TestAutocompleteCities(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := t.Context()
	db, err := repository.NewFromDSN(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cfg := config.Config{}
	h := NewHandler(cfg, db, nil)
	app := fiber.New()
	app.Get("/api/v1/gis/autocomplete/cities", h.AutocompleteCities)

	req := httptest.NewRequest("GET", "/api/v1/gis/autocomplete/cities?q=tam&limit=5", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	var results []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		t.Fatal(err)
	}
	// Empty GIS DB is ok; non-empty should include label with pipe for county.
	for _, row := range results {
		if _, ok := row["city"]; !ok {
			t.Fatalf("missing city: %+v", row)
		}
	}
}
