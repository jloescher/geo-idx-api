package comps

import (
	"math"
	"strings"
	"testing"
	"time"
)

// --- Helpers ---

func ptr[T any](v T) *T { return &v }

func floatClose(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

// --- extractMarketRates tests ---

func TestExtractMarketRates_PoolDiff(t *testing.T) {
	// 5 comps with pool at $300k, 5 without at $280k → pool value ≈ $20k.
	var rows []compRow
	for i := 0; i < 5; i++ {
		p := 300000.0
		rows = append(rows, compRow{ClosePrice: &p, PoolPrivateYn: ptr(true), LivingArea: ptr(1500)})
	}
	for i := 0; i < 5; i++ {
		p := 280000.0
		rows = append(rows, compRow{ClosePrice: &p, PoolPrivateYn: ptr(false), LivingArea: ptr(1500)})
	}

	rates := extractMarketRates(rows, 200.0, AggMedian)
	if rates.PoolValue != 20000 {
		t.Errorf("PoolValue = %.0f, want 20000", rates.PoolValue)
	}
}

func TestExtractMarketRates_Clamping(t *testing.T) {
	// Extreme pool diff $500k → should clamp to $75k.
	var rows []compRow
	for i := 0; i < 3; i++ {
		p := 800000.0
		rows = append(rows, compRow{ClosePrice: &p, PoolPrivateYn: ptr(true), LivingArea: ptr(2000)})
	}
	for i := 0; i < 3; i++ {
		p := 300000.0
		rows = append(rows, compRow{ClosePrice: &p, PoolPrivateYn: ptr(false), LivingArea: ptr(2000)})
	}

	rates := extractMarketRates(rows, 200.0, AggMedian)
	if rates.PoolValue != clampPoolMax {
		t.Errorf("PoolValue = %.0f, want %.0f (clamped)", rates.PoolValue, clampPoolMax)
	}
}

func TestExtractMarketRates_InsufficientData(t *testing.T) {
	// Only 1 comp with pool → falls back to default.
	rows := []compRow{
		{ClosePrice: ptr(300000.0), PoolPrivateYn: ptr(true), LivingArea: ptr(1500)},
		{ClosePrice: ptr(280000.0), PoolPrivateYn: ptr(false), LivingArea: ptr(1500)},
	}

	rates := extractMarketRates(rows, 200.0, AggMedian)
	if rates.PoolValue != defaultPoolValue {
		t.Errorf("PoolValue = %.0f, want %.0f (default)", rates.PoolValue, defaultPoolValue)
	}
}

func TestExtractMarketRates_Confidence(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{12, "high"},
		{10, "high"},
		{7, "moderate"},
		{5, "moderate"},
		{3, "low"},
		{1, "low"},
	}
	for _, tt := range tests {
		rows := make([]compRow, tt.count)
		for i := range rows {
			p := 300000.0
			rows[i] = compRow{ClosePrice: &p, LivingArea: ptr(1500)}
		}
		rates := extractMarketRates(rows, 200.0, AggMedian)
		if rates.Confidence != tt.want {
			t.Errorf("count=%d: Confidence = %q, want %q", tt.count, rates.Confidence, tt.want)
		}
	}
}

// --- resolveRates tests ---

func TestResolveRates_MarketDerived(t *testing.T) {
	derived := MarketDerivedRates{
		GLAPerSqft:       180.0,
		PoolValue:        22000.0,
		GaragePerSpace:   8000.0,
		WaterfrontValue:  60000.0,
		YearBuiltPerYear: 600.0,
		LotPerAcre:       30000.0,
		BedroomValue:     6000.0,
	}
	filters := FiltersInput{}
	r := resolveRates(derived, &filters)

	if r.poolValue != 22000.0 {
		t.Errorf("poolValue = %.0f, want 22000 (market-derived)", r.poolValue)
	}
	if r.bedroomValue != 6000.0 {
		t.Errorf("bedroomValue = %.0f, want 6000 (market-derived)", r.bedroomValue)
	}
}

func TestResolveRates_UserOverride(t *testing.T) {
	derived := MarketDerivedRates{
		GLAPerSqft:       180.0,
		PoolValue:        22000.0,
		GaragePerSpace:   8000.0,
		WaterfrontValue:  60000.0,
		YearBuiltPerYear: 600.0,
		LotPerAcre:       30000.0,
		BedroomValue:     6000.0,
	}
	filters := FiltersInput{
		AdjPoolValue:    ptr(15000.0),
		AdjBedroomValue: ptr(4500.0),
	}
	r := resolveRates(derived, &filters)

	if r.poolValue != 15000.0 {
		t.Errorf("poolValue = %.0f, want 15000 (user override)", r.poolValue)
	}
	if r.bedroomValue != 4500.0 {
		t.Errorf("bedroomValue = %.0f, want 4500 (user override)", r.bedroomValue)
	}
	if r.garagePerSpace != 8000.0 {
		t.Errorf("garagePerSpace = %.0f, want 8000 (market-derived, no override)", r.garagePerSpace)
	}
}

// --- computeSimAdjustmentGrid tests ---

func TestComputeSimAdjGrid_UsesResolvedRates(t *testing.T) {
	subject := &resolvedSubject{
		LivingAreaSqft: ptr(2000),
		Pool:           ptr(true),
		GarageSpaces:   ptr(2),
		Waterfront:     ptr(false),
		YearBuilt:      ptr(2010),
		LotSizeAcres:   ptr(0.25),
	}
	comp := compRow{
		ClosePrice:    ptr(300000.0),
		LivingArea:    ptr(1800),
		PoolPrivateYn: ptr(false),
		GarageSpaces:  ptr(1),
		WaterfrontYn:  ptr(false),
		YearBuilt:     ptr(2005),
		LotSizeAcres:  ptr(0.20),
	}
	rates := resolvedRates{
		glaPerSqft:       200.0,
		poolValue:        25000.0,
		garagePerSpace:   10000.0,
		waterfrontValue:  50000.0,
		yearBuiltPerYear: 800.0,
		lotPerAcre:       20000.0,
		bedroomValue:     5000.0,
	}

	grid := computeSimAdjustmentGrid(subject, comp, rates)
	if grid == nil {
		t.Fatal("grid is nil")
	}

	// GLA: (2000-1800) * 200 = 40000
	// Pool: subject has, comp doesn't = +25000
	// Garage: (2-1) * 10000 = 10000
	// Year: (2010-2005) * 800 = 4000
	// Lot: (0.25-0.20) * 20000 = 1000
	// Net: 40000 + 25000 + 10000 + 4000 + 1000 = 80000
	expectedNet := 80000.0
	if grid.NetAdjustment != expectedNet {
		t.Errorf("NetAdjustment = %.0f, want %.0f", grid.NetAdjustment, expectedNet)
	}
	if grid.AdjustedPrice != 380000.0 {
		t.Errorf("AdjustedPrice = %.0f, want 380000", grid.AdjustedPrice)
	}
}

// --- Quarterly trend tests ---

func TestComputeQuarterlyTrend_Appreciation(t *testing.T) {
	now := time.Now()
	recent := now.AddDate(0, -1, 0)
	prior := now.AddDate(0, -5, 0)

	rows := []compRow{
		// Recent: $200/sqft
		{ClosePrice: ptr(300000.0), LivingArea: ptr(1500), CloseDate: &recent},
		{ClosePrice: ptr(280000.0), LivingArea: ptr(1400), CloseDate: &recent},
		// Prior: $180/sqft
		{ClosePrice: ptr(270000.0), LivingArea: ptr(1500), CloseDate: &prior},
		{ClosePrice: ptr(252000.0), LivingArea: ptr(1400), CloseDate: &prior},
	}

	trend := computeQuarterlyTrend(rows, AggMedian)
	// Recent median PPSF ≈ 200, Prior ≈ 180, trend ≈ 11%
	if trend <= 0 {
		t.Errorf("expected positive trend, got %.4f", trend)
	}
}

func TestComputeQuarterlyTrend_Flat(t *testing.T) {
	now := time.Now()
	recent := now.AddDate(0, -1, 0)
	prior := now.AddDate(0, -5, 0)

	rows := []compRow{
		{ClosePrice: ptr(300000.0), LivingArea: ptr(1500), CloseDate: &recent},
		{ClosePrice: ptr(300000.0), LivingArea: ptr(1500), CloseDate: &prior},
	}

	trend := computeQuarterlyTrend(rows, AggMedian)
	if !floatClose(trend, 0, 0.001) {
		t.Errorf("expected ~0 trend, got %.4f", trend)
	}
}

// --- Time adjustment tests ---

func TestApplyTimeAdjustment_AboveThreshold(t *testing.T) {
	closeDate := time.Now().AddDate(0, -4, 0) // 4 months ago
	adj := applyTimeAdjustment(300000.0, closeDate, 0.03)
	if adj == nil {
		t.Fatal("expected non-nil time adjustment")
	}
	// adj = 300000 * 0.03 * (4 / 3) ≈ 12000
	if !floatClose(adj.Adjustment, 12000, 1000) {
		t.Errorf("Adjustment = %.0f, want ~12000", adj.Adjustment)
	}
	if adj.Feature != "time" {
		t.Errorf("Feature = %q, want time", adj.Feature)
	}
}

func TestApplyTimeAdjustment_BelowThreshold(t *testing.T) {
	closeDate := time.Now().AddDate(0, -4, 0)
	adj := applyTimeAdjustment(300000.0, closeDate, 0.005) // 0.5% quarterly
	if adj != nil {
		t.Errorf("expected nil for below-threshold trend, got adjustment %.0f", adj.Adjustment)
	}
}

// --- Bedroom adjustment tests ---

func TestApplyBedroomAdjustment_OneBed(t *testing.T) {
	adj := applyBedroomAdjustment(ptr(3), ptr(2), 5000)
	if adj == nil {
		t.Fatal("expected non-nil bedroom adjustment")
	}
	if adj.Adjustment != 5000 {
		t.Errorf("Adjustment = %.0f, want 5000", adj.Adjustment)
	}
	if adj.Feature != "bedrooms" {
		t.Errorf("Feature = %q, want bedrooms", adj.Feature)
	}
}

func TestApplyBedroomAdjustment_ThreeBed(t *testing.T) {
	adj := applyBedroomAdjustment(ptr(4), ptr(1), 6000)
	if adj == nil {
		t.Fatal("expected non-nil bedroom adjustment")
	}
	if adj.Adjustment != 18000 {
		t.Errorf("Adjustment = %.0f, want 18000 (3 * 6000)", adj.Adjustment)
	}
}

func TestApplyBedroomAdjustment_NoDiff(t *testing.T) {
	adj := applyBedroomAdjustment(ptr(3), ptr(3), 5000)
	if adj != nil {
		t.Errorf("expected nil for equal bedrooms, got %.0f", adj.Adjustment)
	}
}

// --- Re-ranking tests ---

func TestReRankForSimulation_SelectsClosestLowAdj(t *testing.T) {
	now := time.Now()
	recentDate := now.AddDate(0, -2, 0)

	subject := &resolvedSubject{
		LivingAreaSqft:  ptr(1500),
		Bedrooms:        ptr(3),
		PropertySubType: ptr("Single Family"),
	}

	// Create 10 comps with varying distance and GLA.
	var rows []compRow
	for i := 0; i < 10; i++ {
		dist := float64(i) * 500 // 0, 500, 1000, ..., 4500 meters
		gla := 1500 + i*50       // 1500, 1550, ..., 1950
		rows = append(rows, compRow{
			ClosePrice:      ptr(300000.0),
			LivingArea:      ptr(gla),
			DistanceMeters:  dist,
			CloseDate:       &recentDate,
			PropertySubType: ptr("Single Family"),
			BedroomsTotal:   ptr(3),
		})
	}

	rates := resolvedRates{glaPerSqft: 200, poolValue: 20000, garagePerSpace: 7500, waterfrontValue: 50000, yearBuiltPerYear: 500, lotPerAcre: 25000, bedroomValue: 5000}
	params := &SimulationParamsInput{}

	ranked := reRankForSimulation(rows, subject, rates, params, 0)

	if len(ranked) != 4 {
		t.Errorf("got %d comps, want 4 (default maxComps)", len(ranked))
	}
	// First comp should be the closest (0 meters).
	if len(ranked) > 0 && ranked[0].distanceMiles > 0.5 {
		t.Errorf("first comp distance = %.2f miles, expected close to 0", ranked[0].distanceMiles)
	}
}

func TestReRankForSimulation_ExpandsIfThin(t *testing.T) {
	now := time.Now()
	// All comps older than 6 months and far away.
	oldDate := now.AddDate(0, -8, 0)

	subject := &resolvedSubject{
		LivingAreaSqft: ptr(1500),
		Bedrooms:       ptr(3),
	}

	var rows []compRow
	for i := 0; i < 5; i++ {
		rows = append(rows, compRow{
			ClosePrice:     ptr(300000.0),
			LivingArea:     ptr(1500),
			DistanceMeters: 5000, // ~3.1 miles
			CloseDate:      &oldDate,
			BedroomsTotal:  ptr(3),
		})
	}

	rates := resolvedRates{glaPerSqft: 200, bedroomValue: 5000}
	params := &SimulationParamsInput{}

	ranked := reRankForSimulation(rows, subject, rates, params, 0)

	// Should expand since <3 in preferred range and return up to 4.
	if len(ranked) != 4 {
		t.Errorf("got %d comps, want 4 (expanded from all)", len(ranked))
	}
}

// --- Reconciliation tests ---

func TestReconcileSimComps_InverseWeighting(t *testing.T) {
	comps := []rankedSimComp{
		{fullAdjusted: 300000, fullGrossAdjPct: 5},  // weight: 1/5 = 0.20
		{fullAdjusted: 310000, fullGrossAdjPct: 10}, // weight: 1/10 = 0.10
		{fullAdjusted: 290000, fullGrossAdjPct: 20}, // weight: 1/20 = 0.05
	}

	low, high, mid, _ := reconcileSimComps(comps)

	if low != 290000 {
		t.Errorf("low = %.0f, want 290000", low)
	}
	if high != 310000 {
		t.Errorf("high = %.0f, want 310000", high)
	}
	// Weighted: total weight = 0.20+0.10+0.05 = 0.35
	// Normalized: 0.571, 0.286, 0.143
	// Mid ≈ 300000*0.571 + 310000*0.286 + 290000*0.143 ≈ 301,430
	if mid < 299000 || mid > 303000 {
		t.Errorf("weighted mid = %.0f, expected ~301430", mid)
	}
}

func TestReconcileSimComps_Confidence(t *testing.T) {
	tests := []struct {
		name  string
		comps []rankedSimComp
		want  string
	}{
		{
			"high",
			[]rankedSimComp{
				{fullAdjusted: 300000, fullGrossAdjPct: 10},
				{fullAdjusted: 305000, fullGrossAdjPct: 12},
				{fullAdjusted: 298000, fullGrossAdjPct: 8},
			},
			"high",
		},
		{
			"moderate",
			[]rankedSimComp{
				{fullAdjusted: 300000, fullGrossAdjPct: 20},
				{fullAdjusted: 305000, fullGrossAdjPct: 22},
			},
			"moderate",
		},
		{
			"low_single",
			[]rankedSimComp{
				{fullAdjusted: 300000, fullGrossAdjPct: 30},
			},
			"low",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, confidence := reconcileSimComps(tt.comps)
			if confidence != tt.want {
				t.Errorf("confidence = %q, want %q", confidence, tt.want)
			}
		})
	}
}

// --- Risk scoring tests ---

func TestGapMagnitudeRisk_HighGap(t *testing.T) {
	c := gapMagnitudeComponent(12.0) // 12% gap
	if c.RawScore != 90 {
		t.Errorf("RawScore = %.0f, want 90", c.RawScore)
	}
}

func TestGapMagnitudeRisk_NoGap(t *testing.T) {
	c := gapMagnitudeComponent(-1.0) // simulated above contract
	if c.RawScore != 0 {
		t.Errorf("RawScore = %.0f, want 0", c.RawScore)
	}
}

func TestComputeValuationRisk_HighRisk(t *testing.T) {
	comps := []rankedSimComp{
		{fullAdjusted: 280000, fullGrossAdjPct: 22, bedroomMismatch: 2, subTypeMatch: false},
	}
	subject := &resolvedSubject{}
	risk := computeValuationRisk(350000, 280000, comps, 8.0, 0.06, subject)
	if risk == nil {
		t.Fatal("expected non-nil risk score")
	}
	if risk.Score < 50 {
		t.Errorf("Score = %.1f, expected > 50 for high risk scenario", risk.Score)
	}
	if risk.RiskBand != "Elevated" && risk.RiskBand != "High" {
		t.Errorf("RiskBand = %q, expected Elevated or High", risk.RiskBand)
	}
}

func TestComputeValuationRisk_LowRisk(t *testing.T) {
	comps := []rankedSimComp{
		{fullAdjusted: 300000, fullGrossAdjPct: 8, bedroomMismatch: 0, subTypeMatch: true},
		{fullAdjusted: 305000, fullGrossAdjPct: 10, bedroomMismatch: 0, subTypeMatch: true},
		{fullAdjusted: 298000, fullGrossAdjPct: 6, bedroomMismatch: 0, subTypeMatch: true},
		{fullAdjusted: 302000, fullGrossAdjPct: 9, bedroomMismatch: 0, subTypeMatch: true},
	}
	subject := &resolvedSubject{}
	risk := computeValuationRisk(300000, 301000, comps, 1.0, 0.005, subject)
	if risk == nil {
		t.Fatal("expected non-nil risk score")
	}
	if risk.Score > 25 {
		t.Errorf("Score = %.1f, expected <= 25 for low risk scenario", risk.Score)
	}
	if risk.RiskBand != "Low" {
		t.Errorf("RiskBand = %q, expected Low", risk.RiskBand)
	}
}

func TestRiskBand(t *testing.T) {
	tests := []struct {
		score float64
		want  string
	}{
		{0, "Low"},
		{25, "Low"},
		{25.1, "Moderate"},
		{50, "Moderate"},
		{50.1, "Elevated"},
		{75, "Elevated"},
		{75.1, "High"},
		{100, "High"},
	}
	for _, tt := range tests {
		got := riskBand(tt.score)
		if got != tt.want {
			t.Errorf("riskBand(%.1f) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestBPODisclaimers(t *testing.T) {
	disclaimers := buildBPODisclaimers()
	if len(disclaimers) == 0 {
		t.Fatal("expected at least one disclaimer")
	}
	found := false
	for _, d := range disclaimers {
		if strings.Contains(d, "Broker Price Opinion") && strings.Contains(d, "not an appraisal") {
			found = true
		}
	}
	if !found {
		t.Error("mandatory BPO disclaimer text not found")
	}
	// Verify no "Appraisal" used as a label.
	for _, d := range disclaimers {
		if strings.HasPrefix(d, "Appraisal") {
			t.Error("disclaimer should not start with 'Appraisal'")
		}
	}
}

func TestValidateSimulationParams(t *testing.T) {
	tests := []struct {
		name    string
		params  *SimulationParamsInput
		wantErr bool
	}{
		{"nil params ok", nil, false},
		{"valid contract", &SimulationParamsInput{ContractPrice: ptr(300000.0)}, false},
		{"zero contract", &SimulationParamsInput{ContractPrice: ptr(0.0)}, true},
		{"negative contract", &SimulationParamsInput{ContractPrice: ptr(-100.0)}, true},
		{"valid concession", &SimulationParamsInput{ConcessionPercent: ptr(0.03)}, false},
		{"concession too high", &SimulationParamsInput{ConcessionPercent: ptr(1.5)}, true},
		{"concession negative", &SimulationParamsInput{ConcessionPercent: ptr(-0.01)}, true},
		{"valid max comps", &SimulationParamsInput{MaxComps: ptr(6)}, false},
		{"max comps too low", &SimulationParamsInput{MaxComps: ptr(0)}, true},
		{"max comps too high", &SimulationParamsInput{MaxComps: ptr(11)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &RunCompsRequest{SimulationParams: tt.params}
			err := validateSimulationParams(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSimulationParams() err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}
