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
