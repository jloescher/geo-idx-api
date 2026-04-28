package comps

import (
	"math"
	"testing"
	"time"
)

// --- Financial calculation tests ---

func TestComputeMonthlyPI(t *testing.T) {
	// $300,000 loan at 7% for 30 years → known ~$1,995.91
	pi := computeMonthlyPI(300000, 0.07, 30)
	if math.Abs(pi-1995.91) > 1 {
		t.Errorf("got %f, want ~1995.91", pi)
	}
}

func TestComputeMonthlyPI_ZeroRate(t *testing.T) {
	pi := computeMonthlyPI(360000, 0, 30)
	expected := 360000.0 / 360.0 // 1000
	if math.Abs(pi-expected) > 0.01 {
		t.Errorf("got %f, want %f", pi, expected)
	}
}

func TestComputeMonthlyPI_ZeroTerm(t *testing.T) {
	pi := computeMonthlyPI(300000, 0.07, 0)
	if pi != 0 {
		t.Errorf("got %f, want 0", pi)
	}
}

func TestComputeDownPayment_Amount(t *testing.T) {
	dp := 60000.0
	f := FinancingInput{PurchasePrice: 300000, DownPaymentAmount: &dp}
	amount, pct := computeDownPayment(f)
	if amount != 60000 {
		t.Errorf("amount: got %f, want 60000", amount)
	}
	if math.Abs(pct-0.20) > 0.001 {
		t.Errorf("pct: got %f, want 0.20", pct)
	}
}

func TestComputeDownPayment_Percent(t *testing.T) {
	pct := 0.20
	f := FinancingInput{PurchasePrice: 300000, DownPaymentPercent: &pct}
	amount, gotPct := computeDownPayment(f)
	if math.Abs(amount-60000) > 0.01 {
		t.Errorf("amount: got %f, want 60000", amount)
	}
	if gotPct != 0.20 {
		t.Errorf("pct: got %f, want 0.20", gotPct)
	}
}

func TestComputePMI_BelowTwentyPercent(t *testing.T) {
	dp := 30000.0 // 10% on 300k
	f := FinancingInput{PurchasePrice: 300000, DownPaymentAmount: &dp}
	loanAmount := 270000.0
	pmi := computePMI(f, loanAmount)
	// Default rate 0.0075 → (270000 * 0.0075) / 12 = 168.75
	if math.Abs(pmi-168.75) > 0.01 {
		t.Errorf("got %f, want 168.75", pmi)
	}
}

func TestComputePMI_AboveTwentyPercent(t *testing.T) {
	dp := 60000.0 // 20% on 300k
	f := FinancingInput{PurchasePrice: 300000, DownPaymentAmount: &dp}
	loanAmount := 240000.0
	pmi := computePMI(f, loanAmount)
	if pmi != 0 {
		t.Errorf("got %f, want 0", pmi)
	}
}

func TestComputePMI_ExplicitMonthly(t *testing.T) {
	explicit := 125.0
	dp := 30000.0
	f := FinancingInput{PurchasePrice: 300000, DownPaymentAmount: &dp, PMIMonthly: &explicit}
	pmi := computePMI(f, 270000)
	if pmi != 125 {
		t.Errorf("got %f, want 125", pmi)
	}
}

func TestComputeLoanSummary(t *testing.T) {
	dp := 60000.0
	f := FinancingInput{
		PurchasePrice:     300000,
		DownPaymentAmount: &dp,
		LoanTermYears:     30,
		InterestRate:      0.07,
	}
	loan := computeLoanSummary(f)

	if loan.LoanAmount != 240000 {
		t.Errorf("LoanAmount: got %f, want 240000", loan.LoanAmount)
	}
	if loan.DownPaymentPct != 0.20 {
		t.Errorf("DownPaymentPct: got %f, want 0.20", loan.DownPaymentPct)
	}
	if loan.MonthlyPMI != 0 {
		t.Errorf("MonthlyPMI: got %f, want 0 (20%% down)", loan.MonthlyPMI)
	}
	if loan.MonthlyPI < 1500 || loan.MonthlyPI > 1700 {
		t.Errorf("MonthlyPI: got %f, expected ~1596", loan.MonthlyPI)
	}
	if loan.CashInvested != 60000 {
		t.Errorf("CashInvested: got %f, want 60000", loan.CashInvested)
	}
}

func TestComputeMonthlyBreakdown(t *testing.T) {
	params := &RentalParamsInput{} // defaults: vacancy 5%, mgmt 8%, maint 5%, capex 5%
	ownership := OwnershipInput{
		AnnualPropertyTaxes:       3600, // 300/mo
		AnnualHomeownersInsurance: 1200, // 100/mo
	}
	loan := LoanSummary{
		MonthlyPI:  1596.73,
		MonthlyPMI: 0,
	}

	m := computeMonthlyBreakdown(2000, params, ownership, loan)

	if math.Abs(m.GrossRent-2000) > 0.01 {
		t.Errorf("GrossRent: got %f, want 2000", m.GrossRent)
	}
	// vacancy = 2000 * 0.05 = 100
	if math.Abs(m.Vacancy-100) > 0.01 {
		t.Errorf("Vacancy: got %f, want 100", m.Vacancy)
	}
	// EGI = 2000 - 100 = 1900
	if math.Abs(m.EGI-1900) > 0.01 {
		t.Errorf("EGI: got %f, want 1900", m.EGI)
	}
	// management = 2000 * 0.08 = 160
	if math.Abs(m.PropertyManagement-160) > 0.01 {
		t.Errorf("PropertyManagement: got %f, want 160", m.PropertyManagement)
	}
	// taxes = 3600/12 = 300
	if math.Abs(m.Taxes-300) > 0.01 {
		t.Errorf("Taxes: got %f, want 300", m.Taxes)
	}
	// NOI = 1900 - (300 + 100 + 160 + 100 + 100) = 1900 - 760 = 1140
	if math.Abs(m.NOI-1140) > 1 {
		t.Errorf("NOI: got %f, want ~1140", m.NOI)
	}
	// CashFlow = NOI - debt_service = 1140 - 1596.73 = -456.73
	if math.Abs(m.CashFlow-(-456.73)) > 1 {
		t.Errorf("CashFlow: got %f, want ~-456.73", m.CashFlow)
	}
}

func TestComputeAnnualMetrics(t *testing.T) {
	monthly := MonthlyBreakdown{
		NOI:         1000,
		CashFlow:    500,
		DebtService: 500,
	}
	loan := LoanSummary{
		PurchasePrice: 300000,
		CashInvested:  60000,
	}

	a := computeAnnualMetrics(monthly, loan)

	if math.Abs(a.AnnualNOI-12000) > 0.01 {
		t.Errorf("AnnualNOI: got %f, want 12000", a.AnnualNOI)
	}
	if math.Abs(a.AnnualCashFlow-6000) > 0.01 {
		t.Errorf("AnnualCashFlow: got %f, want 6000", a.AnnualCashFlow)
	}
	// CashOnCash = 6000 / 60000 = 0.10
	if math.Abs(a.CashOnCash-0.10) > 0.001 {
		t.Errorf("CashOnCash: got %f, want 0.10", a.CashOnCash)
	}
	// DSCR = 1000 / 500 = 2.0
	if math.Abs(a.DSCR-2.0) > 0.01 {
		t.Errorf("DSCR: got %f, want 2.0", a.DSCR)
	}
	// CapRate = 12000 / 300000 = 0.04
	if math.Abs(a.CapRate-0.04) > 0.001 {
		t.Errorf("CapRate: got %f, want 0.04", a.CapRate)
	}
}

func TestComputeAnnualMetrics_ZeroDivisors(t *testing.T) {
	monthly := MonthlyBreakdown{NOI: 1000, CashFlow: 1000, DebtService: 0}
	loan := LoanSummary{CashInvested: 0, PurchasePrice: 0}

	a := computeAnnualMetrics(monthly, loan)

	if a.DSCR != 0 {
		t.Errorf("DSCR: got %f, want 0 (zero debt service)", a.DSCR)
	}
	if a.CashOnCash != 0 {
		t.Errorf("CashOnCash: got %f, want 0 (zero cash invested)", a.CashOnCash)
	}
	if a.CapRate != 0 {
		t.Errorf("CapRate: got %f, want 0 (zero purchase price)", a.CapRate)
	}
}

func TestComputeScenarios(t *testing.T) {
	params := &RentalParamsInput{}
	ownership := OwnershipInput{AnnualPropertyTaxes: 3600, AnnualHomeownersInsurance: 1200}
	loan := LoanSummary{MonthlyPI: 1000, MonthlyPMI: 0, CashInvested: 60000, PurchasePrice: 300000}

	scenarios := computeScenarios(1500, 2000, 2500, params, ownership, loan)

	if scenarios.Conservative.MonthlyRent != 1500 {
		t.Errorf("Conservative.MonthlyRent: got %f, want 1500", scenarios.Conservative.MonthlyRent)
	}
	if scenarios.Base.MonthlyRent != 2000 {
		t.Errorf("Base.MonthlyRent: got %f, want 2000", scenarios.Base.MonthlyRent)
	}
	if scenarios.Upside.MonthlyRent != 2500 {
		t.Errorf("Upside.MonthlyRent: got %f, want 2500", scenarios.Upside.MonthlyRent)
	}
	// Higher rent should yield better cash flow.
	if scenarios.Upside.AnnualCashFlow <= scenarios.Base.AnnualCashFlow {
		t.Error("upside should have better cash flow than base")
	}
	if scenarios.Base.AnnualCashFlow <= scenarios.Conservative.AnnualCashFlow {
		t.Error("base should have better cash flow than conservative")
	}
}

func TestComputeScenarioFlags(t *testing.T) {
	// Positive base, good DSCR
	positive := ScenarioOutput{
		Base:         ScenarioResult{AnnualCashFlow: 5000, DSCR: 1.25},
		Conservative: ScenarioResult{AnnualCashFlow: 2000, DSCR: 1.05},
	}
	flags := computeScenarioFlags(positive)
	if !flags.CashFlowPositive {
		t.Error("CashFlowPositive should be true")
	}
	if !flags.DSCRPass {
		t.Error("DSCRPass should be true (1.25 >= 1.10)")
	}
	if !flags.ConservativeCashFlowPositive {
		t.Error("ConservativeCashFlowPositive should be true")
	}

	// Negative base
	negative := ScenarioOutput{
		Base:         ScenarioResult{AnnualCashFlow: -1000, DSCR: 0.9},
		Conservative: ScenarioResult{AnnualCashFlow: -3000, DSCR: 0.7},
	}
	flags = computeScenarioFlags(negative)
	if flags.CashFlowPositive {
		t.Error("CashFlowPositive should be false")
	}
	if flags.DSCRPass {
		t.Error("DSCRPass should be false (0.9 < 1.10)")
	}
	if flags.ConservativeCashFlowPositive {
		t.Error("ConservativeCashFlowPositive should be false")
	}
}

// --- Sub-type matching tests ---

func TestClassifySubType(t *testing.T) {
	tests := []struct {
		subType string
		want    string
	}{
		{"Single Family Residence", "detached"},
		{"Townhouse", "attached"},
		{"Villa", "attached"},
		{"Condominium", "condo"},
		{"Manufactured", "manufactured"},
		{"Duplex", "multi_2_4"},
		{"Unknown Type", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := classifySubType(tt.subType)
		if got != tt.want {
			t.Errorf("classifySubType(%q): got %q, want %q", tt.subType, got, tt.want)
		}
	}
}

func TestScoreSubTypeMatch_Exact(t *testing.T) {
	q, f := scoreSubTypeMatch("Single Family Residence", "Single Family Residence")
	if q != "exact_subtype" || f != 1.0 {
		t.Errorf("got (%q, %f), want (exact_subtype, 1.0)", q, f)
	}
}

func TestScoreSubTypeMatch_Family(t *testing.T) {
	q, f := scoreSubTypeMatch("Townhouse", "Villa")
	if q != "family_match" || f != 0.6 {
		t.Errorf("got (%q, %f), want (family_match, 0.6)", q, f)
	}
}

func TestScoreSubTypeMatch_CrossFamily(t *testing.T) {
	q, f := scoreSubTypeMatch("Single Family Residence", "Condominium")
	if q != "cross_family_low_confidence" || f != 0.2 {
		t.Errorf("got (%q, %f), want (cross_family_low_confidence, 0.2)", q, f)
	}
}

func TestScoreSubTypeMatch_Unknown(t *testing.T) {
	q, f := scoreSubTypeMatch("Single Family Residence", "")
	if q != "family_match" || f != 0.6 {
		t.Errorf("got (%q, %f), want (family_match, 0.6)", q, f)
	}
}

// --- Scoring tests ---

func TestComputeRecencyFactor(t *testing.T) {
	// Recent close (1 day ago)
	recent := time.Now().Add(-24 * time.Hour)
	f := computeRecencyFactor(&recent, 365)
	if f < 0.99 {
		t.Errorf("recent: got %f, want >0.99", f)
	}

	// Old close (364 days ago)
	old := time.Now().Add(-364 * 24 * time.Hour)
	f = computeRecencyFactor(&old, 365)
	if f > 0.01 {
		t.Errorf("old: got %f, want <0.01", f)
	}

	// Nil close date
	f = computeRecencyFactor(nil, 365)
	if f != 0.5 {
		t.Errorf("nil: got %f, want 0.5", f)
	}
}

func TestComputeFinalSimilarity(t *testing.T) {
	// Perfect scores
	score := computeFinalSimilarity(1.0, 1.0, 1.0)
	if math.Abs(score-1.0) > 0.001 {
		t.Errorf("perfect: got %f, want 1.0", score)
	}

	// Weighted check: 0.8*0.65 + 0.6*0.20 + 0.5*0.15 = 0.52 + 0.12 + 0.075 = 0.715
	score = computeFinalSimilarity(0.8, 0.6, 0.5)
	if math.Abs(score-0.715) > 0.001 {
		t.Errorf("weighted: got %f, want 0.715", score)
	}
}

// --- Tiered filtering tests ---

func TestFilterTieredComps_Tier1Sufficient(t *testing.T) {
	subType := "Single Family Residence"
	closeDate := time.Now().Add(-90 * 24 * time.Hour) // 90 days ago
	rent := 2000.0

	rows := make([]compRow, 4)
	for i := range rows {
		rows[i] = compRow{
			PropertySubType: &subType,
			CloseDate:       &closeDate,
			ClosePrice:      &rent,
			SimilarityScore: 0.9,
			DistanceMeters:  800, // ~0.5 miles
		}
	}

	subject := &resolvedSubject{PropertySubType: &subType}
	params := &RentalParamsInput{}

	closed, _, _ := filterAndRankRentalComps(rows, nil, subject, params)

	if len(closed) < minClosedComps {
		t.Errorf("expected >= %d closed comps, got %d", minClosedComps, len(closed))
	}
}

func TestFilterTieredComps_FallbackToTier2(t *testing.T) {
	subType := "Single Family Residence"
	closeDate := time.Now().Add(-300 * 24 * time.Hour) // 10 months ago
	rent := 2000.0

	// Only 2 comps near (tier 1 insufficient), but within tier 2 range
	rows := make([]compRow, 4)
	for i := range rows {
		rows[i] = compRow{
			PropertySubType: &subType,
			CloseDate:       &closeDate,
			ClosePrice:      &rent,
			SimilarityScore: 0.85,
			DistanceMeters:  3000, // ~1.9 miles (within tier 2)
		}
	}

	subject := &resolvedSubject{PropertySubType: &subType}
	params := &RentalParamsInput{}

	closed, _, warnings := filterAndRankRentalComps(rows, nil, subject, params)

	if len(closed) < minClosedComps {
		t.Errorf("expected >= %d closed comps, got %d", minClosedComps, len(closed))
	}
	hasWarning := false
	for _, w := range warnings {
		if w != "" {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Error("expected a fallback warning")
	}
}

func TestFilterTieredComps_FallbackToFamily(t *testing.T) {
	sfr := "Single Family Residence"
	townhouse := "Townhouse" // different family
	villa := "Villa"         // same family as townhouse = attached
	closeDate := time.Now().Add(-60 * 24 * time.Hour)
	rent := 2000.0

	// Only 1 exact match, but 3 family matches
	rows := []compRow{
		{PropertySubType: &sfr, CloseDate: &closeDate, ClosePrice: &rent, SimilarityScore: 0.9, DistanceMeters: 800},
	}

	// The subject is a townhouse, so SFR is cross-family, villa/townhouse are family
	subject := &resolvedSubject{PropertySubType: &townhouse}
	params := &RentalParamsInput{}

	// Add family-match comps (villa = same family as townhouse)
	for i := 0; i < 3; i++ {
		rows = append(rows, compRow{
			PropertySubType: &villa,
			CloseDate:       &closeDate,
			ClosePrice:      &rent,
			SimilarityScore: 0.8,
			DistanceMeters:  2000,
		})
	}

	closed, _, _ := filterAndRankRentalComps(rows, nil, subject, params)

	if len(closed) < minClosedComps {
		t.Errorf("expected >= %d closed comps, got %d", minClosedComps, len(closed))
	}
}

func TestFilterTieredComps_CrossFamilyDisabled(t *testing.T) {
	sfr := "Single Family Residence"
	condo := "Condominium"
	closeDate := time.Now().Add(-60 * 24 * time.Hour)
	rent := 2000.0

	rows := []compRow{
		{PropertySubType: &condo, CloseDate: &closeDate, ClosePrice: &rent, SimilarityScore: 0.9, DistanceMeters: 800},
	}

	subject := &resolvedSubject{PropertySubType: &sfr}
	params := &RentalParamsInput{} // allowCrossFamily defaults to false

	closed, _, _ := filterAndRankRentalComps(rows, nil, subject, params)

	// Condo vs SFR is cross-family. With allow_cross_family=false, tier 4 is skipped.
	// The comp should still appear since applyTieredFiltering returns "all" as fallback.
	// But it won't be filtered into tiers 1-3.
	for _, c := range closed {
		if c.matchQuality == "exact_subtype" {
			t.Error("SFR vs Condo should not be exact_subtype")
		}
	}
}

func TestFilterTieredComps_CrossFamilyEnabled(t *testing.T) {
	sfr := "Single Family Residence"
	condo := "Condominium"
	closeDate := time.Now().Add(-60 * 24 * time.Hour)
	rent := 2000.0
	trueVal := true

	rows := make([]compRow, 4)
	for i := range rows {
		rows[i] = compRow{
			PropertySubType: &condo,
			CloseDate:       &closeDate,
			ClosePrice:      &rent,
			SimilarityScore: 0.75,
			DistanceMeters:  1500,
		}
	}

	subject := &resolvedSubject{PropertySubType: &sfr}
	params := &RentalParamsInput{AllowCrossFamily: &trueVal}

	closed, _, warnings := filterAndRankRentalComps(rows, nil, subject, params)

	if len(closed) == 0 {
		t.Error("expected cross-family comps when allowed")
	}

	hasCrossFamilyWarning := false
	for _, w := range warnings {
		if w != "" {
			hasCrossFamilyWarning = true
		}
	}
	if !hasCrossFamilyWarning {
		t.Error("expected cross-family warning")
	}
}

// --- Rent estimation tests ---

func TestEstimateRent(t *testing.T) {
	rent1, rent2, rent3, rent4 := 1800.0, 2000.0, 2200.0, 2400.0
	closed := []rankedRentalComp{
		{rent: &rent1, finalScore: 0.9},
		{rent: &rent2, finalScore: 0.8},
		{rent: &rent3, finalScore: 0.7},
		{rent: &rent4, finalScore: 0.6},
	}

	est := estimateRent(closed, nil)

	if est.CompCount != 4 {
		t.Errorf("CompCount: got %d, want 4", est.CompCount)
	}
	// Weighted average should be between 1800 and 2400, biased toward 1800 (higher weight)
	if est.Recommended < 1800 || est.Recommended > 2400 {
		t.Errorf("Recommended: got %f, expected between 1800 and 2400", est.Recommended)
	}
	if est.Low >= est.Recommended {
		t.Error("Low should be less than Recommended")
	}
	if est.High <= est.Recommended {
		t.Error("High should be greater than Recommended")
	}
}

func TestEstimateRent_FewComps(t *testing.T) {
	rent1 := 2000.0
	closed := []rankedRentalComp{
		{rent: &rent1, finalScore: 0.9},
	}

	est := estimateRent(closed, nil)

	if est.CompCount != 1 {
		t.Errorf("CompCount: got %d, want 1", est.CompCount)
	}
	if est.Recommended != 2000 {
		t.Errorf("Recommended: got %f, want 2000", est.Recommended)
	}
}

func TestEstimateRent_NoComps(t *testing.T) {
	est := estimateRent(nil, nil)
	if est.CompCount != 0 {
		t.Errorf("CompCount: got %d, want 0", est.CompCount)
	}
	if est.Recommended != 0 {
		t.Errorf("Recommended: got %f, want 0", est.Recommended)
	}
}

func TestEstimateRent_WithActive(t *testing.T) {
	rent1 := 2000.0
	rent2 := 2200.0
	activeRent := 2100.0

	closed := []rankedRentalComp{
		{rent: &rent1, finalScore: 0.9},
		{rent: &rent2, finalScore: 0.8},
	}
	active := []rankedRentalComp{
		{rent: &activeRent, finalScore: 0.85},
	}

	est := estimateRent(closed, active)

	if est.ActiveCompCount != 1 {
		t.Errorf("ActiveCompCount: got %d, want 1", est.ActiveCompCount)
	}
	if est.ActiveMedian == nil {
		t.Fatal("ActiveMedian should not be nil")
	}
	if *est.ActiveMedian != 2100 {
		t.Errorf("ActiveMedian: got %f, want 2100", *est.ActiveMedian)
	}
}

// --- Percentile tests ---

func TestPercentile(t *testing.T) {
	sorted := []float64{100, 200, 300, 400}

	p25 := percentile(sorted, 25)
	if math.Abs(p25-175) > 0.01 {
		t.Errorf("25th: got %f, want 175", p25)
	}

	p50 := percentile(sorted, 50)
	if math.Abs(p50-250) > 0.01 {
		t.Errorf("50th: got %f, want 250", p50)
	}

	p75 := percentile(sorted, 75)
	if math.Abs(p75-325) > 0.01 {
		t.Errorf("75th: got %f, want 325", p75)
	}
}

func TestPercentile_Single(t *testing.T) {
	v := percentile([]float64{42}, 50)
	if v != 42 {
		t.Errorf("got %f, want 42", v)
	}
}

func TestPercentile_Empty(t *testing.T) {
	v := percentile(nil, 50)
	if v != 0 {
		t.Errorf("got %f, want 0", v)
	}
}

// --- Validation tests ---

func TestValidateRentalParams_Missing(t *testing.T) {
	req := &RunCompsRequest{Mode: "rent_hold_cashflow"}
	if err := validateRentalParams(req); err == nil {
		t.Fatal("expected error for nil rental_params")
	}
}

func TestValidateRentalParams_MissingPurchasePrice(t *testing.T) {
	dp := 60000.0
	req := &RunCompsRequest{
		Mode: "rent_hold_cashflow",
		RentalParams: &RentalParamsInput{
			Financing: FinancingInput{PurchasePrice: 0, DownPaymentAmount: &dp, LoanTermYears: 30},
		},
	}
	if err := validateRentalParams(req); err == nil {
		t.Fatal("expected error for zero purchase_price")
	}
}

func TestValidateRentalParams_MissingDownPayment(t *testing.T) {
	req := &RunCompsRequest{
		Mode: "rent_hold_cashflow",
		RentalParams: &RentalParamsInput{
			Financing: FinancingInput{PurchasePrice: 300000, LoanTermYears: 30},
		},
	}
	if err := validateRentalParams(req); err == nil {
		t.Fatal("expected error for missing down payment")
	}
}

func TestValidateRentalParams_Valid(t *testing.T) {
	dp := 60000.0
	req := &RunCompsRequest{
		Mode: "rent_hold_cashflow",
		RentalParams: &RentalParamsInput{
			Financing: FinancingInput{PurchasePrice: 300000, DownPaymentAmount: &dp, LoanTermYears: 30, InterestRate: 0.07},
			Ownership: OwnershipInput{AnnualPropertyTaxes: 3600, AnnualHomeownersInsurance: 1200},
		},
	}
	if err := validateRentalParams(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRequest_RentalMode(t *testing.T) {
	lat, lng := 27.95, -82.46
	dp := 60000.0
	req := &RunCompsRequest{
		Subject: SubjectInput{Type: "off_market", Lat: &lat, Lng: &lng},
		Mode:    "rent_hold_cashflow",
		Scope:   ScopeInput{Type: "radius"},
		RentalParams: &RentalParamsInput{
			Financing: FinancingInput{PurchasePrice: 300000, DownPaymentAmount: &dp, LoanTermYears: 30, InterestRate: 0.07},
			Ownership: OwnershipInput{AnnualPropertyTaxes: 3600, AnnualHomeownersInsurance: 1200},
		},
	}
	if err := validateRequest(req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRequest_RentalModeNoParams(t *testing.T) {
	lat, lng := 27.95, -82.46
	req := &RunCompsRequest{
		Subject: SubjectInput{Type: "off_market", Lat: &lat, Lng: &lng},
		Mode:    "rent_hold_cashflow",
		Scope:   ScopeInput{Type: "radius"},
	}
	if err := validateRequest(req); err == nil {
		t.Fatal("expected error for rent_hold_cashflow without rental_params")
	}
}

// --- Rental defaults tests ---

func TestRentalParamsDefaults(t *testing.T) {
	p := &RentalParamsInput{}

	if p.targetHoldPeriodMonths() != 12 {
		t.Errorf("targetHoldPeriodMonths: got %d, want 12", p.targetHoldPeriodMonths())
	}
	if p.occupancyRate() != 0.95 {
		t.Errorf("occupancyRate: got %f, want 0.95", p.occupancyRate())
	}
	if p.propertyManagementPercent() != 0.08 {
		t.Errorf("propertyManagementPercent: got %f, want 0.08", p.propertyManagementPercent())
	}
	if p.maintenancePercent() != 0.05 {
		t.Errorf("maintenancePercent: got %f, want 0.05", p.maintenancePercent())
	}
	if p.capexPercent() != 0.05 {
		t.Errorf("capexPercent: got %f, want 0.05", p.capexPercent())
	}
	if math.Abs(p.vacancyPercent()-0.05) > 0.001 {
		t.Errorf("vacancyPercent: got %f, want 0.05", p.vacancyPercent())
	}
	if p.allowCrossFamily() != false {
		t.Error("allowCrossFamily: got true, want false")
	}
}

// --- Build rental filters ---

func TestBuildRentalFilters(t *testing.T) {
	f := buildRentalFilters()

	if f.livingAreaPct() != 30 {
		t.Errorf("livingAreaPct: got %d, want 30", f.livingAreaPct())
	}
	if f.matchPropertySubType() != false {
		t.Error("matchPropertySubType should be false for rental")
	}
	if f.matchPool() != false {
		t.Error("matchPool should be false for rental")
	}
	if f.matchWaterfront() != false {
		t.Error("matchWaterfront should be false for rental")
	}
	if f.yearBuiltTolerance() != 25 {
		t.Errorf("yearBuiltTolerance: got %d, want 25", f.yearBuiltTolerance())
	}
}

// --- Map rental comps ---

func TestMapRentalComps(t *testing.T) {
	lid := "MLS-123"
	lat := 27.95
	lng := -82.46
	rent := 2000.0
	closeDate := time.Now().Add(-30 * 24 * time.Hour)
	subType := "Single Family Residence"
	beds := 3
	baths := 2

	ranked := []rankedRentalComp{
		{
			row: compRow{
				ListingID:       &lid,
				Latitude:        &lat,
				Longitude:       &lng,
				ClosePrice:      &rent,
				CloseDate:       &closeDate,
				PropertySubType: &subType,
				BedroomsTotal:   &beds,
				BathroomsTotal:  &baths,
				DistanceMeters:  1609.344,
			},
			matchQuality:   "exact_subtype",
			finalScore:     0.9,
			rent:           &rent,
			rentSource:     "close_price",
			distanceMiles:  1.0,
			isClosedLeased: true,
		},
	}

	comps := mapRentalComps(ranked, nil)

	if len(comps) != 1 {
		t.Fatalf("expected 1 comp, got %d", len(comps))
	}

	c := comps[0]
	if c.ListingID != "MLS-123" {
		t.Errorf("ListingID: got %q, want MLS-123", c.ListingID)
	}
	if c.StatusLabel != "Leased/Closed" {
		t.Errorf("StatusLabel: got %q, want Leased/Closed", c.StatusLabel)
	}
	if c.MatchQuality != "exact_subtype" {
		t.Errorf("MatchQuality: got %q, want exact_subtype", c.MatchQuality)
	}
	if c.RentSource != "close_price" {
		t.Errorf("RentSource: got %q, want close_price", c.RentSource)
	}
	if c.CloseDate == nil {
		t.Error("CloseDate should not be nil for closed comp")
	}
}

func TestMapRentalComps_Active(t *testing.T) {
	lid := "MLS-456"
	lat := 27.95
	lng := -82.46
	rent := 2100.0

	ranked := []rankedRentalComp{
		{
			row: compRow{
				ListingID:      &lid,
				Latitude:       &lat,
				Longitude:      &lng,
				ListPrice:      &rent,
				DistanceMeters: 800,
			},
			matchQuality:   "family_match",
			finalScore:     0.75,
			rent:           &rent,
			rentSource:     "list_price",
			distanceMiles:  0.5,
			isClosedLeased: false,
		},
	}

	comps := mapRentalComps(ranked, nil)

	if len(comps) != 1 {
		t.Fatalf("expected 1 comp, got %d", len(comps))
	}

	c := comps[0]
	if c.StatusLabel != "Active" {
		t.Errorf("StatusLabel: got %q, want Active", c.StatusLabel)
	}
	if c.CloseDate != nil {
		t.Error("CloseDate should be nil for active comp")
	}
}
