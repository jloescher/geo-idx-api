package api

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestProductionRouteOrder_AdminGISNotDomainAuth ensures admin routes register before /v1 MLS middleware.
func TestProductionRouteOrder_AdminGISNotDomainAuth(t *testing.T) {
	app := fiber.New()
	api := app.Group("/api")

	adminAPI := api.Group("/v1/admin", func(c *fiber.Ctx) error { return c.Next() })
	adminAPI.Post("/gis/probe", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true, "handler": "gis-probe"})
	})

	api.Group("/v1", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusForbidden).SendString("domain-middleware-blocked")
	}).Get("/listings", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest("POST", "/api/v1/admin/gis/probe", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == fiber.StatusForbidden && string(body) == "domain-middleware-blocked" {
		t.Fatalf("admin GIS route hit domain middleware; body=%q", body)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", resp.StatusCode, body)
	}
}
