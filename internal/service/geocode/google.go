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

// Geocode returns WGS84 lat/lng for a free-form address (Google Geocoding API).
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
		return 0, 0, fmt.Errorf("geocoding failed: %s", payload.Status)
	}
	loc := payload.Results[0].Geometry.Location
	return loc.Lat, loc.Lng, nil
}
