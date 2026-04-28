package rto

import "math"

// --- Request / Response types ---

// EstimateRequest is the JSON body for POST /api/v1/rto/estimate.
type EstimateRequest struct {
	ListingID             string   `json:"listing_id"`
	ListPrice             *float64 `json:"list_price"`
	OptionTermYears       *int     `json:"option_term_years"`
	AnnualAppreciationPct *float64 `json:"annual_appreciation_pct"`
	YieldPct              *float64 `json:"yield_pct"`
	PremiumPct            *float64 `json:"premium_pct"`
	InterestRatePct       *float64 `json:"interest_rate_pct"`
	LoanTermYears         *int     `json:"loan_term_years"`
	DownPaymentPct        *float64 `json:"down_payment_pct"`
	TaxRatePct            *float64 `json:"tax_rate_pct"`
	InsuranceRatePct      *float64 `json:"insurance_rate_pct"`
}

// EstimateResponse is the top-level JSON response.
type EstimateResponse struct {
	ListingID    string       `json:"listing_id,omitempty"`
	ListPrice    float64      `json:"list_price"`
	ModelVersion string       `json:"model_version"`
	Models       ModelsOutput `json:"models"`
	Disclaimer   string       `json:"disclaimer"`
}

// ModelsOutput holds the three model results.
type ModelsOutput struct {
	YieldBased  YieldResult  `json:"yield_based"`
	PITIPremium PITIResult   `json:"piti_premium"`
	Hybrid      HybridResult `json:"hybrid"`
}

// YieldResult is the yield-based model output.
type YieldResult struct {
	EstimatedMonthly       float64 `json:"estimated_monthly"`
	EstimatedPurchasePrice float64 `json:"estimated_purchase_price"`
}

// PITIResult is the PITI+premium model output.
type PITIResult struct {
	EstimatedMonthly       float64       `json:"estimated_monthly"`
	EstimatedPurchasePrice float64       `json:"estimated_purchase_price"`
	Breakdown              PITIBreakdown `json:"breakdown"`
}

// PITIBreakdown shows the components of the PITI model.
type PITIBreakdown struct {
	PrincipalInterest float64 `json:"principal_interest"`
	Taxes             float64 `json:"taxes"`
	Insurance         float64 `json:"insurance"`
	BasePITI          float64 `json:"base_piti"`
	Premium           float64 `json:"premium"`
}

// HybridResult is the weighted-blend model output.
type HybridResult struct {
	EstimatedMonthly       float64 `json:"estimated_monthly"`
	EstimatedPurchasePrice float64 `json:"estimated_purchase_price"`
	Recommended            bool    `json:"recommended"`
}

// --- Defaults ---

const (
	defaultOptionTermYears       = 3
	defaultAnnualAppreciationPct = 3.0
	defaultYieldPct              = 0.8
	defaultPremiumPct            = 15.0
	defaultInterestRatePct       = 7.5
	defaultLoanTermYears         = 30
	defaultDownPaymentPct        = 5.0
	defaultInsuranceRatePct      = 0.35
	defaultTaxRatePct            = 1.2

	hybridYieldWeight = 0.4
	hybridPITIWeight  = 0.6

	modelVersion = "1.0.0"
	disclaimer   = "Estimates only. Not a binding offer. Actual terms may vary."
)

// resolvedInputs holds all parameters after applying defaults.
type resolvedInputs struct {
	ListPrice             float64
	OptionTermYears       int
	AnnualAppreciationPct float64
	YieldPct              float64
	PremiumPct            float64
	InterestRatePct       float64
	LoanTermYears         int
	DownPaymentPct        float64
	TaxRatePct            float64
	InsuranceRatePct      float64
}

func resolveInputs(req EstimateRequest, listPrice float64, taxRatePct *float64) resolvedInputs {
	ri := resolvedInputs{
		ListPrice:             listPrice,
		OptionTermYears:       defaultOptionTermYears,
		AnnualAppreciationPct: defaultAnnualAppreciationPct,
		YieldPct:              defaultYieldPct,
		PremiumPct:            defaultPremiumPct,
		InterestRatePct:       defaultInterestRatePct,
		LoanTermYears:         defaultLoanTermYears,
		DownPaymentPct:        defaultDownPaymentPct,
		TaxRatePct:            defaultTaxRatePct,
		InsuranceRatePct:      defaultInsuranceRatePct,
	}
	if req.OptionTermYears != nil {
		ri.OptionTermYears = *req.OptionTermYears
	}
	if req.AnnualAppreciationPct != nil {
		ri.AnnualAppreciationPct = *req.AnnualAppreciationPct
	}
	if req.YieldPct != nil {
		ri.YieldPct = *req.YieldPct
	}
	if req.PremiumPct != nil {
		ri.PremiumPct = *req.PremiumPct
	}
	if req.InterestRatePct != nil {
		ri.InterestRatePct = *req.InterestRatePct
	}
	if req.LoanTermYears != nil {
		ri.LoanTermYears = *req.LoanTermYears
	}
	if req.DownPaymentPct != nil {
		ri.DownPaymentPct = *req.DownPaymentPct
	}
	if req.InsuranceRatePct != nil {
		ri.InsuranceRatePct = *req.InsuranceRatePct
	}
	// Tax rate: prefer request override, then DB-derived value, then default.
	if req.TaxRatePct != nil {
		ri.TaxRatePct = *req.TaxRatePct
	} else if taxRatePct != nil {
		ri.TaxRatePct = *taxRatePct
	}
	return ri
}

// --- Purchase price ---

// computePurchasePrice applies compound annual appreciation over the option term.
func computePurchasePrice(listPrice float64, annualAppreciationPct float64, termYears int) float64 {
	return round2(listPrice * math.Pow(1+annualAppreciationPct/100, float64(termYears)))
}

// --- Model 1: Yield-based ---

func computeYield(ri resolvedInputs) YieldResult {
	monthly := round2(ri.ListPrice * (ri.YieldPct / 100) / 12)
	purchasePrice := computePurchasePrice(ri.ListPrice, ri.AnnualAppreciationPct, ri.OptionTermYears)
	return YieldResult{
		EstimatedMonthly:       monthly,
		EstimatedPurchasePrice: purchasePrice,
	}
}

// --- Model 2: PITI + Premium ---

// computeMonthlyPI returns the monthly principal + interest payment.
func computeMonthlyPI(loanAmount, annualRate float64, termYears int) float64 {
	if termYears <= 0 {
		return 0
	}
	if annualRate == 0 {
		return loanAmount / float64(termYears*12)
	}
	monthlyRate := annualRate / 12
	n := float64(termYears * 12)
	return loanAmount * monthlyRate * math.Pow(1+monthlyRate, n) / (math.Pow(1+monthlyRate, n) - 1)
}

func computePITI(ri resolvedInputs) PITIResult {
	purchasePrice := computePurchasePrice(ri.ListPrice, ri.AnnualAppreciationPct, ri.OptionTermYears)
	downPayment := purchasePrice * (ri.DownPaymentPct / 100)
	loanAmount := purchasePrice - downPayment

	pi := computeMonthlyPI(loanAmount, ri.InterestRatePct/100, ri.LoanTermYears)
	taxes := ri.ListPrice * (ri.TaxRatePct / 100) / 12
	insurance := ri.ListPrice * (ri.InsuranceRatePct / 100) / 12
	basePITI := pi + taxes + insurance
	premium := basePITI * (ri.PremiumPct / 100)
	monthly := round2(basePITI + premium)

	return PITIResult{
		EstimatedMonthly:       monthly,
		EstimatedPurchasePrice: purchasePrice,
		Breakdown: PITIBreakdown{
			PrincipalInterest: round2(pi),
			Taxes:             round2(taxes),
			Insurance:         round2(insurance),
			BasePITI:          round2(basePITI),
			Premium:           round2(premium),
		},
	}
}

// --- Model 3: Hybrid ---

func computeHybrid(yieldMonthly, pitiMonthly float64, purchasePrice float64) HybridResult {
	monthly := round2(yieldMonthly*hybridYieldWeight + pitiMonthly*hybridPITIWeight)
	return HybridResult{
		EstimatedMonthly:       monthly,
		EstimatedPurchasePrice: purchasePrice,
		Recommended:            true,
	}
}

// --- Orchestrator ---

func computeAllModels(ri resolvedInputs) ModelsOutput {
	yield := computeYield(ri)
	piti := computePITI(ri)
	hybrid := computeHybrid(yield.EstimatedMonthly, piti.EstimatedMonthly, piti.EstimatedPurchasePrice)
	return ModelsOutput{
		YieldBased:  yield,
		PITIPremium: piti,
		Hybrid:      hybrid,
	}
}

// --- Exported for search integration ---

// ComputeHybridForSearch returns the recommended hybrid RTO estimate.
// taxAnnualAmount is the per-listing tax from the DB (nullable).
// Uses max(default 1.2% rate, DB-derived rate) to never underestimate taxes.
func ComputeHybridForSearch(listPrice float64, taxAnnualAmount *float64) HybridResult {
	taxRate := defaultTaxRatePct // 1.2
	if taxAnnualAmount != nil && *taxAnnualAmount > 0 && listPrice > 0 {
		dbRate := (*taxAnnualAmount / listPrice) * 100
		if dbRate > taxRate {
			taxRate = dbRate
		}
	}
	ri := resolvedInputs{
		ListPrice:             listPrice,
		OptionTermYears:       defaultOptionTermYears,
		AnnualAppreciationPct: defaultAnnualAppreciationPct,
		YieldPct:              defaultYieldPct,
		PremiumPct:            defaultPremiumPct,
		InterestRatePct:       defaultInterestRatePct,
		LoanTermYears:         defaultLoanTermYears,
		DownPaymentPct:        defaultDownPaymentPct,
		TaxRatePct:            taxRate,
		InsuranceRatePct:      defaultInsuranceRatePct,
	}
	yieldRes := computeYield(ri)
	pitiRes := computePITI(ri)
	return computeHybrid(yieldRes.EstimatedMonthly, pitiRes.EstimatedMonthly, pitiRes.EstimatedPurchasePrice)
}

// --- Helpers ---

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
