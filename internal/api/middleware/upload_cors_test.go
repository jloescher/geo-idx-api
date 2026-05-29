package middleware_test

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/middleware"
	"github.com/quantyralabs/idx-api/internal/config"
)

func TestUploadCORSAllowsPlatformOrigin(t *testing.T) {
	cfg := config.Config{
		Idx: config.IdxURLsConfig{
			PlatformURL: "https://idx.quantyralabs.cc",
		},
	}
	app := fiber.New()
	app.Options("/upload", middleware.UploadCORS(cfg))
	app.Post("/upload", middleware.UploadCORS(cfg), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusCreated)
	})

	req := httptest.NewRequest(fiber.MethodOptions, "/upload", nil)
	req.Header.Set("Origin", "https://idx.quantyralabs.cc")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("status=%d want 204", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://idx.quantyralabs.cc" {
		t.Fatalf("Allow-Origin=%q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Allow-Credentials=%q", got)
	}
}

func TestUploadCORSRejectsUnknownOrigin(t *testing.T) {
	cfg := config.Config{
		Idx: config.IdxURLsConfig{
			PlatformURL: "https://idx.quantyralabs.cc",
		},
	}
	app := fiber.New()
	app.Post("/upload", middleware.UploadCORS(cfg), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusCreated)
	})

	req := httptest.NewRequest(fiber.MethodPost, "/upload", nil)
	req.Header.Set("Origin", "https://evil.example")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Allow-Origin=%q want empty", got)
	}
}
