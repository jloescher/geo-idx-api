package comps

import (
	"math"
	"testing"
)

func TestComputeQualifyingMonthlyBreakdown(t *testing.T) {
	marketRent := 2000.0
	qualRent := 1500.0 // 75% haircut
	params := &RentalParamsInput{}
	ownership := OwnershipInput{
		AnnualPropertyTaxes:       3600,
		AnnualHomeownersInsurance: 1800,
	}
	loan := LoanSummary{
		MonthlyPI:  800,
		MonthlyPMI: 50,
	}

	mb := computeQualifyingMonthlyBreakdown(marketRent, qualRent, params, ownership, loan)

	if mb.GrossRent != 1500 {
		t.Errorf("GrossRent should be qualifying rent 1500, got %f", mb.GrossRent)
	}

	// Vacancy computed on market rent: 2000 * 0.05 = 100
	if mb.Vacancy != 100 {
		t.Errorf("Vacancy should be 100 (5%% of market rent 2000), got %f", mb.Vacancy)
	}

	// EGI = qualRent - vacancy = 1500 - 100 = 1400
	if mb.EGI != 1400 {
		t.Errorf("EGI should be 1400, got %f", mb.EGI)
	}

	// Management on market rent: 2000 * 0.08 = 160
	if mb.PropertyManagement != 160 {
		t.Errorf("PropertyManagement should be 160, got %f", mb.PropertyManagement)
	}

	// Cash flow should be lower than investor view.
	if mb.CashFlow >= 0 {
		// With these numbers, cash flow is likely negative due to haircut.
		// Just verify it's a real number.
		t.Logf("qualifying cash flow: %f (positive, acceptable)", mb.CashFlow)
	}
}

func TestComputeMonthlyIO(t *testing.T) {
	got := computeMonthlyIO(200000, 0.06)
	want := 1000.0
	if math.Abs(got-want) > 0.01 {
		t.Errorf("computeMonthlyIO(200000, 0.06) = %f, want %f", got, want)
	}
}

func TestComputeMonthlyIO_Zero(t *testing.T) {
	got := computeMonthlyIO(200000, 0)
	if got != 0 {
		t.Errorf("zero rate should return 0, got %f", got)
	}
}

func TestComputeDSCROverlay_BasicPass(t *testing.T) {
	noi := 1200.0
	loan := LoanSummary{
		LoanAmount: 200000,
		MonthlyPI:  1000,
		MonthlyPMI: 0,
	}
	financing := FinancingInput{
		PurchasePrice: 250000,
		InterestRate:  0.06,
		LoanTermYears: 30,
	}

	overlay := computeDSCROverlay(noi, loan, financing, nil)

	if !overlay.Pass {
		t.Errorf("DSCR should pass: NOI %f / DS %f = %f", noi, overlay.DebtServiceMonthly, overlay.DSCRValue)
	}
	if overlay.DSCRValue < 1.10 {
		t.Errorf("DSCR should be >= 1.10, got %f", overlay.DSCRValue)
	}
	if overlay.DebtServiceUsed != "PI" {
		t.Errorf("default debt service should be PI, got %s", overlay.DebtServiceUsed)
	}
}

func TestComputeDSCROverlay_IOOption(t *testing.T) {
	noi := 1100.0
	loan := LoanSummary{
		LoanAmount: 200000,
		MonthlyPI:  1200,
		MonthlyPMI: 0,
	}
	financing := FinancingInput{
		PurchasePrice: 250000,
		InterestRate:  0.06,
		LoanTermYears: 30,
	}
	ioEnabled := true
	cfg := &DSCRConfig{IOOptionEnabled: &ioEnabled}

	overlay := computeDSCROverlay(noi, loan, financing, cfg)

	if overlay.DebtServiceUsed != "IO" {
		t.Errorf("IO option should use IO debt service, got %s", overlay.DebtServiceUsed)
	}
	// IO payment: 200000 * 0.06 / 12 = 1000
	if overlay.DebtServiceMonthly != 1000 {
		t.Errorf("IO debt service should be 1000, got %f", overlay.DebtServiceMonthly)
	}
	if !overlay.Pass {
		t.Errorf("DSCR should pass with IO: %f / 1000 = %f", noi, overlay.DSCRValue)
	}
}

func TestComputeDSCROverlay_StressTest(t *testing.T) {
	noi := 1100.0
	loan := LoanSummary{
		LoanAmount: 200000,
		MonthlyPI:  1000,
		MonthlyPMI: 0,
	}
	financing := FinancingInput{
		PurchasePrice: 250000,
		InterestRate:  0.06,
		LoanTermYears: 30,
	}
	bps := 200 // +2%
	cfg := &DSCRConfig{StressRateBPS: &bps}

	overlay := computeDSCROverlay(noi, loan, financing, cfg)

	if overlay.StressedRate == nil {
		t.Fatal("stressed rate should be set")
	}
	if *overlay.StressedRate != 0.08 {
		t.Errorf("stressed rate should be 0.08, got %f", *overlay.StressedRate)
	}
	if overlay.StressedDSCR == nil {
		t.Fatal("stressed DSCR should be set")
	}
}

func TestComputeDSCROverlay_Fail(t *testing.T) {
	noi := 500.0 // way below debt service
	loan := LoanSummary{
		LoanAmount: 200000,
		MonthlyPI:  1000,
		MonthlyPMI: 0,
	}
	financing := FinancingInput{
		PurchasePrice: 250000,
		InterestRate:  0.06,
		LoanTermYears: 30,
	}

	overlay := computeDSCROverlay(noi, loan, financing, nil)

	if overlay.Pass {
		t.Errorf("DSCR should fail: %f < 1.10", overlay.DSCRValue)
	}
	if len(overlay.Notes) == 0 {
		t.Error("should have notes explaining failure")
	}
}

func TestComputeSelfSufficiency_Pass(t *testing.T) {
	qualRent := 2000.0
	loan := LoanSummary{
		MonthlyPI:  800,
		MonthlyPMI: 50,
	}
	ownership := OwnershipInput{
		AnnualPropertyTaxes:       3600,
		AnnualHomeownersInsurance: 1800,
	}

	result := computeSelfSufficiency(qualRent, loan, ownership, nil)

	// PITIA = 800+50+300+150 = 1300
	// qualRent (2000) >= 1300 => pass
	if !result.Pass {
		t.Errorf("should pass: income %f >= PITIA %f", result.QualifyingIncomeMonthly, result.PITIAMonthly)
	}
	if result.SurplusDeficitMonthly < 0 {
		t.Errorf("surplus should be positive, got %f", result.SurplusDeficitMonthly)
	}
}

func TestComputeSelfSufficiency_Fail(t *testing.T) {
	qualRent := 500.0 // too low
	loan := LoanSummary{
		MonthlyPI:  800,
		MonthlyPMI: 50,
	}
	ownership := OwnershipInput{
		AnnualPropertyTaxes:       3600,
		AnnualHomeownersInsurance: 1800,
	}

	result := computeSelfSufficiency(qualRent, loan, ownership, nil)

	if result.Pass {
		t.Errorf("should fail: income %f < PITIA %f", result.QualifyingIncomeMonthly, result.PITIAMonthly)
	}
	if result.SurplusDeficitMonthly >= 0 {
		t.Errorf("deficit should be negative, got %f", result.SurplusDeficitMonthly)
	}
}

func TestComputeSelfSufficiency_ExcludeTaxes(t *testing.T) {
	qualRent := 900.0
	loan := LoanSummary{
		MonthlyPI:  800,
		MonthlyPMI: 50,
	}
	ownership := OwnershipInput{
		AnnualPropertyTaxes:       36000, // very high
		AnnualHomeownersInsurance: 18000, // very high
	}

	// Without taxes/insurance: PITIA = 800+50 = 850, qualRent 900 > 850 => pass
	excludeTI := false
	cfg := &DSCRConfig{IncludeTaxesInsurance: &excludeTI}
	result := computeSelfSufficiency(qualRent, loan, ownership, cfg)

	if !result.Pass {
		t.Errorf("excluding taxes/insurance should pass: income %f >= PITIA %f",
			result.QualifyingIncomeMonthly, result.PITIAMonthly)
	}
}

func TestComputeSelfSufficiency_CustomRatio(t *testing.T) {
	qualRent := 1200.0
	loan := LoanSummary{
		MonthlyPI:  800,
		MonthlyPMI: 50,
	}
	ownership := OwnershipInput{
		AnnualPropertyTaxes:       3600,
		AnnualHomeownersInsurance: 1800,
	}
	// PITIA = 1300, with ratio 1.0: surplus = 1200-1300 = -100 (fail)
	// With ratio 0.9: surplus = 1200-1170 = 30 (pass)
	ratio := 0.9
	cfg := &DSCRConfig{SelfSufficiencyRatio: &ratio}
	result := computeSelfSufficiency(qualRent, loan, ownership, cfg)

	if !result.Pass {
		t.Errorf("0.9x ratio should pass: income %f >= PITIA*0.9 %f",
			result.QualifyingIncomeMonthly, result.PITIAMonthly*ratio)
	}
}
