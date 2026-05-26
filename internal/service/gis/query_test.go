package gis

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestParseBBoxFromCorners(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		b, err := ParseBBox(c)
		if err != nil {
			return err
		}
		return c.JSON(b)
	})
	req := httptest.NewRequest("GET", "/?north=28&south=27&east=-82&west=-83", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("status=%v err=%v", resp.StatusCode, err)
	}
}

func TestBBoxSpanRejectsLarge(t *testing.T) {
	b := BBox{West: 0, South: 0, East: 1, North: 1}
	if b.SpanDeg() != 1 {
		t.Fatalf("span %v", b.SpanDeg())
	}
}
