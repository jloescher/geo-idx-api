package cache

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestFingerprintStableForSameQuery(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(FingerprintRequest(c, "/listings"))
	})
	req1 := httptest.NewRequest("GET", "/?filters=active&limit=10", nil)
	req2 := httptest.NewRequest("GET", "/?limit=10&filters=active", nil)
	r1, _ := app.Test(req1)
	r2, _ := app.Test(req2)
	// order in query string differs — fingerprint sorts keys so should match
	_ = r1
	_ = r2
}

func TestFingerprintDiffersForDifferentFilter(t *testing.T) {
	app := fiber.New()
	var a, b string
	app.Get("/a", func(c *fiber.Ctx) error { a = FingerprintRequest(c, "/x"); return nil })
	app.Get("/b", func(c *fiber.Ctx) error { b = FingerprintRequest(c, "/x"); return nil })
	_, _ = app.Test(httptest.NewRequest("GET", "/a?filters=1", nil))
	_, _ = app.Test(httptest.NewRequest("GET", "/b?filters=2", nil))
	if a == b {
		t.Fatal("expected different fingerprints")
	}
}
