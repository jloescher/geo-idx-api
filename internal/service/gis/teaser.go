package gis

import (
	"encoding/json"
	"math"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
)

// requestFullAccess reports whether the caller may receive full GIS payloads.
// Domain identification is always full; PATs without idx:full are teaser-tier when enabled.
func requestFullAccess(c *fiber.Ctx) bool {
	v, ok := c.Locals(ctxkeys.MLSFullAccess).(bool)
	if !ok {
		return true
	}
	return v
}

// applyTeaser caps feature count and coordinate precision for non-full-access callers.
// Revenue impact: idx:access tokens see parcel context without full cadastral fidelity.
func applyTeaser(geojson []byte, cfg config.GISConfig, fullAccess bool) ([]byte, bool) {
	if fullAccess {
		return geojson, false
	}
	maxFeatures := cfg.TeaserMaxFeatures
	if maxFeatures <= 0 {
		maxFeatures = 40
	}
	decimals := cfg.TeaserCoordDecimals
	if decimals <= 0 {
		decimals = 4
	}
	var fc map[string]any
	if json.Unmarshal(geojson, &fc) != nil {
		return geojson, false
	}
	truncated := truncateFeatureCollection(fc, maxFeatures)
	roundFeatureCollectionCoords(fc, decimals)
	out, err := json.Marshal(fc)
	if err != nil {
		return geojson, false
	}
	return out, truncated
}

func truncateFeatureCollection(fc map[string]any, max int) bool {
	feats, ok := fc["features"].([]any)
	if !ok || len(feats) <= max {
		return false
	}
	fc["features"] = feats[:max]
	return true
}

func roundFeatureCollectionCoords(fc map[string]any, decimals int) {
	feats, ok := fc["features"].([]any)
	if !ok {
		return
	}
	for _, f := range feats {
		feat, ok := f.(map[string]any)
		if !ok {
			continue
		}
		geom, ok := feat["geometry"].(map[string]any)
		if !ok {
			continue
		}
		if coords, ok := geom["coordinates"]; ok {
			geom["coordinates"] = roundCoordsAny(coords, decimals)
		}
	}
}

func roundCoordsAny(v any, decimals int) any {
	switch t := v.(type) {
	case float64:
		return roundCoord(t, decimals)
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = roundCoordsAny(item, decimals)
		}
		return out
	default:
		return v
	}
}

func roundCoord(v float64, decimals int) float64 {
	if decimals < 0 {
		return v
	}
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}
