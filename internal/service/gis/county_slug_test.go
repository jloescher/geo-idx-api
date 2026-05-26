package gis

import "testing"

func TestCountyNameToSlug(t *testing.T) {
	cases := map[string]string{
		"Pinellas County": "pinellas",
		"St. Lucie":       "st-lucie",
		"Miami-Dade":      "miami-dade",
		"Palm Beach":      "palm-beach",
	}
	for in, want := range cases {
		if got := CountyNameToSlug(in); got != want {
			t.Fatalf("%q => %q, want %q", in, got, want)
		}
	}
}
