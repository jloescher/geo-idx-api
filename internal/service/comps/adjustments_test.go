package comps

import "testing"

func TestApplyAdjustmentsAddsPoolLine(t *testing.T) {
	subject := SubjectProfile{LivingArea: 2000, Bedrooms: 3, PoolPrivate: true, YearBuilt: 2000}
	comp := CompRecord{
		ListingKey: "X", ClosePrice: 400000, LivingArea: 1800, Bedrooms: 3,
		PoolPrivate: false, YearBuilt: 1998,
	}
	out := applyAdjustments(subject, []CompRecord{comp}, FiltersInput{})
	if len(out) != 1 || out[0].AdjustedPrice <= 400000 {
		t.Fatalf("expected positive pool adjustment, got %+v", out[0])
	}
}

func TestBPOEstimateOLS(t *testing.T) {
	subject := SubjectProfile{LivingArea: 2000}
	sold := []CompRecord{
		{ClosePrice: 360000, LivingArea: 1800, Bedrooms: 3, Bathrooms: 2, YearBuilt: 1995, CloseDate: "2025-01-01"},
		{ClosePrice: 380000, LivingArea: 1900, Bedrooms: 3, Bathrooms: 2, YearBuilt: 1998, CloseDate: "2025-02-01"},
		{ClosePrice: 400000, LivingArea: 2000, Bedrooms: 3, Bathrooms: 2, YearBuilt: 2000, CloseDate: "2025-03-01"},
		{ClosePrice: 420000, LivingArea: 2100, Bedrooms: 3, Bathrooms: 2, YearBuilt: 2002, CloseDate: "2025-04-01"},
		{ClosePrice: 440000, LivingArea: 2200, Bedrooms: 4, Bathrooms: 2, YearBuilt: 2004, CloseDate: "2025-05-01"},
		{ClosePrice: 460000, LivingArea: 2300, Bedrooms: 4, Bathrooms: 3, YearBuilt: 2006, CloseDate: "2025-06-01"},
	}
	rates := extractMarketRates(sold, 6)
	sold, grids := applyURARGrid(subject, sold, rates)
	recon := reconcileBPO(subject, sold, grids, rates)
	if recon.PointEstimate <= 0 || recon.Confidence < 40 {
		t.Fatalf("est=%v conf=%v rates=%+v", recon.PointEstimate, recon.Confidence, rates)
	}
}

func intPtr(n int) *int { return &n }
