package gis

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
)

func TestApplyTeaserTruncatesFeatures(t *testing.T) {
	in := []byte(`{"type":"FeatureCollection","features":[
		{"type":"Feature","geometry":{"type":"Point","coordinates":[-82.123456789,27.987654321]}},
		{"type":"Feature","geometry":{"type":"Point","coordinates":[-82.2,27.9]}},
		{"type":"Feature","geometry":{"type":"Point","coordinates":[-82.3,27.8]}}
	]}`)
	cfg := config.GISConfig{TeaserMaxFeatures: 2, TeaserCoordDecimals: 4}
	out, teaser := applyTeaser(in, cfg, false)
	if !teaser {
		t.Fatal("expected teaser applied")
	}
	var fc map[string]any
	if json.Unmarshal(out, &fc) != nil {
		t.Fatal("invalid json")
	}
	feats, _ := fc["features"].([]any)
	if len(feats) != 2 {
		t.Fatalf("features %d want 2", len(feats))
	}
	f0 := feats[0].(map[string]any)
	geom := f0["geometry"].(map[string]any)
	coords := geom["coordinates"].([]any)
	if coords[0].(float64) != -82.1235 {
		t.Fatalf("rounded lng %v", coords[0])
	}
}

func TestApplyTeaserSkippedForFullAccess(t *testing.T) {
	in := []byte(`{"type":"FeatureCollection","features":[{"type":"Feature"}]}`)
	out, teaser := applyTeaser(in, config.GISConfig{TeaserMaxFeatures: 1}, true)
	if teaser || string(out) != string(in) {
		t.Fatalf("teaser=%v body changed", teaser)
	}
}

func TestRequestFullAccessFromLocals(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		c.Locals(ctxkeys.MLSFullAccess, false)
		if requestFullAccess(c) {
			return c.SendString("full")
		}
		return c.SendString("teaser")
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
