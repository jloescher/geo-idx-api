package api

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestAdminGISRoutesNotBlockedByV1Middleware(t *testing.T) {
	app := fiber.New()
	api := app.Group("/api")

	adminHit := false
	adminAPI := api.Group("/v1/admin", func(c *fiber.Ctx) error {
		c.Locals("dashboard_user_id", int64(1))
		return c.Next()
	})
	adminAPI.Post("/gis/probe", func(c *fiber.Ctx) error {
		adminHit = true
		return c.JSON(fiber.Map{"ok": true})
	})

	domainHit := false
	api.Group("/v1", func(c *fiber.Ctx) error {
		domainHit = true
		return c.Status(fiber.StatusForbidden).SendString("domain-middleware")
	}).Get("/listings", func(c *fiber.Ctx) error { return c.SendString("listings") })

	req := httptest.NewRequest("POST", "/api/v1/admin/gis/probe", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if domainHit {
		t.Fatalf("domain middleware ran for admin GIS route; body=%q", body)
	}
	if !adminHit {
		t.Fatalf("admin GIS handler did not run; status=%d body=%q", resp.StatusCode, body)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", resp.StatusCode, body)
	}
}
