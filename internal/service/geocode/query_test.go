package geocode

import "testing"

func TestBuildGeocodeQuery_BeachesComplete(t *testing.T) {
	unparsed := "8746 Waterstone Boulevard, Fort Pierce, FL 34951"
	q, ok := BuildGeocodeQuery("beaches", unparsed, str("Fort Pierce"), str("FL"), str("34951"), nil, nil)
	if !ok {
		t.Fatal("expected ok")
	}
	if q != unparsed {
		t.Fatalf("query = %q, want full unparsed", q)
	}
}

func TestBuildGeocodeQuery_StellarStreetPlusTyped(t *testing.T) {
	q, ok := BuildGeocodeQuery("stellar", "353 N Temple AVENUE", str("Tampa"), str("FL"), str("33602"), nil, nil)
	if !ok {
		t.Fatal("expected ok")
	}
	want := "353 N Temple AVENUE, Tampa, FL 33602"
	if q != want {
		t.Fatalf("query = %q, want %q", q, want)
	}
}

func TestBuildGeocodeQuery_StreetNumberNameFallback(t *testing.T) {
	q, ok := BuildGeocodeQuery("stellar", "", str("Orlando"), str("FL"), str("32801"), str("100"), str("Main St"))
	if !ok {
		t.Fatal("expected ok")
	}
	if q != "100 Main St, Orlando, FL 32801" {
		t.Fatalf("query = %q", q)
	}
}

func str(s string) *string { return &s }
