package gis

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// BBox holds west, south, east, north in WGS84 degrees.
type BBox struct {
	West, South, East, North float64
}

// ParseBBox reads bbox from query params (bbox=west,south,east,north or N/S/E/W).
func ParseBBox(c *fiber.Ctx) (BBox, error) {
	if raw := strings.TrimSpace(c.Query("bbox")); raw != "" {
		parts := strings.Split(raw, ",")
		if len(parts) != 4 {
			return BBox{}, fmt.Errorf("bbox must have four comma-separated values")
		}
		vals := make([]float64, 4)
		for i, p := range parts {
			v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
			if err != nil {
				return BBox{}, fmt.Errorf("invalid bbox")
			}
			vals[i] = v
		}
		return BBox{West: vals[0], South: vals[1], East: vals[2], North: vals[3]}, nil
	}
	north, err1 := parseFloatQuery(c, "north")
	south, err2 := parseFloatQuery(c, "south")
	east, err3 := parseFloatQuery(c, "east")
	west, err4 := parseFloatQuery(c, "west")
	if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
		return BBox{West: west, South: south, East: east, North: north}, nil
	}
	lat, latErr := parseFloatQuery(c, "lat")
	lng, lngErr := parseFloatQuery(c, "lng")
	radius, radErr := parseFloatQuery(c, "radius")
	if latErr == nil && lngErr == nil && radErr == nil && radius > 0 {
		deg := radius / 111320.0
		return BBox{
			West:  lng - deg,
			East:  lng + deg,
			South: lat - deg,
			North: lat + deg,
		}, nil
	}
	return BBox{}, fmt.Errorf("bbox, north/south/east/west, or lat/lng/radius required")
}

func parseFloatQuery(c *fiber.Ctx, key string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(c.Query(key)), 64)
}

// SpanDeg returns max of width and height in degrees.
func (b BBox) SpanDeg() float64 {
	w := b.East - b.West
	h := b.North - b.South
	if w < 0 {
		w = -w
	}
	if h < 0 {
		h = -h
	}
	if w > h {
		return w
	}
	return h
}

// EsriEnvelope returns xmin,ymin,xmax,ymax for ArcGIS geometry param.
func (b BBox) EsriEnvelope() string {
	return fmt.Sprintf("%f,%f,%f,%f", b.West, b.South, b.East, b.North)
}

// Centroid returns approximate center.
func (b BBox) Centroid() (lat, lng float64) {
	return (b.South + b.North) / 2, (b.West + b.East) / 2
}
