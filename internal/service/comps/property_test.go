package comps

import (
	"encoding/json"
	"testing"
)

func TestParsePropertyCoordinates(t *testing.T) {
	raw := json.RawMessage(`{
		"ListingKey": "ABC",
		"StandardStatus": "Closed",
		"ClosePrice": 450000,
		"BedroomsTotal": 3,
		"LivingArea": 1800,
		"Latitude": 27.95,
		"Longitude": -82.46
	}`)
	c := parseProperty(raw)
	if c.ListingKey != "ABC" || c.ClosePrice != 450000 {
		t.Fatalf("parse: %+v", c)
	}
	if c.Lat != 27.95 || c.Lng != -82.46 {
		t.Fatalf("coords: %f %f", c.Lat, c.Lng)
	}
}

func TestHaversineMiles(t *testing.T) {
	d := haversineMiles(27.95, -82.46, 27.96, -82.47)
	if d <= 0 || d > 2 {
		t.Fatalf("distance %f", d)
	}
}
