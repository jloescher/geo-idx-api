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

// IsInsideFLBBox reports whether lat/lng fall within Stellar/Beaches FL service bounds.
func IsInsideFLBBox(lat, lng float64) bool {
	return lat >= flLatMin && lat <= flLatMax && lng >= flLngMin && lng <= flLngMax
}

// IsOutsideFLNFHLCoverage reports listings that should skip US NFHL point queries.
func IsOutsideFLNFHLCoverage(lat, lng float64, stateOrProvince string) bool {
	state := strings.ToUpper(strings.TrimSpace(stateOrProvince))
	if state == "FL" {
		return false
	}
	if state != "" {
		return true
	}
	return !IsInsideFLBBox(lat, lng)
}

// NormalizeFLCoordinates corrects swapped or out-of-bbox FL listing coordinates at sync ingest.
func NormalizeFLCoordinates(lat, lng float64, stateOrProvince string) (float64, float64, bool) {
	state := strings.ToUpper(strings.TrimSpace(stateOrProvince))
	if state != "FL" {
		return lat, lng, false
	}
	if IsInsideFLBBox(lat, lng) {
		return lat, lng, false
	}
	swapHeuristic := math.Abs(lat) > 50 && math.Abs(lng) < 45
	if swapHeuristic || IsInsideFLBBox(lng, lat) {
		return lng, lat, true
	}
	return lat, lng, false
}

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
