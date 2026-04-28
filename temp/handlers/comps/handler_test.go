package comps

import (
	"math"
	"strings"
	"testing"
)

// --- Validation tests ---

func TestValidateRequest_MissingSubjectType(t *testing.T) {
	req := &RunCompsRequest{
		Mode:  "A",
		Scope: ScopeInput{Type: "radius"},
	}
	if err := validateRequest(req); err == nil {
		t.Fatal("expected error for missing subject.type")
	}
}

func TestValidateRequest_InvalidSubjectType(t *testing.T) {
	req := &RunCompsRequest{
		Subject: SubjectInput{Type: "invalid"},
		Mode:    "A",
		Scope:   ScopeInput{Type: "radius"},
	}
	if err := validateRequest(req); err == nil {
		t.Fatal("expected error for invalid subject.type")
	}
}

func TestValidateRequest_MLSMissingListingID(t *testing.T) {
	req := &RunCompsRequest{
		Subject: SubjectInput{Type: "mls"},
		Mode:    "A",
		Scope:   ScopeInput{Type: "radius"},
	}
	if err := validateRequest(req); err == nil {
		t.Fatal("expected error for missing listing_id with type=mls")
	}
}

func TestValidateRequest_OffMarketMissingCoords(t *testing.T) {
	req := &RunCompsRequest{
		Subject: SubjectInput{Type: "off_market"},
		Mode:    "A",
		Scope:   ScopeInput{Type: "radius"},
	}
	if err := validateRequest(req); err == nil {
		t.Fatal("expected error for missing lat/lng with type=off_market")
	}
}

func TestValidateRequest_InvalidMode(t *testing.T) {
	lat, lng := 27.95, -82.46
	req := &RunCompsRequest{
		Subject: SubjectInput{Type: "off_market", Lat: &lat, Lng: &lng},
		Mode:    "Z",
		Scope:   ScopeInput{Type: "radius"},
	}
	if err := validateRequest(req); err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestValidateRequest_InvalidScopeType(t *testing.T) {
	lat, lng := 27.95, -82.46
	req := &RunCompsRequest{
		Subject: SubjectInput{Type: "off_market", Lat: &lat, Lng: &lng},
		Mode:    "A",
		Scope:   ScopeInput{Type: "invalid"},
	}
	if err := validateRequest(req); err == nil {
		t.Fatal("expected error for invalid scope type")
	}
}

func TestValidateRequest_ValidMLS(t *testing.T) {
	req := &RunCompsRequest{
		Subject: SubjectInput{Type: "mls", ListingID: "MLS-123"},
		Mode:    "A",
		Scope:   ScopeInput{Type: "radius"},
	}
	if err := validateRequest(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRequest_ValidOffMarket(t *testing.T) {
	lat, lng := 27.95, -82.46
	req := &RunCompsRequest{
		Subject: SubjectInput{Type: "off_market", Lat: &lat, Lng: &lng},
		Mode:    "D",
		Scope:   ScopeInput{Type: "neighborhood"},
	}
	if err := validateRequest(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRequest_AllModes(t *testing.T) {
	lat, lng := 27.95, -82.46
	for _, mode := range []string{"A", "B", "C", "D", "E"} {
		req := &RunCompsRequest{
			Subject: SubjectInput{Type: "off_market", Lat: &lat, Lng: &lng},
			Mode:    mode,
			Scope:   ScopeInput{Type: "radius"},
		}
		if err := validateRequest(req); err != nil {
			t.Fatalf("mode %s: unexpected error: %v", mode, err)
		}
	}
}

// --- Filter defaults ---

func TestFilterDefaults(t *testing.T) {
	f := &FiltersInput{}

	if f.livingAreaPct() != 10 {
		t.Errorf("livingAreaPct default: got %d, want 10", f.livingAreaPct())
	}
	if f.bedsTolerance() != 1 {
		t.Errorf("bedsTolerance default: got %d, want 1", f.bedsTolerance())
	}
	if f.bathsTolerance() != 1 {
		t.Errorf("bathsTolerance default: got %d, want 1", f.bathsTolerance())
	}
	if f.matchPool() != true {
		t.Error("matchPool default: got false, want true")
	}
	if f.soldMonthsBack() != 6 {
		t.Errorf("soldMonthsBack default: got %d, want 6", f.soldMonthsBack())
	}
	if f.maxSoldComps() != 12 {
		t.Errorf("maxSoldComps default: got %d, want 12", f.maxSoldComps())
	}
	if f.maxCompetitionComps() != 20 {
		t.Errorf("maxCompetitionComps default: got %d, want 20", f.maxCompetitionComps())
	}
	if f.maxFailedListings() != 20 {
		t.Errorf("maxFailedListings default: got %d, want 20", f.maxFailedListings())
	}
	if f.includeActivePending() != true {
		t.Error("includeActivePending default: got false, want true")
	}
	if f.includeOverpricedSignals() != true {
		t.Error("includeOverpricedSignals default: got false, want true")
	}
	if f.includeFailedListings() != true {
		t.Error("includeFailedListings default: got false, want true")
	}
}

func TestFilterOverrides(t *testing.T) {
	pct := 20
	beds := 2
	sold := 12
	max := 5
	falseVal := false

	f := &FiltersInput{
		LivingAreaPct:        &pct,
		BedsTolerance:        &beds,
		SoldMonthsBack:       &sold,
		MaxSoldComps:         &max,
		IncludeActivePending: &falseVal,
	}

	if f.livingAreaPct() != 20 {
		t.Errorf("got %d, want 20", f.livingAreaPct())
	}
	if f.bedsTolerance() != 2 {
		t.Errorf("got %d, want 2", f.bedsTolerance())
	}
	if f.soldMonthsBack() != 12 {
		t.Errorf("got %d, want 12", f.soldMonthsBack())
	}
	if f.maxSoldComps() != 5 {
		t.Errorf("got %d, want 5", f.maxSoldComps())
	}
	if f.includeActivePending() != false {
		t.Error("got true, want false")
	}
}

// --- Keyword scoring tests ---

func TestComputeKeywordScores_ModeDE(t *testing.T) {
	keywords := map[string][]Keyword{
		"disrepair":      {{Phrase: "as-is", Weight: 1.0}},
		"good_condition": {{Phrase: "renovated", Weight: 1.0}},
	}
	for _, mode := range []string{"D", "E"} {
		result := computeKeywordScores("This is an as-is renovated property", keywords, mode)
		if result.DisrepairScore != 0.5 {
			t.Errorf("mode %s: disrepair_score got %f, want 0.5", mode, result.DisrepairScore)
		}
		if result.GoodConditionScore != 0.5 {
			t.Errorf("mode %s: good_condition_score got %f, want 0.5", mode, result.GoodConditionScore)
		}
		if len(result.DisrepairMatches) != 0 {
			t.Errorf("mode %s: expected no disrepair matches", mode)
		}
	}
}

func TestComputeKeywordScores_DisrepairMatch(t *testing.T) {
	keywords := map[string][]Keyword{
		"disrepair": {
			{Phrase: "as-is", Weight: 1.0},
			{Phrase: "investor special", Weight: 1.0},
		},
	}
	result := computeKeywordScores("Sold as-is, investor special!", keywords, "A")
	if result.DisrepairScore != 1.0 {
		t.Errorf("got %f, want 1.0", result.DisrepairScore)
	}
	if len(result.DisrepairMatches) != 2 {
		t.Errorf("got %d matches, want 2", len(result.DisrepairMatches))
	}
}

func TestComputeKeywordScores_PartialMatch(t *testing.T) {
	keywords := map[string][]Keyword{
		"good_condition": {
			{Phrase: "renovated", Weight: 1.0},
			{Phrase: "new roof", Weight: 0.5},
			{Phrase: "granite", Weight: 0.5},
		},
	}
	result := computeKeywordScores("Beautifully renovated home with granite counters", keywords, "B")
	// matched weight = 1.0 + 0.5 = 1.5, total weight = 2.0
	expected := 1.5 / 2.0
	if math.Abs(result.GoodConditionScore-expected) > 0.001 {
		t.Errorf("got %f, want %f", result.GoodConditionScore, expected)
	}
}

func TestComputeKeywordScores_RegexMatch(t *testing.T) {
	keywords := map[string][]Keyword{
		"disrepair": {
			{Phrase: "(?i)as.is", Weight: 1.0, IsRegex: true},
		},
	}
	result := computeKeywordScores("Property sold AS IS condition", keywords, "A")
	if result.DisrepairScore != 1.0 {
		t.Errorf("got %f, want 1.0", result.DisrepairScore)
	}
}

func TestComputeKeywordScores_EmptyRemarks(t *testing.T) {
	keywords := map[string][]Keyword{
		"disrepair": {{Phrase: "as-is", Weight: 1.0}},
	}
	result := computeKeywordScores("", keywords, "A")
	if result.DisrepairScore != 0 {
		t.Errorf("got %f, want 0", result.DisrepairScore)
	}
}

func TestComputeKeywordScores_NoKeywords(t *testing.T) {
	result := computeKeywordScores("some text", nil, "A")
	if result.DisrepairScore != 0 {
		t.Errorf("got %f, want 0", result.DisrepairScore)
	}
}

// --- Scope tests ---

func TestBuildScopeClause_Radius(t *testing.T) {
	miles := 1.5
	subject := &resolvedSubject{Lat: 27.95, Lng: -82.46}
	scope := ScopeInput{Type: "radius", RadiusMiles: &miles}

	result, err := buildScopeClause(subject, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.clause == "" {
		t.Fatal("expected non-empty clause")
	}
	if len(result.args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(result.args))
	}
	// args: lng, lat, radius_meters
	if result.args[0] != -82.46 {
		t.Errorf("lng: got %v, want -82.46", result.args[0])
	}
	if result.args[1] != 27.95 {
		t.Errorf("lat: got %v, want 27.95", result.args[1])
	}
	expectedMeters := 1.5 * 1609.344
	if result.args[2] != expectedMeters {
		t.Errorf("radius_meters: got %v, want %v", result.args[2], expectedMeters)
	}
}

func TestBuildScopeClause_Neighborhood(t *testing.T) {
	subdivID := int64(42)
	subject := &resolvedSubject{}
	scope := ScopeInput{Type: "neighborhood", SubdivisionRefID: &subdivID}

	result, err := buildScopeClause(subject, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(result.args))
	}
	if result.args[0] != int64(42) {
		t.Errorf("got %v, want 42", result.args[0])
	}
}

func TestBuildScopeClause_Zip(t *testing.T) {
	subject := &resolvedSubject{}
	scope := ScopeInput{Type: "zip", PostalCodes: []string{"33601", "33602"}}

	result, err := buildScopeClause(subject, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(result.args))
	}
}

func TestBuildScopeClause_MissingRequired(t *testing.T) {
	subject := &resolvedSubject{}

	// Neighborhood without subdivision_ref_id
	_, err := buildScopeClause(subject, ScopeInput{Type: "neighborhood"})
	if err == nil {
		t.Fatal("expected error for missing subdivision_ref_id")
	}

	// Zip without postal_codes
	_, err = buildScopeClause(subject, ScopeInput{Type: "zip"})
	if err == nil {
		t.Fatal("expected error for missing postal_codes")
	}

	// Polygon without geojson
	_, err = buildScopeClause(subject, ScopeInput{Type: "polygon"})
	if err == nil {
		t.Fatal("expected error for missing polygon_geojson")
	}
}

func TestBuildScopeClause_RadiusExplicitCenter(t *testing.T) {
	miles := 1.0
	cLat, cLng := 28.0, -82.0
	subject := &resolvedSubject{Lat: 27.95, Lng: -82.46}
	scope := ScopeInput{Type: "radius", RadiusMiles: &miles, CenterLat: &cLat, CenterLng: &cLng}

	result, err := buildScopeClause(subject, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should use explicit center, not subject
	if result.args[0] != -82.0 {
		t.Errorf("lng: got %v, want -82.0", result.args[0])
	}
	if result.args[1] != 28.0 {
		t.Errorf("lat: got %v, want 28.0", result.args[1])
	}
}

// --- Overpriced signals tests ---

func TestComputeOverpricedSignals_DOMAboveMedian(t *testing.T) {
	dom10, dom20, dom15 := 10, 20, 15
	dom50 := 50

	soldComps := []SoldComp{
		{DOM: &dom10},
		{DOM: &dom20},
		{DOM: &dom15},
	}
	competitionComps := []CompetitionComp{
		{ListingID: "COMP-1", DOM: &dom50},
	}

	signals := computeOverpricedSignals(soldComps, competitionComps, nil, AggMedian)

	found := false
	for _, s := range signals {
		if s.Indicator == "dom_above_median" && s.ListingID == "COMP-1" {
			found = true
		}
	}
	if !found {
		t.Error("expected dom_above_median signal for COMP-1")
	}
}

func TestComputeOverpricedSignals_ExpiredSimilar(t *testing.T) {
	price := 350000.0
	dom := 180
	failedListings := []FailedListing{
		{
			ListingID:       "FAIL-1",
			StandardStatus:  "Expired",
			LastListPrice:   &price,
			DOM:             &dom,
			PriceReductions: 2,
			SimilarityScore: 85.1,
		},
	}

	signals := computeOverpricedSignals(nil, nil, failedListings, AggMedian)

	found := false
	for _, s := range signals {
		if s.Indicator == "expired_similar" && s.ListingID == "FAIL-1" {
			found = true
			if s.Severity != "high" {
				t.Errorf("severity: got %s, want high", s.Severity)
			}
		}
	}
	if !found {
		t.Error("expected expired_similar signal for FAIL-1")
	}
}

func TestComputeOverpricedSignals_Empty(t *testing.T) {
	signals := computeOverpricedSignals(nil, nil, nil, AggMedian)
	if len(signals) != 0 {
		t.Errorf("expected empty signals, got %d", len(signals))
	}
}

// --- Address formatting ---

func TestBuildAddress(t *testing.T) {
	num := "123"
	name := "Main"
	suf := "St"
	city := "Tampa"
	state := "FL"
	zip := "33601"

	addr := buildAddress(&num, nil, &name, &suf, nil, nil, &city, &state, &zip)
	want := "123 Main St, Tampa, FL 33601"
	if addr != want {
		t.Errorf("got %q, want %q", addr, want)
	}
}

func TestBuildAddress_WithUnit(t *testing.T) {
	num := "456"
	name := "Oak"
	suf := "Ave"
	unit := "B"
	city := "Tampa"
	state := "FL"
	zip := "33602"

	addr := buildAddress(&num, nil, &name, &suf, nil, &unit, &city, &state, &zip)
	want := "456 Oak Ave #B, Tampa, FL 33602"
	if addr != want {
		t.Errorf("got %q, want %q", addr, want)
	}
}

func TestBuildAddress_AllNil(t *testing.T) {
	addr := buildAddress(nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if addr != "" {
		t.Errorf("got %q, want empty", addr)
	}
}

// --- Placeholder renumbering ---

func TestRenumberPlaceholders(t *testing.T) {
	sql := "AND ST_DWithin(pc.location, ST_SetSRID(ST_MakePoint($SCOPE1, $SCOPE2), 4326)::geography, $SCOPE3)"
	result, nextIdx := renumberPlaceholders(sql, 13)

	if nextIdx != 16 {
		t.Errorf("nextIdx: got %d, want 16", nextIdx)
	}
	expected := "AND ST_DWithin(pc.location, ST_SetSRID(ST_MakePoint($13, $14), 4326)::geography, $15)"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRenumberPlaceholders_WithFilters(t *testing.T) {
	sql := "AND pc.subdivision_ref_id = $SCOPE1\n  AND ABS(pc.living_area - $FILTER1) <= $FILTER2"
	result, nextIdx := renumberPlaceholders(sql, 13)

	if nextIdx != 16 {
		t.Errorf("nextIdx: got %d, want 16", nextIdx)
	}
	expected := "AND pc.subdivision_ref_id = $13\n  AND ABS(pc.living_area - $14) <= $15"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestRenumberPlaceholders_PrefixCollision(t *testing.T) {
	// Regression: $FILTER1 replacement must not corrupt $FILTER10+.
	sql := "AND ABS(pc.living_area - $FILTER1) <= $FILTER2" +
		"\n  AND pc.bedrooms_total = $FILTER3" +
		"\n  AND pc.bathrooms_total = $FILTER4" +
		"\n  AND pc.pool_private_yn = $FILTER5" +
		"\n  AND pc.association_yn = $FILTER6" +
		"\n  AND pc.low_risk_floodzone_yn = $FILTER7" +
		"\n  AND pc.waterfront_yn = $FILTER8" +
		"\n  AND pc.property_sub_type = $FILTER9" +
		"\n  AND ABS(pc.year_built - $FILTER10) <= $FILTER11" +
		"\n  AND ABS(pc.lot_size_acres - $FILTER12) <= $FILTER13"

	result, nextIdx := renumberPlaceholders(sql, 18)

	if nextIdx != 31 {
		t.Errorf("nextIdx: got %d, want 31", nextIdx)
	}
	// $FILTER10 must become $27, not "$180" (prefix corruption of $FILTER1→$18).
	expected := "AND ABS(pc.living_area - $18) <= $19" +
		"\n  AND pc.bedrooms_total = $20" +
		"\n  AND pc.bathrooms_total = $21" +
		"\n  AND pc.pool_private_yn = $22" +
		"\n  AND pc.association_yn = $23" +
		"\n  AND pc.low_risk_floodzone_yn = $24" +
		"\n  AND pc.waterfront_yn = $25" +
		"\n  AND pc.property_sub_type = $26" +
		"\n  AND ABS(pc.year_built - $27) <= $28" +
		"\n  AND ABS(pc.lot_size_acres - $29) <= $30"
	if result != expected {
		t.Errorf("prefix collision:\n got %q\nwant %q", result, expected)
	}
}

// --- Median helper ---

func TestMedian(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
	}{
		{"empty", nil, 0},
		{"single", []float64{5}, 5},
		{"odd", []float64{3, 1, 2}, 2},
		{"even", []float64{4, 2, 1, 3}, 2.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := median(tt.values)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("got %f, want %f", got, tt.want)
			}
		})
	}
}

// --- Off-market subject resolution ---

func TestResolveOffMarketSubject(t *testing.T) {
	lat, lng := 27.95, -82.46
	beds := 3
	sqft := 1800
	lotSqft := 7500
	pool := true
	price := 180000.0

	input := SubjectInput{
		Type:           "off_market",
		Lat:            &lat,
		Lng:            &lng,
		Bedrooms:       &beds,
		LivingAreaSqft: &sqft,
		LotSizeSqft:    &lotSqft,
		Pool:           &pool,
		AskingPrice:    &price,
	}

	s := resolveOffMarketSubject(input)

	if s.Lat != 27.95 {
		t.Errorf("Lat: got %f, want 27.95", s.Lat)
	}
	if s.Lng != -82.46 {
		t.Errorf("Lng: got %f, want -82.46", s.Lng)
	}
	if *s.Bedrooms != 3 {
		t.Errorf("Bedrooms: got %d, want 3", *s.Bedrooms)
	}
	if *s.LivingAreaSqft != 1800 {
		t.Errorf("LivingAreaSqft: got %d, want 1800", *s.LivingAreaSqft)
	}
	if *s.LotSizeSqft != 7500 {
		t.Errorf("LotSizeSqft: got %d, want 7500", *s.LotSizeSqft)
	}
	if s.LotSizeAcres == nil {
		t.Fatal("LotSizeAcres should not be nil")
	}
	expectedAcres := 7500.0 / 43560.0
	if math.Abs(*s.LotSizeAcres-expectedAcres) > 0.001 {
		t.Errorf("LotSizeAcres: got %f, want %f", *s.LotSizeAcres, expectedAcres)
	}
	if *s.Pool != true {
		t.Error("Pool: got false, want true")
	}
	if *s.ListPrice != 180000.0 {
		t.Errorf("ListPrice: got %f, want 180000", *s.ListPrice)
	}
}

func TestResolveOffMarketSubject_NewFields(t *testing.T) {
	lat, lng := 27.95, -82.46
	waterfront := true
	garageSpaces := 2

	input := SubjectInput{
		Type:         "off_market",
		Lat:          &lat,
		Lng:          &lng,
		Waterfront:   &waterfront,
		GarageSpaces: &garageSpaces,
	}

	s := resolveOffMarketSubject(input)

	if s.Waterfront == nil || *s.Waterfront != true {
		t.Error("Waterfront: expected true")
	}
	if s.GarageSpaces == nil || *s.GarageSpaces != 2 {
		t.Errorf("GarageSpaces: got %v, want 2", s.GarageSpaces)
	}
}

// --- Appraisal filter defaults ---

func TestFilterDefaults_AppraisalFilters(t *testing.T) {
	f := &FiltersInput{}

	if f.matchPropertySubType() != true {
		t.Error("matchPropertySubType default: got false, want true")
	}
	if f.matchWaterfront() != true {
		t.Error("matchWaterfront default: got false, want true")
	}
	if f.yearBuiltTolerance() != 15 {
		t.Errorf("yearBuiltTolerance default: got %d, want 15", f.yearBuiltTolerance())
	}
	if f.lotSizePct() != 50 {
		t.Errorf("lotSizePct default: got %d, want 50", f.lotSizePct())
	}
	if f.adjPoolValue() != 20000 {
		t.Errorf("adjPoolValue default: got %f, want 20000", f.adjPoolValue())
	}
	if f.adjGaragePerSpace() != 7500 {
		t.Errorf("adjGaragePerSpace default: got %f, want 7500", f.adjGaragePerSpace())
	}
	if f.adjWaterfrontValue() != 50000 {
		t.Errorf("adjWaterfrontValue default: got %f, want 50000", f.adjWaterfrontValue())
	}
	if f.adjYearBuiltPerYear() != 500 {
		t.Errorf("adjYearBuiltPerYear default: got %f, want 500", f.adjYearBuiltPerYear())
	}
	if f.adjLotPerAcre() != 25000 {
		t.Errorf("adjLotPerAcre default: got %f, want 25000", f.adjLotPerAcre())
	}
}

// --- Similarity args ---

func TestBuildSimilarityArgs_14Elements(t *testing.T) {
	sqft := 2000
	beds := 3
	baths := 2
	yearBuilt := 2010
	pool := true
	hoa := false
	flood := true
	lotAcres := 0.25
	propType := "Residential"
	waterfront := false
	garageSpaces := 2

	subject := &resolvedSubject{
		Lat: 27.95, Lng: -82.46,
		LivingAreaSqft: &sqft,
		Bedrooms:       &beds,
		Bathrooms:      &baths,
		YearBuilt:      &yearBuilt,
		Pool:           &pool,
		HOA:            &hoa,
		FloodZone:      &flood,
		LotSizeAcres:   &lotAcres,
		PropertyType:   &propType,
		Waterfront:     &waterfront,
		GarageSpaces:   &garageSpaces,
	}

	args := buildSimilarityArgs(subject, 16093.44)

	if len(args) != 14 {
		t.Fatalf("expected 14 args, got %d", len(args))
	}
	if args[0] != -82.46 {
		t.Errorf("$1 lng: got %v, want -82.46", args[0])
	}
	if args[1] != 27.95 {
		t.Errorf("$2 lat: got %v, want 27.95", args[1])
	}
	if args[3] != 2000 {
		t.Errorf("$4 living_area: got %v, want 2000", args[3])
	}
	if args[12] != false {
		t.Errorf("$13 waterfront: got %v, want false", args[12])
	}
	if args[13] != 2 {
		t.Errorf("$14 garage_spaces: got %v, want 2", args[13])
	}
}

// --- New filter clauses ---

func TestBuildFilterClauses_AppraisalFilters(t *testing.T) {
	sqft := 2000
	yearBuilt := 2010
	lotAcres := 0.25
	propSubType := "Single Family Residence"
	waterfront := true

	subject := &resolvedSubject{
		LivingAreaSqft:  &sqft,
		YearBuilt:       &yearBuilt,
		LotSizeAcres:    &lotAcres,
		PropertySubType: &propSubType,
		Waterfront:      &waterfront,
	}

	filters := &FiltersInput{}

	clause, args := buildFilterClauses(subject, filters)

	if !strings.Contains(clause, "property_sub_type") {
		t.Error("expected property_sub_type filter clause")
	}
	if !strings.Contains(clause, "waterfront_yn") {
		t.Error("expected waterfront_yn filter clause")
	}
	if !strings.Contains(clause, "year_built") {
		t.Error("expected year_built filter clause")
	}
	if !strings.Contains(clause, "lot_size_acres") {
		t.Error("expected lot_size_acres filter clause")
	}

	// living_area(2) + property_sub_type(1) + waterfront(1) + year_built(2) + lot_size(2) = 8
	if len(args) != 8 {
		t.Errorf("expected 8 args, got %d", len(args))
	}
}

// --- Adjustment grid ---

func TestComputeAdjustmentGrid(t *testing.T) {
	sqft := 2000
	pool := true
	garage := 2
	waterfront := false
	yearBuilt := 2010
	lotAcres := 0.25

	subject := &resolvedSubject{
		LivingAreaSqft: &sqft,
		Pool:           &pool,
		GarageSpaces:   &garage,
		Waterfront:     &waterfront,
		YearBuilt:      &yearBuilt,
		LotSizeAcres:   &lotAcres,
	}

	compSqft := 1800
	compPool := false
	compGarage := 1
	compWaterfront := false
	compYearBuilt := 2005
	compLotAcres := 0.20
	compClosePrice := 400000.0

	comp := compRow{
		LivingArea:    &compSqft,
		PoolPrivateYn: &compPool,
		GarageSpaces:  &compGarage,
		WaterfrontYn:  &compWaterfront,
		YearBuilt:     &compYearBuilt,
		LotSizeAcres:  &compLotAcres,
		ClosePrice:    &compClosePrice,
	}

	medianPPSF := 200.0
	filters := &FiltersInput{}

	grid := computeAdjustmentGrid(subject, comp, medianPPSF, filters)
	if grid == nil {
		t.Fatal("expected non-nil grid")
	}

	// GLA: (2000-1800) * 200 = 40000
	// Pool: subject has, comp doesn't = +20000
	// Garage: (2-1) * 7500 = 7500
	// Waterfront: both false, no adjustment
	// Year built: (2010-2005) * 500 = 2500
	// Lot size: (0.25-0.20) * 25000 = 1250
	expectedLines := map[string]float64{
		"gla":        40000,
		"pool":       20000,
		"garage":     7500,
		"year_built": 2500,
		"lot_size":   1250,
	}

	for _, line := range grid.Lines {
		expected, ok := expectedLines[line.Feature]
		if !ok {
			t.Errorf("unexpected feature: %s", line.Feature)
			continue
		}
		if math.Abs(line.Adjustment-expected) > 1 {
			t.Errorf("%s: got %f, want %f", line.Feature, line.Adjustment, expected)
		}
		if line.Reasoning == "" {
			t.Errorf("%s: reasoning should not be empty", line.Feature)
		}
		delete(expectedLines, line.Feature)
	}
	for feature := range expectedLines {
		t.Errorf("missing adjustment line: %s", feature)
	}

	// Spot-check reasoning text.
	for _, line := range grid.Lines {
		switch line.Feature {
		case "gla":
			if !strings.Contains(line.Reasoning, "200 sqft larger") {
				t.Errorf("gla reasoning missing size delta: %s", line.Reasoning)
			}
			if !strings.Contains(line.Reasoning, "median PPSF") {
				t.Errorf("gla reasoning missing PPSF reference: %s", line.Reasoning)
			}
		case "pool":
			if !strings.Contains(line.Reasoning, "Subject has pool") {
				t.Errorf("pool reasoning missing subject pool: %s", line.Reasoning)
			}
		case "garage":
			if !strings.Contains(line.Reasoning, "1 more garage") {
				t.Errorf("garage reasoning missing diff: %s", line.Reasoning)
			}
		case "year_built":
			if !strings.Contains(line.Reasoning, "5 years newer") {
				t.Errorf("year_built reasoning missing age diff: %s", line.Reasoning)
			}
		case "lot_size":
			if !strings.Contains(line.Reasoning, "0.05 acres larger") {
				t.Errorf("lot_size reasoning missing size diff: %s", line.Reasoning)
			}
		}
	}

	// Net: 40000 + 20000 + 7500 + 2500 + 1250 = 71250
	if math.Abs(grid.NetAdjustment-71250) > 1 {
		t.Errorf("NetAdjustment: got %f, want 71250", grid.NetAdjustment)
	}

	// Adjusted price: 400000 + 71250 = 471250
	if math.Abs(grid.AdjustedPrice-471250) > 1 {
		t.Errorf("AdjustedPrice: got %f, want 471250", grid.AdjustedPrice)
	}

	// Gross: 71250 / 400000 * 100 = 17.8125%
	if math.Abs(grid.GrossAdjPct-17.8) > 0.2 {
		t.Errorf("GrossAdjPct: got %f, want ~17.8", grid.GrossAdjPct)
	}

	if grid.HighAdjWarning {
		t.Error("HighAdjWarning should be false for 17.8%")
	}
}

func TestComputeAdjustmentGrid_HighAdjWarning(t *testing.T) {
	sqft := 3000
	subject := &resolvedSubject{
		LivingAreaSqft: &sqft,
	}

	compSqft := 1500
	compClosePrice := 300000.0

	comp := compRow{
		LivingArea: &compSqft,
		ClosePrice: &compClosePrice,
	}

	// GLA adj: (3000-1500)*200 = 300000, gross = 100%
	grid := computeAdjustmentGrid(subject, comp, 200.0, &FiltersInput{})
	if grid == nil {
		t.Fatal("expected non-nil grid")
	}
	if !grid.HighAdjWarning {
		t.Error("HighAdjWarning should be true for >25% gross adjustment")
	}
	for _, line := range grid.Lines {
		if line.Reasoning == "" {
			t.Errorf("%s: reasoning should not be empty", line.Feature)
		}
	}
}

func TestComputeAdjustmentGrid_NilClosePrice(t *testing.T) {
	sqft := 2000
	subject := &resolvedSubject{LivingAreaSqft: &sqft}
	comp := compRow{} // no close price

	grid := computeAdjustmentGrid(subject, comp, 200.0, &FiltersInput{})
	if grid != nil {
		t.Error("expected nil grid when close price is nil")
	}
}

// --- Market conditions ---

func TestComputeMarketConditions(t *testing.T) {
	closePrice1, closePrice2, closePrice3 := 300000.0, 400000.0, 350000.0
	sqft1, sqft2, sqft3 := 1500, 2000, 1750
	dom1, dom2, dom3 := 30, 60, 45
	origPrice1, origPrice2, origPrice3 := 310000.0, 420000.0, 360000.0

	rows := []compRow{
		{ClosePrice: &closePrice1, LivingArea: &sqft1, DaysOnMarket: &dom1, OriginalListPrice: &origPrice1},
		{ClosePrice: &closePrice2, LivingArea: &sqft2, DaysOnMarket: &dom2, OriginalListPrice: &origPrice2},
		{ClosePrice: &closePrice3, LivingArea: &sqft3, DaysOnMarket: &dom3, OriginalListPrice: &origPrice3},
	}

	mc := computeMarketConditions(rows, 50, 30, 6, AggMedian)

	if mc.MedianSoldPPSF == nil {
		t.Fatal("MedianSoldPPSF should not be nil")
	}
	// PPSFs: 200, 200, 200 -> median = 200
	if math.Abs(*mc.MedianSoldPPSF-200) > 1 {
		t.Errorf("MedianSoldPPSF: got %f, want 200", *mc.MedianSoldPPSF)
	}

	if mc.MedianSoldDOM == nil {
		t.Fatal("MedianSoldDOM should not be nil")
	}
	// DOMs: 30, 45, 60 -> median = 45
	if *mc.MedianSoldDOM != 45 {
		t.Errorf("MedianSoldDOM: got %d, want 45", *mc.MedianSoldDOM)
	}

	if mc.AvgListToSaleRatio == nil {
		t.Fatal("AvgListToSaleRatio should not be nil")
	}
	// Ratios: 300/310=0.968, 400/420=0.952, 350/360=0.972 -> avg ≈ 0.964
	if math.Abs(*mc.AvgListToSaleRatio-0.964) > 0.01 {
		t.Errorf("AvgListToSaleRatio: got %f, want ~0.964", *mc.AvgListToSaleRatio)
	}

	if mc.MonthsOfInventory == nil {
		t.Fatal("MonthsOfInventory should not be nil")
	}
	// MOI: 30 / (50/6) = 30 / 8.333 = 3.6
	if math.Abs(*mc.MonthsOfInventory-3.6) > 0.1 {
		t.Errorf("MonthsOfInventory: got %f, want 3.6", *mc.MonthsOfInventory)
	}
}

func TestComputeMarketConditions_Empty(t *testing.T) {
	mc := computeMarketConditions(nil, 0, 0, 6, AggMedian)

	if mc.MedianSoldPPSF != nil {
		t.Error("MedianSoldPPSF should be nil for empty rows")
	}
	if mc.MedianSoldDOM != nil {
		t.Error("MedianSoldDOM should be nil for empty rows")
	}
	if mc.AvgListToSaleRatio != nil {
		t.Error("AvgListToSaleRatio should be nil for empty rows")
	}
	if mc.MonthsOfInventory != nil {
		t.Error("MonthsOfInventory should be nil for empty rows")
	}
}

// --- formatDollarsSigned ---

func TestFormatDollarsSigned(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{40000, "+$40,000"},
		{-20000, "-$20,000"},
		{0, "+$0"},
		{7500, "+$7,500"},
		{-1250, "-$1,250"},
	}
	for _, tt := range tests {
		got := formatDollarsSigned(tt.input)
		if got != tt.want {
			t.Errorf("formatDollarsSigned(%f): got %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- Flood zone filter removed from SQL ---

func TestBuildFilterClauses_FloodRemovedFromSQL(t *testing.T) {
	flood := true
	subject := &resolvedSubject{FloodZone: &flood}
	filters := &FiltersInput{} // matchFlood defaults to true

	clause, args := buildFilterClauses(subject, filters)

	if strings.Contains(clause, "low_risk_floodzone_yn") {
		t.Error("flood zone boolean filter should no longer be generated in SQL (handled Go-side)")
	}
	// With only flood set, no other filters should produce args.
	if len(args) != 0 {
		t.Errorf("expected 0 args (flood removed), got %d", len(args))
	}
}

func TestBuildFilterClauses_SeniorCommunityFilter(t *testing.T) {
	senior := true
	subject := &resolvedSubject{SeniorCommunity: &senior}
	filters := &FiltersInput{} // matchSeniorCommunity defaults to true

	clause, args := buildFilterClauses(subject, filters)

	if !strings.Contains(clause, "senior_community_yn") {
		t.Error("expected senior_community_yn filter clause")
	}
	if len(args) != 1 {
		t.Errorf("expected 1 arg, got %d", len(args))
	}
	if args[0] != true {
		t.Errorf("expected arg to be true, got %v", args[0])
	}
}

func TestBuildFilterClauses_SeniorCommunityDisabled(t *testing.T) {
	senior := true
	falseVal := false
	subject := &resolvedSubject{SeniorCommunity: &senior}
	filters := &FiltersInput{MatchSeniorCommunity: &falseVal}

	clause, _ := buildFilterClauses(subject, filters)

	if strings.Contains(clause, "senior_community_yn") {
		t.Error("senior_community_yn should not appear when match_senior_community is false")
	}
}

func TestResolveOffMarketSubject_FloodZoneCodes(t *testing.T) {
	lat, lng := 27.95, -82.46
	floodCode := "AE,X"

	input := SubjectInput{
		Type:          "off_market",
		Lat:           &lat,
		Lng:           &lng,
		FloodZoneCode: &floodCode,
	}

	s := resolveOffMarketSubject(input)

	if s.FloodZone == nil {
		t.Fatal("FloodZone should not be nil")
	}
	if len(s.FloodZoneCodes) != 2 {
		t.Fatalf("expected 2 flood zone codes, got %d", len(s.FloodZoneCodes))
	}
	if s.FloodZoneCodes[0] != "AE" || s.FloodZoneCodes[1] != "X" {
		t.Errorf("got %v, want [AE X]", s.FloodZoneCodes)
	}
}
