package mls

import (
	"math"
	"strings"
)

// Florida service area bounds for Stellar and Beaches MLS coverage.
const (
	flLatMin = 24.5
	flLatMax = 31.2
	flLngMin = -87.8
	flLngMax = -79.8
)

// IsSuspiciousCoordinate reports implausible lat/lng for FEMA/geocode recovery.
// Used when NFHL misses on first pass to detect bad MLS coordinates.
func IsSuspiciousCoordinate(lat, lng float64, stateOrProvince string) bool {
	if lat < -60 {
		return true
	}
	if lat == 0 && lng == 0 {
		return true
	}
	state := strings.ToUpper(strings.TrimSpace(stateOrProvince))
	if state != "FL" {
		return false
	}
	if math.Abs(lat) > 50 && math.Abs(lng) < 45 {
		return true
	}
	if lat < flLatMin || lat > flLatMax || lng < flLngMin || lng > flLngMax {
		return true
	}
	return false
}
