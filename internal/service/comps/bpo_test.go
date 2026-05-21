package comps

import (
	"testing"
	"time"
)

func TestURARGridHas14Lines(t *testing.T) {
	subject := SubjectProfile{
		LivingArea: 2000, Bedrooms: 3, Bathrooms: 2, YearBuilt: 2000,
		LotSizeAcres: 0.25, GarageSpaces: 2, PoolPrivate: true,
	}
	comp := CompRecord{
		ListingKey: "C1", ClosePrice: 400000, LivingArea: 1900,
		Bedrooms: 3, Bathrooms: 2, YearBuilt: 1998, CloseDate: "2025-06-01",
		GarageSpaces: 2, PoolPrivate: false,
	}
	rates := BpoMarketRates{
		GLAPerSF: 200, BedPerRoom: 8000, BathPerFull: 5000, AgePerYear: 1500,
		LotPerAcre: 25000, GaragePerSpace: 10000, PoolValue: 22000, WaterfrontValue: 50000,
		Method: bpoRateSourceOLS,
	}
	lines := buildURARGrid(subject, comp, rates)
	if len(lines) != 14 {
		t.Fatalf("expected 14 URAR lines, got %d", len(lines))
	}
	for _, l := range lines {
		if l.RateSource == "" {
			t.Fatalf("line %s missing rate_source", l.Feature)
		}
	}
}

func TestRenovationCreditsDepreciate(t *testing.T) {
	year := currentYear() - 3
	subject := SubjectProfile{RenovatedKitchenYear: year}
	rates := BpoMarketRates{MedianGLA: 2000, GLAPerSF: 200}
	credits := renovationCredits(subject, rates)
	if len(credits) != 1 || credits[0].Amount <= 0 {
		t.Fatalf("expected full kitchen credit, got %+v", credits)
	}
	subject.RenovatedKitchenYear = currentYear() - 12
	credits = renovationCredits(subject, rates)
	if len(credits) != 0 {
		t.Fatalf("expected no credit after 10+ years, got %+v", credits)
	}
}

func TestExtractMarketRatesOLS(t *testing.T) {
	sold := []CompRecord{
		{ClosePrice: 360000, LivingArea: 1800, Bedrooms: 3, Bathrooms: 2, YearBuilt: 1995, CloseDate: "2025-01-01"},
		{ClosePrice: 380000, LivingArea: 1900, Bedrooms: 3, Bathrooms: 2, YearBuilt: 1998, CloseDate: "2025-02-01"},
		{ClosePrice: 400000, LivingArea: 2000, Bedrooms: 3, Bathrooms: 2, YearBuilt: 2000, CloseDate: "2025-03-01"},
		{ClosePrice: 420000, LivingArea: 2100, Bedrooms: 3, Bathrooms: 2, YearBuilt: 2002, CloseDate: "2025-04-01"},
		{ClosePrice: 440000, LivingArea: 2200, Bedrooms: 4, Bathrooms: 2, YearBuilt: 2004, CloseDate: "2025-05-01"},
		{ClosePrice: 460000, LivingArea: 2300, Bedrooms: 4, Bathrooms: 3, YearBuilt: 2006, CloseDate: "2025-06-01"},
	}
	rates := extractMarketRates(sold, 6)
	if rates.GLAPerSF <= 0 {
		t.Fatalf("expected positive GLA rate, got %+v", rates)
	}
	if rates.Method != bpoRateSourceOLS && rates.Method != bpoRateSourcePaired {
		t.Fatalf("unexpected method %s", rates.Method)
	}
}

func TestReconcileBPOWeights(t *testing.T) {
	subject := SubjectProfile{LivingArea: 2000}
	sold := []CompRecord{
		{ClosePrice: 400000, LivingArea: 2000, DistanceMiles: 0.2, CloseDate: "2025-06-01"},
		{ClosePrice: 380000, LivingArea: 1800, DistanceMiles: 2.0, CloseDate: "2024-01-01"},
	}
	rates := extractMarketRates(sold, 3)
	sold, grids := applyURARGrid(subject, sold, rates)
	recon := reconcileBPO(subject, sold, grids, rates)
	if recon.PointEstimate <= 0 {
		t.Fatalf("expected positive estimate, got %+v", recon)
	}
}

func currentYear() int {
	return time.Now().Year()
}
