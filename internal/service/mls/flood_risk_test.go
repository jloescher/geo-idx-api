package mls_test

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func ptr(s string) *string { return &s }

func TestComputeLowRiskFloodZoneYN(t *testing.T) {
	tests := []struct {
		name string
		code *string
		want bool
	}{
		{"nil", nil, false},
		{"empty", ptr(""), false},
		{"whitespace", ptr("   "), false},
		{"X", ptr("X"), true},
		{"lowercase x", ptr("x"), true},
		{"X500", ptr("X500"), true},
		{"x500", ptr("x500"), true},
		{"NO", ptr("NO"), true},
		{"No Flood", ptr("No Flood"), true},
		{"AE", ptr("AE"), false},
		{"VE", ptr("VE"), false},
		{"ZONE A", ptr("ZONE A"), false},
		{"unrelated", ptr("B"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mls.ComputeLowRiskFloodZoneYN(tc.code)
			if got != tc.want {
				t.Fatalf("ComputeLowRiskFloodZoneYN(%v) = %v, want %v", tc.code, got, tc.want)
			}
		})
	}
}
