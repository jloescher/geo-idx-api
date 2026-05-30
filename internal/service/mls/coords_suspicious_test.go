package mls

import "testing"

func TestIsSuspiciousCoordinate(t *testing.T) {
	tests := []struct {
		name  string
		lat   float64
		lng   float64
		state string
		want  bool
	}{
		{"Antarctica", -75, 0, "FL", true},
		{"Null Island", 0, 0, "FL", true},
		{"valid Tampa", 27.95, -82.45, "FL", false},
		{"valid Miami", 25.76, -80.19, "FL", false},
		{"swapped FL lat lng", -82, 27, "FL", true},
		{"outside FL bbox north", 32.5, -82, "FL", true},
		{"outside FL bbox west", 27, -88.5, "FL", true},
		{"non FL state ignored bbox", 40.7, -74.0, "NY", false},
		{"non FL Antarctica still flagged", -70, 10, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSuspiciousCoordinate(tt.lat, tt.lng, tt.state); got != tt.want {
				t.Fatalf("IsSuspiciousCoordinate(%v, %v, %q) = %v, want %v", tt.lat, tt.lng, tt.state, got, tt.want)
			}
		})
	}
}

func TestNormalizeFLCoordinates(t *testing.T) {
	tests := []struct {
		name      string
		lat, lng  float64
		state     string
		wantLat   float64
		wantLng   float64
		corrected bool
	}{
		{"swapped Tampa", -82, 27, "FL", 27, -82, true},
		{"valid Tampa unchanged", 27.95, -82.45, "FL", 27.95, -82.45, false},
		{"non-FL unchanged", 40.7, -74.0, "NY", 40.7, -74.0, false},
		{"outside bbox swap", -87, 28, "FL", 28, -87, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLat, gotLng, corrected := NormalizeFLCoordinates(tt.lat, tt.lng, tt.state)
			if gotLat != tt.wantLat || gotLng != tt.wantLng || corrected != tt.corrected {
				t.Fatalf("NormalizeFLCoordinates(%v, %v, %q) = (%v, %v, %v), want (%v, %v, %v)",
					tt.lat, tt.lng, tt.state, gotLat, gotLng, corrected, tt.wantLat, tt.wantLng, tt.corrected)
			}
		})
	}
}

func TestIsOutsideFLNFHLCoverage(t *testing.T) {
	tests := []struct {
		name  string
		lat   float64
		lng   float64
		state string
		want  bool
	}{
		{"FL state in bbox", 27.95, -82.45, "FL", false},
		{"PR state", 18.2, -66.5, "PR", true},
		{"null state international", 18.44, -64.61, "", true},
		{"null state in FL bbox", 27.95, -82.45, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsOutsideFLNFHLCoverage(tt.lat, tt.lng, tt.state); got != tt.want {
				t.Fatalf("IsOutsideFLNFHLCoverage(%v, %v, %q) = %v, want %v", tt.lat, tt.lng, tt.state, got, tt.want)
			}
		})
	}
}
