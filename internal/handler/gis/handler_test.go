package gis

import (
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

func TestHandlerShowDefaultType(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := t.Context()
	db, err := repository.NewFromDSN(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cfg := config.Config{GIS: config.GISConfig{MaxBboxSpanDeg: 0.35, MaxFeatures: 50}}
	h := NewHandler(cfg, db, nil)
	app := fiber.New()
	app.Get("/api/v1/gis", h.Show)

	req := httptest.NewRequest("GET", "/api/v1/gis?bbox=-82.83,27.95,-82.79,27.98&type=county", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
}
