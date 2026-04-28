package comps

import (
	"math"
	"testing"
	"time"
)

func TestKernelSimilarity_ExactMatch(t *testing.T) {
	beds, baths, sqft, year := 3, 2, 1800, 2000
	subject := &resolvedSubject{
		Bedrooms:       &beds,
		Bathrooms:      &baths,
		LivingAreaSqft: &sqft,
		YearBuilt:      &year,
	}
	comp := compRow{
		BedroomsTotal:  &beds,
		BathroomsTotal: &baths,
		LivingArea:     &sqft,
		YearBuilt:      &year,
	}
	got := kernelSimilarity(subject, comp, 1.0)
	if got < 0.99 {
		t.Errorf("exact match should be ~1.0, got %f", got)
	}
}

func TestKernelSimilarity_NilValues(t *testing.T) {
	subject := &resolvedSubject{}
	comp := compRow{}
	got := kernelSimilarity(subject, comp, 0.5)
	if got != 0.5 {
		t.Errorf("all-nil should return 0.5, got %f", got)
	}
}

func TestKernelSimilarity_DifferentBeds(t *testing.T) {
	beds3, beds5 := 3, 5
	baths := 2
	sqft := 1800
	year := 2000
	subject := &resolvedSubject{
		Bedrooms:       &beds3,
		Bathrooms:      &baths,
		LivingAreaSqft: &sqft,
		YearBuilt:      &year,
	}
	comp := compRow{
		BedroomsTotal:  &beds5,
		BathroomsTotal: &baths,
		LivingArea:     &sqft,
		YearBuilt:      &year,
	}
	got := kernelSimilarity(subject, comp, 1.0)
	// beds differ by 2 => exp(-2*2)=exp(-4)~0.018
	// everything else matches perfectly
	if got > 0.95 || got < 0.7 {
		t.Errorf("2-bed diff should lower score significantly from 1.0, got %f", got)
	}
}

func TestDistanceDecayFactor(t *testing.T) {
	tests := []struct {
		miles     float64
		decay     float64
		wantAbove float64
		wantBelow float64
	}{
		{0, 0.75, 0.99, 1.01},
		{0.3, 0.75, 0.80, 0.90},
		{0.75, 0.75, 0.35, 0.40},
		{1.5, 0.75, 0.01, 0.03},
	}
	for _, tt := range tests {
		got := distanceDecayFactor(tt.miles, tt.decay)
		if got < tt.wantAbove || got > tt.wantBelow {
			t.Errorf("distanceDecayFactor(%v, %v) = %f, want in [%f, %f]",
				tt.miles, tt.decay, got, tt.wantAbove, tt.wantBelow)
		}
	}
}

func TestDistanceDecayFactor_ZeroDecay(t *testing.T) {
	got := distanceDecayFactor(5.0, 0)
	if got != 1.0 {
		t.Errorf("zero decay miles should return 1.0, got %f", got)
	}
}

func TestRecencyDecayFactor(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		closeDate *time.Time
		halfLife  int
		wantAbove float64
		wantBelow float64
	}{
		{"nil date", nil, 90, 0.49, 0.51},
		{"1 day old", timePtr(now.AddDate(0, 0, -1)), 90, 0.98, 1.0},
		{"90 days (half-life)", timePtr(now.AddDate(0, 0, -90)), 90, 0.48, 0.52},
		{"180 days (2x half-life)", timePtr(now.AddDate(0, 0, -180)), 90, 0.24, 0.26},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recencyDecayFactor(tt.closeDate, tt.halfLife)
			if got < tt.wantAbove || got > tt.wantBelow {
				t.Errorf("recencyDecayFactor = %f, want in [%f, %f]",
					got, tt.wantAbove, tt.wantBelow)
			}
		})
	}
}

func TestWeightedMedian_Simple(t *testing.T) {
	values := []float64{100, 200, 300}
	weights := []float64{1, 1, 1}
	got := weightedMedian(values, weights)
	if got != 200 {
		t.Errorf("equal-weight median of [100,200,300] should be 200, got %f", got)
	}
}

func TestWeightedMedian_Skewed(t *testing.T) {
	values := []float64{100, 200, 300}
	weights := []float64{0.1, 0.1, 10.0}
	got := weightedMedian(values, weights)
	if got != 300 {
		t.Errorf("heavily-weighted 300 should be median, got %f", got)
	}
}

func TestWeightedMedian_Single(t *testing.T) {
	got := weightedMedian([]float64{42}, []float64{1})
	if got != 42 {
		t.Errorf("single value should return itself, got %f", got)
	}
}

func TestWeightedMedian_Empty(t *testing.T) {
	got := weightedMedian(nil, nil)
	if got != 0 {
		t.Errorf("empty should return 0, got %f", got)
	}
}

func TestWinsorizeRentPerSqft_TooFew(t *testing.T) {
	comps := make([]rankedRentalComp, 3)
	for i := range comps {
		r := float64(1000 + i*100)
		sqft := 1000
		comps[i].rent = &r
		comps[i].row.LivingArea = &sqft
	}
	clamped := winsorizeRentPerSqft(comps, 0.10)
	if clamped != 0 {
		t.Errorf("fewer than 5 comps should skip winsorization, got %d clamped", clamped)
	}
}

func TestWinsorizeRentPerSqft_ClampOutliers(t *testing.T) {
	// 7 comps: 5 normal, 1 low outlier, 1 high outlier.
	rents := []float64{500, 1000, 1050, 1100, 1050, 1000, 2000}
	sqft := 1000
	comps := make([]rankedRentalComp, len(rents))
	for i, r := range rents {
		r := r // shadow
		comps[i].rent = &r
		comps[i].row.LivingArea = &sqft
	}
	clamped := winsorizeRentPerSqft(comps, 0.10)
	if clamped == 0 {
		t.Error("expected at least one value to be winsorized")
	}
	// After winsorization, extreme values should be clamped.
	for _, c := range comps {
		if *c.rent < 400 || *c.rent > 2100 {
			t.Errorf("unexpected rent after winsorization: %f", *c.rent)
		}
	}
}

func TestEstimateRentV2_NilConfig(t *testing.T) {
	beds, baths, sqft, year := 3, 2, 1800, 2005
	subject := &resolvedSubject{
		Bedrooms:       &beds,
		Bathrooms:      &baths,
		LivingAreaSqft: &sqft,
		YearBuilt:      &year,
	}

	closeDate := time.Now().AddDate(0, -1, 0)
	rents := []float64{1500, 1600, 1550, 1700, 1580}
	closed := make([]rankedRentalComp, len(rents))
	for i, r := range rents {
		r := r
		closed[i] = rankedRentalComp{
			row: compRow{
				BedroomsTotal:  &beds,
				BathroomsTotal: &baths,
				LivingArea:     &sqft,
				YearBuilt:      &year,
				CloseDate:      &closeDate,
				ClosePrice:     &r,
			},
			rent:           &r,
			rentSource:     "close_price",
			distanceMiles:  0.5,
			isClosedLeased: true,
		}
	}

	est := estimateRentV2(closed, nil, subject, nil)

	if est.Recommended <= 0 {
		t.Errorf("recommended rent should be positive, got %f", est.Recommended)
	}
	if est.CompCount != 5 {
		t.Errorf("comp count should be 5, got %d", est.CompCount)
	}
	if est.MethodVersion != "rent_weighting_v2_kernel_winsor_blend" {
		t.Errorf("unexpected method version: %s", est.MethodVersion)
	}
	if est.Low > est.Recommended || est.Recommended > est.High {
		t.Errorf("expected Low <= Recommended <= High, got %f, %f, %f",
			est.Low, est.Recommended, est.High)
	}
}

func TestEstimateRentV2_NoComps(t *testing.T) {
	subject := &resolvedSubject{}
	est := estimateRentV2(nil, nil, subject, nil)
	if est.Recommended != 0 {
		t.Errorf("no comps should return 0 recommended, got %f", est.Recommended)
	}
	if est.MethodVersion != "rent_weighting_v2_kernel_winsor_blend" {
		t.Errorf("unexpected method version: %s", est.MethodVersion)
	}
}

func TestEstimateRentV2_OnlyActive(t *testing.T) {
	beds, baths, sqft, year := 3, 2, 1800, 2005
	subject := &resolvedSubject{
		Bedrooms:       &beds,
		Bathrooms:      &baths,
		LivingAreaSqft: &sqft,
		YearBuilt:      &year,
	}

	rents := []float64{1500, 1600, 1550}
	active := make([]rankedRentalComp, len(rents))
	for i, r := range rents {
		r := r
		active[i] = rankedRentalComp{
			row: compRow{
				BedroomsTotal:  &beds,
				BathroomsTotal: &baths,
				LivingArea:     &sqft,
				YearBuilt:      &year,
				ListPrice:      &r,
			},
			rent:           &r,
			rentSource:     "list_price",
			distanceMiles:  0.3,
			isClosedLeased: false,
		}
	}

	est := estimateRentV2(nil, active, subject, nil)
	if est.Recommended <= 0 {
		t.Errorf("active-only should still produce a rent estimate, got %f", est.Recommended)
	}
	if est.CompCount != 0 {
		t.Errorf("closed comp count should be 0 for active-only, got %d", est.CompCount)
	}
	if est.ActiveCompCount != 3 {
		t.Errorf("active comp count should be 3, got %d", est.ActiveCompCount)
	}
}

func TestWeightedPercentile_Basic(t *testing.T) {
	values := []float64{100, 200, 300, 400, 500}
	weights := []float64{1, 1, 1, 1, 1}

	p25 := weightedPercentile(values, weights, 25)
	p75 := weightedPercentile(values, weights, 75)

	if p25 >= p75 {
		t.Errorf("p25 (%f) should be < p75 (%f)", p25, p75)
	}
	if p25 < 100 || p25 > 300 {
		t.Errorf("p25 should be around 200, got %f", p25)
	}
	if p75 < 300 || p75 > 500 {
		t.Errorf("p75 should be around 400, got %f", p75)
	}
}

func TestEstimateRentV2_WeightsPropagate(t *testing.T) {
	beds, baths, sqft, year := 3, 2, 1800, 2005
	subject := &resolvedSubject{
		Bedrooms:       &beds,
		Bathrooms:      &baths,
		LivingAreaSqft: &sqft,
		YearBuilt:      &year,
	}

	closeDate := time.Now().AddDate(0, -1, 0)
	r := 1500.0
	closed := []rankedRentalComp{{
		row: compRow{
			BedroomsTotal:  &beds,
			BathroomsTotal: &baths,
			LivingArea:     &sqft,
			YearBuilt:      &year,
			CloseDate:      &closeDate,
			ClosePrice:     &r,
		},
		rent:           &r,
		rentSource:     "close_price",
		distanceMiles:  0.5,
		isClosedLeased: true,
	}}

	_ = estimateRentV2(closed, nil, subject, nil)

	// After estimateRentV2, weight fields should be populated.
	if closed[0].kernelSim <= 0 {
		t.Errorf("kernelSim should be populated, got %f", closed[0].kernelSim)
	}
	if closed[0].distanceDecay <= 0 {
		t.Errorf("distanceDecay should be populated, got %f", closed[0].distanceDecay)
	}
	if closed[0].rawWeight <= 0 {
		t.Errorf("rawWeight should be populated, got %f", closed[0].rawWeight)
	}
	if math.Abs(closed[0].normWeight-1.0) > 0.01 {
		t.Errorf("single comp normWeight should be ~1.0, got %f", closed[0].normWeight)
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
