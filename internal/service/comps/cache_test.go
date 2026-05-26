package comps

import (
	"encoding/json"
	"testing"
	compscache "github.com/quantyralabs/idx-api/internal/repository/comps_cache"
)

func TestParseCloseDate(t *testing.T) {
	if parseCloseDate("") != nil {
		t.Fatal("empty")
	}
	tm := parseCloseDate("2025-01-15")
	if tm == nil || tm.UTC().Format("2006-01-02") != "2025-01-15" {
		t.Fatalf("date %v", tm)
	}
}

func TestCompsFromCached(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"ListingKey":     "C1",
		"StandardStatus": "Closed",
		"ClosePrice":     400000,
		"CloseDate":      "2025-01-01",
		"Latitude":       27.95,
		"Longitude":      -82.46,
		"LivingArea":     1800,
	})
	compressed, err := compscache.GzipForTest(raw)
	if err != nil {
		t.Fatal(err)
	}
	subject := SubjectProfile{Lat: 27.95, Lng: -82.45}
	radius := 5.0
	scope := ScopeInput{Type: "radius", RadiusMiles: &radius}
	out := compsFromCached([]compscache.CachedClosedRow{{
		ListingKey:        "C1",
		CompressedPayload: compressed,
		Latitude:          27.95,
		Longitude:         -82.46,
	}}, subject, scope)
	if len(out) != 1 || out[0].ClosePrice != 400000 {
		t.Fatalf("got %+v", out)
	}
}
