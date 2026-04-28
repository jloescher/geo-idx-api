package comps

import "math"

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

// computeDownPayment returns the down payment amount and percentage.
func computeDownPayment(financing FinancingInput) (amount, pct float64) {
	if financing.DownPaymentAmount != nil {
		amount = *financing.DownPaymentAmount
		if financing.PurchasePrice > 0 {
			pct = amount / financing.PurchasePrice
		}
		return
	}
	if financing.DownPaymentPercent != nil {
		pct = *financing.DownPaymentPercent
		amount = financing.PurchasePrice * pct
		return
	}
	return 0, 0
}

// computePMI returns the monthly PMI amount.
func computePMI(financing FinancingInput, loanAmount float64) float64 {
	if financing.PMIMonthly != nil {
		return *financing.PMIMonthly
	}
	dpAmount, _ := computeDownPayment(financing)
	if financing.PurchasePrice > 0 && dpAmount/financing.PurchasePrice < 0.20 {
		return (loanAmount * financing.pmiRateAnnual()) / 12
	}
	return 0
}

// computeLoanSummary computes all loan-related figures from financing inputs.
func computeLoanSummary(financing FinancingInput) LoanSummary {
	dpAmount, dpPct := computeDownPayment(financing)
	loanAmount := financing.PurchasePrice - dpAmount
	monthlyPI := computeMonthlyPI(loanAmount, financing.InterestRate, financing.LoanTermYears)
	monthlyPMI := computePMI(financing, loanAmount)
	closingCosts := financing.closingCosts()
	cashInvested := dpAmount + closingCosts

	return LoanSummary{
		PurchasePrice:  financing.PurchasePrice,
		DownPayment:    dpAmount,
		DownPaymentPct: dpPct,
		LoanAmount:     loanAmount,
		InterestRate:   financing.InterestRate,
		LoanTermYears:  financing.LoanTermYears,
		MonthlyPI:      round2(monthlyPI),
		MonthlyPMI:     round2(monthlyPMI),
		ClosingCosts:   closingCosts,
		CashInvested:   cashInvested,
	}
}

// computeMonthlyBreakdown computes the full monthly cash flow breakdown.
func computeMonthlyBreakdown(grossRent float64, params *RentalParamsInput, ownership OwnershipInput, loan LoanSummary) MonthlyBreakdown {
	vacancy := grossRent * params.vacancyPercent()
	egi := grossRent - vacancy
	management := grossRent * params.propertyManagementPercent()
	maintenance := grossRent * params.maintenancePercent()
	capex := grossRent * params.capexPercent()
	hoa := ownership.monthlyHOA()
	taxes := ownership.AnnualPropertyTaxes / 12
	insurance := ownership.AnnualHomeownersInsurance / 12
	flood := ownership.floodInsuranceAnnual() / 12
	utilities := ownership.utilitiesMonthly()
	other := ownership.otherMonthly()
	totalOperating := hoa + taxes + insurance + flood + utilities + other
	noi := egi - (totalOperating + management + maintenance + capex)
	debtService := loan.MonthlyPI + loan.MonthlyPMI
	cashFlow := noi - debtService

	return MonthlyBreakdown{
		GrossRent:          round2(grossRent),
		Vacancy:            round2(vacancy),
		EGI:                round2(egi),
		PropertyManagement: round2(management),
		Maintenance:        round2(maintenance),
		CapEx:              round2(capex),
		HOA:                round2(hoa),
		Taxes:              round2(taxes),
		Insurance:          round2(insurance),
		FloodInsurance:     round2(flood),
		Utilities:          round2(utilities),
		Other:              round2(other),
		TotalOperating:     round2(totalOperating),
		NOI:                round2(noi),
		DebtService:        round2(debtService),
		CashFlow:           round2(cashFlow),
	}
}

// computeAnnualMetrics derives annual metrics from the monthly breakdown.
func computeAnnualMetrics(monthly MonthlyBreakdown, loan LoanSummary) AnnualMetrics {
	annualNOI := monthly.NOI * 12
	annualCashFlow := monthly.CashFlow * 12

	var cashOnCash float64
	if loan.CashInvested > 0 {
		cashOnCash = annualCashFlow / loan.CashInvested
	}

	var dscr float64
	if monthly.DebtService > 0 {
		dscr = monthly.NOI / monthly.DebtService
	}

	var capRate float64
	if loan.PurchasePrice > 0 {
		capRate = annualNOI / loan.PurchasePrice
	}

	return AnnualMetrics{
		AnnualNOI:      round2(annualNOI),
		AnnualCashFlow: round2(annualCashFlow),
		CashOnCash:     round4(cashOnCash),
		DSCR:           round2(dscr),
		CapRate:        round4(capRate),
	}
}

// computeScenarios runs cash flow analysis at three rent levels.
func computeScenarios(rentLow, rentBase, rentHigh float64, params *RentalParamsInput, ownership OwnershipInput, loan LoanSummary) ScenarioOutput {
	return ScenarioOutput{
		Conservative: buildScenario(rentLow, params, ownership, loan),
		Base:         buildScenario(rentBase, params, ownership, loan),
		Upside:       buildScenario(rentHigh, params, ownership, loan),
	}
}

func buildScenario(rent float64, params *RentalParamsInput, ownership OwnershipInput, loan LoanSummary) ScenarioResult {
	m := computeMonthlyBreakdown(rent, params, ownership, loan)
	a := computeAnnualMetrics(m, loan)
	return ScenarioResult{
		MonthlyRent:     round2(rent),
		MonthlyCashFlow: m.CashFlow,
		AnnualCashFlow:  a.AnnualCashFlow,
		CashOnCash:      a.CashOnCash,
		DSCR:            a.DSCR,
	}
}

// computeScenarioFlags evaluates pass/fail conditions on the base and conservative scenarios.
func computeScenarioFlags(scenarios ScenarioOutput) ScenarioFlags {
	return ScenarioFlags{
		CashFlowPositive:             scenarios.Base.AnnualCashFlow > 0,
		DSCRPass:                     scenarios.Base.DSCR >= 1.10,
		ConservativeCashFlowPositive: scenarios.Conservative.AnnualCashFlow > 0,
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}
