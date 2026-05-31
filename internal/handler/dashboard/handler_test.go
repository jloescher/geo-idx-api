package dashboard

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

func TestMonitoringJSONUnauthorized(t *testing.T) {
	cfg := config.Config{}
	logger := testLogger()
	db := &repository.DB{}
	h := NewHandler(cfg, db, nil, logger)

	app := fiber.New()
	app.Get("/dashboard/monitoring/data", h.SessionAuthMiddleware, h.MonitoringJSON)

	req := httptest.NewRequest("GET", "/dashboard/monitoring/data", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}

func TestFormValuesMultipleDatasets(t *testing.T) {
	app := fiber.New()
	var got []string
	app.Post("/test", func(c *fiber.Ctx) error {
		got = formValues(c, "mls_datasets[]")
		return c.SendStatus(204)
	})

	body := "domain_slug=example.com&mls_datasets[]=bridge_stellar&mls_datasets[]=spark_beaches"
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	if len(got) != 2 {
		t.Fatalf("datasets: %v", got)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRedirectLegacyDashboardRoutes(t *testing.T) {
	cfg := config.Config{}
	h := NewHandler(cfg, &repository.DB{}, nil, testLogger())
	app := fiber.New()
	app.Get("/dashboard/setup", h.redirectToDomains)
	app.Get("/dashboard/api-keys", h.redirectToDomains)

	for _, path := range []string{"/dashboard/setup?verified=1", "/dashboard/api-keys"} {
		req := httptest.NewRequest("GET", path, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusFound {
			t.Fatalf("%s status: %d", path, resp.StatusCode)
		}
		loc := resp.Header.Get("Location")
		if !strings.Contains(loc, "/dashboard/domains") {
			t.Fatalf("%s location: %s", path, loc)
		}
	}
}
