package comps

import (
	"math"
	"testing"
)

func TestComputeARV_MedianAdjusted(t *testing.T) {
	beds, baths, sqft, year := 3, 2, 1800, 2005
	subject := &resolvedSubject{
		Bedrooms:       &beds,
		Bathrooms:      &baths,
		LivingAreaSqft: &sqft,
		YearBuilt:      &year,
	}

	prices := []float64{300000, 310000, 320000, 305000, 315000}
	rows := make([]compRow, len(prices))
	for i, p := range prices {
		p := p
		rows[i] = compRow{
			ClosePrice:     &p,
			LivingArea:     &sqft,
			BedroomsTotal:  &beds,
			BathroomsTotal: &baths,
			YearBuilt:      &year,
		}
	}

	arv, method, count := computeARV(rows, subject, &FiltersInput{}, AggMedian)

	if method != "median_adjusted_price" {
		t.Errorf("expected median_adjusted_price, got %s", method)
	}
	if count == 0 {
		t.Error("count should be > 0")
	}
	if arv < 290000 || arv > 330000 {
		t.Errorf("ARV should be around 310k, got %f", arv)
	}
}

func TestComputeARV_NoComps(t *testing.T) {
	subject := &resolvedSubject{}
	arv, method, count := computeARV(nil, subject, &FiltersInput{}, AggMedian)
	if arv != 0 || method != "no_comps" || count != 0 {
		t.Errorf("no comps: arv=%f, method=%s, count=%d", arv, method, count)
	}
}

func TestComputeFlipSummary_Basic(t *testing.T) {
	arv := 350000.0
	fp := &FlipParamsInput{
		RepairBudget: 50000,
	}
	rp := &RentalParamsInput{
		Financing: FinancingInput{
			PurchasePrice: 250000,
			LoanTermYears: 30,
			InterestRate:  0.06,
		},
		Ownership: OwnershipInput{
			AnnualPropertyTaxes:       3600,
			AnnualHomeownersInsurance: 1800,
		},
	}

	summary := computeFlipSummary(arv, "median_adjusted_price", 5, fp, rp)

	if summary.ARV != 350000 {
		t.Errorf("ARV should be 350000, got %f", summary.ARV)
	}
	if summary.PurchasePrice != 250000 {
		t.Errorf("PurchasePrice should be 250000, got %f", summary.PurchasePrice)
	}
	// totalRepairs = 50000 * 1.10 = 55000
	if math.Abs(summary.TotalRepairs-55000) > 1 {
		t.Errorf("TotalRepairs should be ~55000, got %f", summary.TotalRepairs)
	}
	if summary.SalePrice != 350000 {
		t.Errorf("SalePrice should equal ARV, got %f", summary.SalePrice)
	}
	// With no financing, the flip should produce a positive profit.
	if summary.NetProfit <= 0 {
		t.Errorf("expected positive profit, got %f", summary.NetProfit)
	}
	if summary.CashInvested <= 0 {
		t.Error("cash invested should be positive")
	}
	if len(summary.KeyAssumptions) == 0 {
		t.Error("should have key assumptions")
	}
}

func TestComputeFlipSummary_WithFinancing(t *testing.T) {
	arv := 350000.0
	ir := 0.12
	fp := &FlipParamsInput{
		RepairBudget: 50000,
		FlipFinancing: &FlipFinancing{
			InterestRate: ir,
		},
	}
	rp := &RentalParamsInput{
		Financing: FinancingInput{
			PurchasePrice: 250000,
			LoanTermYears: 30,
			InterestRate:  0.06,
		},
		Ownership: OwnershipInput{
			AnnualPropertyTaxes:       3600,
			AnnualHomeownersInsurance: 1800,
		},
	}

	summary := computeFlipSummary(arv, "median_adjusted_price", 5, fp, rp)

	// With 12% interest, financing cost should be reflected.
	if summary.MonthlyCarrying.Financing <= 0 {
		t.Errorf("financing carrying should be > 0, got %f", summary.MonthlyCarrying.Financing)
	}
	// Carrying costs will be substantial with 12% rate on 250k.
	// Monthly financing: 250000 * 0.12 / 12 = 2500
	// Total carry over 6 months is significant, profit may be negative.
	// Just verify the math is consistent.
	if summary.SalePrice != 350000 {
		t.Errorf("sale price should equal ARV, got %f", summary.SalePrice)
	}
	if summary.TotalCostBasis <= 0 {
		t.Error("total cost basis should be positive")
	}
}

func TestComputeMaxOfferPrice_WithTarget(t *testing.T) {
	arv := 350000.0
	targetProfit := 50000.0
	fp := &FlipParamsInput{
		RepairBudget:        50000,
		DesiredProfitAmount: &targetProfit,
	}

	monthlyCarrying := FlipMonthlyCarrying{
		Taxes:     300,
		Insurance: 150,
		Total:     450,
	}

	maxOffer := computeMaxOfferPrice(arv, fp, monthlyCarrying)

	if maxOffer <= 0 {
		t.Fatalf("max offer should be positive, got %f", maxOffer)
	}
	if maxOffer >= arv {
		t.Errorf("max offer (%f) should be less than ARV (%f)", maxOffer, arv)
	}
	// Max offer should leave room for profit.
	if maxOffer > 250000 {
		t.Errorf("max offer seems too high: %f", maxOffer)
	}
}

func TestComputeMaxOfferPrice_NoTarget(t *testing.T) {
	fp := &FlipParamsInput{RepairBudget: 50000}
	maxOffer := computeMaxOfferPrice(350000, fp, FlipMonthlyCarrying{})
	if maxOffer != 0 {
		t.Errorf("no target profit should return 0, got %f", maxOffer)
	}
}

func TestComputeMaxOfferPrice_PercentARV(t *testing.T) {
	arv := 350000.0
	pct := 0.15 // want 15% of ARV as profit
	fp := &FlipParamsInput{
		RepairBudget:            50000,
		DesiredProfitPercentARV: &pct,
	}
	monthlyCarrying := FlipMonthlyCarrying{
		Taxes:     300,
		Insurance: 150,
		Total:     450,
	}

	maxOffer := computeMaxOfferPrice(arv, fp, monthlyCarrying)

	if maxOffer <= 0 {
		t.Fatalf("max offer should be positive, got %f", maxOffer)
	}
	// targetProfit = 350000 * 0.15 = 52500
	// Similar to the dollar amount test.
	if maxOffer >= arv {
		t.Errorf("max offer (%f) should be less than ARV (%f)", maxOffer, arv)
	}
}

func TestComputeFlipVsHoldComparison_BothViable(t *testing.T) {
	flip := FlipSummary{
		NetProfit:     50000,
		ROI:           0.20,
		AnnualizedROI: 0.45,
	}
	hold := HoldSummary{
		AnnualCashFlow: 5000,
		CashOnCash:     0.08,
		DSCR:           1.20,
		SelfSufficient: true,
	}

	cmp := computeFlipVsHoldComparison(flip, hold)

	if !cmp.FlipProfitable {
		t.Error("flip should be profitable")
	}
	if !cmp.HoldCashFlowPositive {
		t.Error("hold should have positive cash flow")
	}
	if !cmp.BothViable {
		t.Error("both should be viable")
	}
	if cmp.NeitherViable {
		t.Error("neither_viable should be false")
	}
	if !cmp.FlipROIAboveThreshold {
		t.Error("flip ROI 20% should be above 15% threshold")
	}
	if !cmp.FlipHigherShortTermROI {
		t.Error("flip annualized 45% should exceed hold CoC 8%")
	}
	if len(cmp.Explanation) == 0 {
		t.Error("should have explanation lines")
	}
}

func TestComputeFlipVsHoldComparison_NeitherViable(t *testing.T) {
	flip := FlipSummary{
		NetProfit:     -10000,
		ROI:           -0.05,
		AnnualizedROI: -0.10,
	}
	hold := HoldSummary{
		AnnualCashFlow: -2000,
		CashOnCash:     -0.03,
		DSCR:           0.85,
		SelfSufficient: false,
	}

	cmp := computeFlipVsHoldComparison(flip, hold)

	if cmp.FlipProfitable {
		t.Error("flip should not be profitable")
	}
	if cmp.HoldCashFlowPositive {
		t.Error("hold should not have positive cash flow")
	}
	if !cmp.NeitherViable {
		t.Error("neither should be viable")
	}
}

func TestFormatDollars(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{50000, "$50,000"},
		{1234567, "$1,234,567"},
		{-5000, "-$5,000"},
		{0, "$0"},
		{999, "$999"},
	}
	for _, tt := range tests {
		got := formatDollars(tt.input)
		if got != tt.want {
			t.Errorf("formatDollars(%f) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{42, "42"},
		{-7, "-7"},
		{1000, "1000"},
	}
	for _, tt := range tests {
		got := itoa(tt.input)
		if got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
