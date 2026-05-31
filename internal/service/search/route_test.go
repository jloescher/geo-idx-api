package search

import (
	"strings"
	"testing"
)

func TestDecideRoutePostgresOnlyDefault(t *testing.T) {
	if DecideRoute(SearchRequest{}) != RoutePostgresOnly {
		t.Fatal("expected mirror-only default")
	}
}

func TestDecideRouteUpstreamClosed(t *testing.T) {
	req := SearchRequest{Statuses: []string{"Closed"}}
	if DecideRoute(req) != RouteUpstreamOnly {
		t.Fatal("closed-only should use live upstream")
	}
}

func TestDecideRouteSplit(t *testing.T) {
	req := SearchRequest{Statuses: []string{"Active", "Closed"}}
	if DecideRoute(req) != RouteSplit {
		t.Fatal("mixed statuses should split")
	}
}

func TestBuildODataFilterStatuses(t *testing.T) {
	req := SearchRequest{Statuses: []string{"Active", "Closed"}}
	f := buildODataFilter(req, "stellar")
	if f == "" || !strings.Contains(f, "StandardStatus") {
		t.Fatalf("filter %q", f)
	}
	if !strings.Contains(f, "InternetEntireListingDisplayYN") {
		t.Fatalf("expected compliance filter in %q", f)
	}
}

// TestBuildODataFilter_AllFields asserts that every SearchRequest field with a direct OData
// equivalent is represented in the filter output. This guards against SearchRequest additions
// that silently skip the upstream leg.
func TestBuildODataFilter_AllFields(t *testing.T) {
	minPrice := 100000.0
	maxPrice := 500000.0
	bedsMin := 2
	bedsMax := 5
	bathsMin := 1.5
	livingMin := 1000
	livingMax := 3000
	lotMin := 0.25
	yearMin := 2000
	propType := "Residential"
	propSubType := "Single Family Residence"
	city := "Largo"
	county := "Pinellas"
	postal := "33770"
	pool := true
	water := true
	remarks := "waterfront view"
	priceReduced := 30

	req := SearchRequest{
		Statuses:               []string{"Closed"},
		MinPrice:               &minPrice,
		MaxPrice:               &maxPrice,
		BedsMin:                &bedsMin,
		BedsMax:                &bedsMax,
		BathsMin:               &bathsMin,
		LivingAreaMin:          &livingMin,
		LivingAreaMax:          &livingMax,
		LotSizeAcresMin:        &lotMin,
		YearBuiltMin:           &yearMin,
		PropertyType:           &propType,
		PropertySubType:        &propSubType,
		City:                   &city,
		CountyOrParish:         &county,
		PostalCode:             &postal,
		PoolPrivate:            &pool,
		Waterfront:             &water,
		RemarksQuery:           &remarks,
		PriceReducedWithinDays: &priceReduced,
	}

	cases := []struct {
		dataset string
		checks  []string
	}{
		{
			dataset: "stellar",
			checks: []string{
				"InternetEntireListingDisplayYN ne false",
				"IDXParticipationYN",
				"StandardStatus in ('Closed')",
				"ListPrice ge 100000",
				"ListPrice le 500000",
				"BedroomsTotal ge 2",
				"BedroomsTotal le 5",
				"BathroomsTotal ge 1.5",
				"LivingArea ge 1000",
				"LivingArea le 3000",
				"LotSizeAcres ge 0.25",
				"YearBuilt ge 2000",
				"PropertyType eq 'Residential'",
				"PropertySubType eq 'Single Family Residence'",
				"City eq 'Largo'",
				"CountyOrParish eq 'Pinellas'",
				"PostalCode eq '33770'",
				"PoolPrivateYN eq true",
				"WaterfrontYN eq true",
				"contains(PublicRemarks,'waterfront view')",
				"PriceChangeTimestamp gt",
			},
		},
		{
			dataset: "beaches",
			checks: []string{
				"InternetEntireListingDisplayYN ne false",
				"ListPrice ge 100000",
				"BedroomsTotal ge 2",
				"BathroomsTotal ge 1.5",
				"LivingArea ge 1000",
				"LotSizeAcres ge 0.25",
				"YearBuilt ge 2000",
				"PropertyType eq 'Residential'",
				"City eq 'Largo'",
				"PostalCode eq '33770'",
				"PoolPrivateYN eq true",
				"WaterfrontYN eq true",
				"contains(PublicRemarks,'waterfront view')",
				"PriceChangeTimestamp gt",
			},
		},
	}

	for _, tc := range cases {
		f := buildODataFilter(req, tc.dataset)
		for _, want := range tc.checks {
			if !strings.Contains(f, want) {
				t.Errorf("dataset=%s: filter missing %q\nfull filter: %s", tc.dataset, want, f)
			}
		}
	}
}

// TestBuildODataFilter_StellarIDXGate asserts the IDXParticipationYN gate is stellar-only.
func TestBuildODataFilter_StellarIDXGate(t *testing.T) {
	req := SearchRequest{}
	stellar := buildODataFilter(req, "stellar")
	beaches := buildODataFilter(req, "beaches")
	if !strings.Contains(stellar, "IDXParticipationYN") {
		t.Errorf("stellar filter missing IDXParticipationYN: %s", stellar)
	}
	if strings.Contains(beaches, "IDXParticipationYN") {
		t.Errorf("beaches filter should not include IDXParticipationYN: %s", beaches)
	}
}

// TestBuildODataFilter_ODataEscape asserts single quotes in string values are escaped.
func TestBuildODataFilter_ODataEscape(t *testing.T) {
	city := "O'Brien"
	req := SearchRequest{City: &city}
	f := buildODataFilter(req, "stellar")
	if !strings.Contains(f, "City eq 'O''Brien'") {
		t.Errorf("single quote not escaped in OData filter: %s", f)
	}
}

// TestBuildODataFilter_MirrorOnlyAbsent confirms mirror-only fields produce no OData clause.
func TestBuildODataFilter_MirrorOnlyAbsent(t *testing.T) {
	lowRisk := true
	minFees := 100.0
	lat := 27.9
	lng := -82.7
	radius := 5.0
	req := SearchRequest{
		LowRiskFloodzone: &lowRisk,
		MinMonthlyFees:   &minFees,
		Lat:              &lat,
		Lng:              &lng,
		RadiusMiles:      &radius,
	}
	f := buildODataFilter(req, "stellar")
	for _, absent := range []string{"low_risk", "flood", "monthly", "geo.distance", "LivingArea", "EstimatedTotal"} {
		if strings.Contains(strings.ToLower(f), strings.ToLower(absent)) {
			t.Errorf("mirror-only field leaked into OData filter (%q found): %s", absent, f)
		}
	}
}

// TestBuildODataFilter_PriceReducedUsesTimestamp asserts DaysOnMarket is no longer used
// for PriceReducedWithinDays (semantic fix: DOM counts listing age, not price-change age).
func TestBuildODataFilter_PriceReducedUsesTimestamp(t *testing.T) {
	days := 14
	req := SearchRequest{PriceReducedWithinDays: &days}
	f := buildODataFilter(req, "stellar")
	if strings.Contains(f, "DaysOnMarket") {
		t.Errorf("PriceReducedWithinDays must not map to DaysOnMarket: %s", f)
	}
	if !strings.Contains(f, "PriceChangeTimestamp gt") {
		t.Errorf("PriceReducedWithinDays must use PriceChangeTimestamp: %s", f)
	}
}

// TestSearchSplit_FilterParity verifies that RouteSplit is resolved for mixed-status requests
// and that the OData filter for the Closed leg contains the same portable field constraints
// as would be sent by any upstream call with equivalent parameters.
func TestSearchSplit_FilterParity(t *testing.T) {
	minPrice := 200000.0
	bedsMin := 3
	bathsMin := 2.0
	propType := "Residential"
	city := "Clearwater"
	postal := "33755"

	req := SearchRequest{
		Statuses:     []string{"Active", "Closed"},
		MinPrice:     &minPrice,
		BedsMin:      &bedsMin,
		BathsMin:     &bathsMin,
		PropertyType: &propType,
		City:         &city,
		PostalCode:   &postal,
	}

	if DecideRoute(req) != RouteSplit {
		t.Fatal("expected RouteSplit for Active+Closed request")
	}

	// Build the Closed leg filter (what LiveSearch will send).
	closedReq := req
	closedReq.Statuses = []string{"Closed"}
	f := buildODataFilter(closedReq, "stellar")

	// Every portable field must appear in the Closed leg OData filter.
	for _, want := range []string{
		"ListPrice ge 200000",
		"BedroomsTotal ge 3",
		"BathroomsTotal ge 2",
		"PropertyType eq 'Residential'",
		"City eq 'Clearwater'",
		"PostalCode eq '33755'",
		"StandardStatus in ('Closed')",
	} {
		if !strings.Contains(f, want) {
			t.Errorf("Closed leg OData filter missing portable field %q\nfull filter: %s", want, f)
		}
	}
}
