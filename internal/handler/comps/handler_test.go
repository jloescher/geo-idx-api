package comps

import (
	"bytes"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
)

func TestRunInvalidJSON(t *testing.T) {
	h := NewHandler(config.Config{}, nil, slog.Default())
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(ctxkeys.MLSFeedCode, "bridge_stellar")
		return c.Next()
	})
	app.Post("/comps/run", h.Run)

	req := httptest.NewRequest("POST", "/comps/run", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status %d want 400", resp.StatusCode)
	}
}

func TestRunValidationError(t *testing.T) {
	h := NewHandler(config.Config{}, nil, slog.Default())
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(ctxkeys.MLSFeedCode, "bridge_stellar")
		return c.Next()
	})
	app.Post("/comps/run", h.Run)

	body := `{"mode":"A","scope":{"type":"radius"},"subject":{"type":"off_market","lat":27.95,"lng":-82.46}}`
	req := httptest.NewRequest("POST", "/comps/run", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusUnprocessableEntity {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d body %s want 422", resp.StatusCode, b)
	}
}
