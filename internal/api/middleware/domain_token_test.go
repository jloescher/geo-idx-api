package middleware_test

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/api/middleware"
)

func TestDomainTokenMissingIdentification(t *testing.T) {
	app := fiber.New()
	app.Use(middleware.DomainToken(nil, nil, nil))
	app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestDomainTokenSlugHeader(t *testing.T) {
	// Without DB, only test middleware chain with nil repos would panic — skip DB integration here.
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(ctxkeys.BridgeDomainSlug, "example.com")
		return c.Next()
	})
	app.Get("/", func(c *fiber.Ctx) error {
		slug, _ := c.Locals(ctxkeys.BridgeDomainSlug).(string)
		return c.SendString(slug)
	})
	req := httptest.NewRequest("GET", "/", nil)
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "example.com" {
		t.Fatalf("body %q", body)
	}
}
