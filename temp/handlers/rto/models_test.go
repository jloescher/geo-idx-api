package rto

import (
	"math"
	"testing"
)

func closeTo(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

func TestComputePurchasePrice(t *testing.T) {
	// $350,000 at 3% for 3 years = 350000 * 1.03^3 ≈ 382,454.45
	got := computePurchasePrice(350000, 3.0, 3)
	if !closeTo(got, 382454.45, 0.01) {
		t.Fatalf("expected ~382454.45, got %.2f", got)
	}
}

func TestComputePurchasePrice_ZeroAppreciation(t *testing.T) {
	got := computePurchasePrice(350000, 0, 3)
	if got != 350000 {
		t.Fatalf("expected 350000, got %.2f", got)
	}
}

func TestComputeYield(t *testing.T) {
	ri := resolvedInputs{
		ListPrice:             350000,
		YieldPct:              0.8,
		AnnualAppreciationPct: 3.0,
		OptionTermYears:       3,
	}
	result := computeYield(ri)

	// monthly = 350000 * 0.008 / 12 = 233.33
	if !closeTo(result.EstimatedMonthly, 233.33, 0.01) {
		t.Fatalf("yield monthly: expected ~233.33, got %.2f", result.EstimatedMonthly)
	}
	if !closeTo(result.EstimatedPurchasePrice, 382454.45, 0.01) {
		t.Fatalf("yield purchase price: expected ~382454.45, got %.2f", result.EstimatedPurchasePrice)
	}
}

func TestComputeMonthlyPI(t *testing.T) {
	// $300,000 loan at 7.5% for 30 years
	pi := computeMonthlyPI(300000, 0.075, 30)
	// Expected ~$2,097.64
	if !closeTo(pi, 2097.64, 1.0) {
		t.Fatalf("expected ~2097.64, got %.2f", pi)
	}
}

func TestComputeMonthlyPI_ZeroRate(t *testing.T) {
	pi := computeMonthlyPI(360000, 0, 30)
	// 360000 / 360 = 1000
	if !closeTo(pi, 1000, 0.01) {
		t.Fatalf("expected 1000, got %.2f", pi)
	}
}

func TestComputePITI(t *testing.T) {
	ri := resolvedInputs{
		ListPrice:             350000,
		AnnualAppreciationPct: 3.0,
		OptionTermYears:       3,
		YieldPct:              0.8,
		PremiumPct:            15.0,
		InterestRatePct:       7.5,
		LoanTermYears:         30,
		DownPaymentPct:        5.0,
		TaxRatePct:            1.2,
		InsuranceRatePct:      0.35,
	}
	result := computePITI(ri)

	// Purchase price ≈ 382377.27
	// Down payment = 382377.27 * 0.05 = 19118.86
	// Loan = 382377.27 - 19118.86 = 363258.41
	// Taxes = 350000 * 0.012 / 12 = 350.00
	// Insurance = 350000 * 0.0035 / 12 ≈ 102.08
	if result.Breakdown.Taxes < 340 || result.Breakdown.Taxes > 360 {
		t.Fatalf("taxes: expected ~350, got %.2f", result.Breakdown.Taxes)
	}
	if result.Breakdown.Insurance < 95 || result.Breakdown.Insurance > 110 {
		t.Fatalf("insurance: expected ~102, got %.2f", result.Breakdown.Insurance)
	}
	if result.Breakdown.PrincipalInterest < 2400 || result.Breakdown.PrincipalInterest > 2600 {
		t.Fatalf("P&I: expected ~2540, got %.2f", result.Breakdown.PrincipalInterest)
	}
	// Premium = basePITI * 0.15
	if result.Breakdown.Premium < 400 || result.Breakdown.Premium > 500 {
		t.Fatalf("premium: expected ~450, got %.2f", result.Breakdown.Premium)
	}
	// Total should be basePITI + premium
	expectedTotal := result.Breakdown.BasePITI + result.Breakdown.Premium
	if !closeTo(result.EstimatedMonthly, expectedTotal, 0.02) {
		t.Fatalf("total: expected %.2f, got %.2f", expectedTotal, result.EstimatedMonthly)
	}
}

func TestComputeHybrid(t *testing.T) {
	// hybrid = yield*0.4 + piti*0.6
	result := computeHybrid(233.33, 3000.00, 382377.27)
	expected := 233.33*0.4 + 3000.00*0.6 // 93.33 + 1800.00 = 1893.33
	if !closeTo(result.EstimatedMonthly, expected, 0.01) {
		t.Fatalf("hybrid monthly: expected %.2f, got %.2f", expected, result.EstimatedMonthly)
	}
	if !result.Recommended {
		t.Fatal("hybrid should be marked as recommended")
	}
}

func TestComputeAllModels(t *testing.T) {
	ri := resolvedInputs{
		ListPrice:             350000,
		AnnualAppreciationPct: 3.0,
		OptionTermYears:       3,
		YieldPct:              0.8,
		PremiumPct:            15.0,
		InterestRatePct:       7.5,
		LoanTermYears:         30,
		DownPaymentPct:        5.0,
		TaxRatePct:            1.2,
		InsuranceRatePct:      0.35,
	}
	models := computeAllModels(ri)

	if models.YieldBased.EstimatedMonthly <= 0 {
		t.Fatal("yield model should produce positive monthly")
	}
	if models.PITIPremium.EstimatedMonthly <= 0 {
		t.Fatal("PITI model should produce positive monthly")
	}
	if models.Hybrid.EstimatedMonthly <= 0 {
		t.Fatal("hybrid model should produce positive monthly")
	}
	if !models.Hybrid.Recommended {
		t.Fatal("hybrid should be recommended")
	}
	// Hybrid should be between yield and PITI
	if models.Hybrid.EstimatedMonthly < models.YieldBased.EstimatedMonthly {
		t.Fatal("hybrid should be >= yield")
	}
	if models.Hybrid.EstimatedMonthly > models.PITIPremium.EstimatedMonthly {
		t.Fatal("hybrid should be <= PITI")
	}
}

func TestResolveInputs_Defaults(t *testing.T) {
	req := EstimateRequest{}
	ri := resolveInputs(req, 300000, nil)
	if ri.ListPrice != 300000 {
		t.Fatalf("expected list price 300000, got %.2f", ri.ListPrice)
	}
	if ri.OptionTermYears != 3 {
		t.Fatalf("expected option term 3, got %d", ri.OptionTermYears)
	}
	if ri.YieldPct != 0.8 {
		t.Fatalf("expected yield 0.8, got %.2f", ri.YieldPct)
	}
	if ri.TaxRatePct != defaultTaxRatePct {
		t.Fatalf("expected tax rate %.2f, got %.2f", defaultTaxRatePct, ri.TaxRatePct)
	}
}

func TestResolveInputs_DBTaxRate(t *testing.T) {
	req := EstimateRequest{}
	dbTax := 1.5
	ri := resolveInputs(req, 300000, &dbTax)
	if ri.TaxRatePct != 1.5 {
		t.Fatalf("expected DB tax rate 1.5, got %.2f", ri.TaxRatePct)
	}
}

func TestResolveInputs_RequestOverridesDB(t *testing.T) {
	reqTax := 2.0
	req := EstimateRequest{TaxRatePct: &reqTax}
	dbTax := 1.5
	ri := resolveInputs(req, 300000, &dbTax)
	if ri.TaxRatePct != 2.0 {
		t.Fatalf("expected request tax rate 2.0, got %.2f", ri.TaxRatePct)
	}
}

func TestComputeHybridForSearch_NilTax(t *testing.T) {
	result := ComputeHybridForSearch(350000, nil)
	if result.EstimatedMonthly <= 0 {
		t.Fatal("expected positive monthly estimate")
	}
	if result.EstimatedPurchasePrice <= 350000 {
		t.Fatal("expected purchase price > list price (appreciation)")
	}
	if !result.Recommended {
		t.Fatal("hybrid should be recommended")
	}
}

func TestComputeHybridForSearch_TaxBelowDefault(t *testing.T) {
	// Tax annual $3,000 on $350,000 = 0.857% < 1.2% default → should use default
	taxAnnual := 3000.0
	resultDefault := ComputeHybridForSearch(350000, nil)
	resultLowTax := ComputeHybridForSearch(350000, &taxAnnual)

	if resultLowTax.EstimatedMonthly != resultDefault.EstimatedMonthly {
		t.Fatalf("expected same monthly when DB tax < default: default=%.2f, lowTax=%.2f",
			resultDefault.EstimatedMonthly, resultLowTax.EstimatedMonthly)
	}
}

func TestComputeHybridForSearch_TaxAboveDefault(t *testing.T) {
	// Tax annual $7,000 on $350,000 = 2.0% > 1.2% default → should use DB rate
	taxAnnual := 7000.0
	resultDefault := ComputeHybridForSearch(350000, nil)
	resultHighTax := ComputeHybridForSearch(350000, &taxAnnual)

	if resultHighTax.EstimatedMonthly <= resultDefault.EstimatedMonthly {
		t.Fatalf("expected higher monthly when DB tax > default: default=%.2f, highTax=%.2f",
			resultDefault.EstimatedMonthly, resultHighTax.EstimatedMonthly)
	}
}

func TestComputeHybridForSearch_ZeroTax(t *testing.T) {
	// Zero tax amount should fall back to default
	taxAnnual := 0.0
	resultDefault := ComputeHybridForSearch(350000, nil)
	resultZero := ComputeHybridForSearch(350000, &taxAnnual)

	if resultZero.EstimatedMonthly != resultDefault.EstimatedMonthly {
		t.Fatalf("expected same monthly with zero tax: default=%.2f, zero=%.2f",
			resultDefault.EstimatedMonthly, resultZero.EstimatedMonthly)
	}
}
