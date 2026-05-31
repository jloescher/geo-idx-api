package geocode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrGeocodeStatus is returned when the Google Geocoding API responds with a
// non-OK status. Callers can inspect Status to distinguish permanent failures
// (ZERO_RESULTS, INVALID_REQUEST) from transient ones (UNKNOWN_ERROR) and
// quota exhaustion (OVER_QUERY_LIMIT, OVER_DAILY_LIMIT).
type ErrGeocodeStatus struct {
	Status string
}

func (e ErrGeocodeStatus) Error() string {
	return "geocoding API status: " + e.Status
}

// Geocoder resolves street addresses to coordinates.
type Geocoder struct {
	apiKey string
	http   *http.Client
}

func NewGeocoder(apiKey string, timeout time.Duration) *Geocoder {
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	return &Geocoder{
		apiKey: strings.TrimSpace(apiKey),
		http:   &http.Client{Timeout: timeout},
	}
}

// IsConfigured reports whether an API key has been provided.
func (g *Geocoder) IsConfigured() bool {
	return g.apiKey != ""
}

// Geocode returns WGS84 lat/lng for a free-form address (Google Geocoding API).
// Non-OK API statuses are returned as ErrGeocodeStatus so callers can branch
// without string matching.
func (g *Geocoder) Geocode(ctx context.Context, address string) (lat, lng float64, err error) {
	if g.apiKey == "" {
		return 0, 0, fmt.Errorf("GOOGLE_MAPS_GEOCODING_API_KEY is not configured")
	}
	address = strings.TrimSpace(address)
	if address == "" {
		return 0, 0, fmt.Errorf("address is required")
	}

	u, _ := url.Parse("https://maps.googleapis.com/maps/api/geocode/json")
	q := u.Query()
	q.Set("address", address)
	q.Set("key", g.apiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, 0, err
	}
	resp, err := g.http.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}
	if resp.StatusCode >= 400 {
		return 0, 0, fmt.Errorf("geocoding upstream status %d", resp.StatusCode)
	}

	var payload struct {
		Status  string `json:"status"`
		Results []struct {
			Geometry struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0, err
	}
	if payload.Status != "OK" || len(payload.Results) == 0 {
		return 0, 0, ErrGeocodeStatus{Status: payload.Status}
	}
	loc := payload.Results[0].Geometry.Location
	return loc.Lat, loc.Lng, nil
}
