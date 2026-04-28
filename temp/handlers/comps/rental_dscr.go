package comps

// computeQualifyingMonthlyBreakdown computes a monthly breakdown using the qualifying
// rent (market_rent * qualifying_percent) as income, but computing variable expenses
// (vacancy, management, maintenance, capex) against market rent.
func computeQualifyingMonthlyBreakdown(
	marketRent float64,
	qualifyingRent float64,
	params *RentalParamsInput,
	ownership OwnershipInput,
	loan LoanSummary,
) MonthlyBreakdown {
	// Income uses qualifying rent.
	vacancy := marketRent * params.vacancyPercent()
	egi := qualifyingRent - vacancy

	// Variable expenses computed on market rent.
	management := marketRent * params.propertyManagementPercent()
	maintenance := marketRent * params.maintenancePercent()
	capex := marketRent * params.capexPercent()

	// Fixed expenses.
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
		GrossRent:          round2(qualifyingRent),
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

// computeMonthlyIO returns the monthly interest-only payment.
func computeMonthlyIO(loanAmount, annualRate float64) float64 {
	return loanAmount * annualRate / 12
}

// computeDSCROverlay computes the DSCR lender qualification overlay.
func computeDSCROverlay(
	noi float64,
	loan LoanSummary,
	financing FinancingInput,
	cfg *DSCRConfig,
) DSCROverlay {
	threshold := cfg.minThreshold()

	// Determine debt service.
	var debtService float64
	var dsUsed string
	if cfg.ioOptionEnabled() {
		debtService = round2(computeMonthlyIO(loan.LoanAmount, financing.InterestRate))
		dsUsed = "IO"
	} else {
		debtService = loan.MonthlyPI + loan.MonthlyPMI
		dsUsed = "PI"
	}

	var dscr float64
	if debtService > 0 {
		dscr = noi / debtService
	}

	pass := dscr >= threshold
	viewUsed := cfg.qualifyUses()
	var notes []string

	// Stress test.
	var stressedRate *float64
	var stressedDSCR *float64
	bps := cfg.stressRateBPS()
	if bps > 0 {
		sr := financing.InterestRate + float64(bps)/10000
		stressedRate = &sr

		var stressDS float64
		if cfg.ioOptionEnabled() {
			stressDS = computeMonthlyIO(loan.LoanAmount, sr)
		} else {
			stressDS = computeMonthlyPI(loan.LoanAmount, sr, financing.LoanTermYears)
		}

		var sd float64
		if stressDS > 0 {
			sd = noi / stressDS
		}
		stressedDSCR = &sd

		if sd < threshold {
			pass = false
			notes = append(notes, "Fails at stressed rate")
		}
	}

	if dscr < threshold {
		notes = append(notes, "DSCR below minimum threshold")
	}

	if notes == nil {
		notes = []string{}
	}

	return DSCROverlay{
		DSCRValue:          round2(dscr),
		DSCRThreshold:      threshold,
		Pass:               pass,
		QualifyViewUsed:    viewUsed,
		DebtServiceUsed:    dsUsed,
		DebtServiceMonthly: round2(debtService),
		StressedRate:       stressedRate,
		StressedDSCR:       stressedDSCR,
		Notes:              notes,
	}
}

// computeSelfSufficiency tests whether qualifying rent covers PITIA obligations.
func computeSelfSufficiency(
	qualifyingRent float64,
	loan LoanSummary,
	ownership OwnershipInput,
	cfg *DSCRConfig,
) SelfSufficiencyResult {
	qualPct := cfg.rentQualifyingPercent()
	qualIncome := qualifyingRent

	// Build PITIA.
	pitia := loan.MonthlyPI + loan.MonthlyPMI
	if cfg.includeTaxesInsurance() {
		pitia += ownership.AnnualPropertyTaxes / 12
		pitia += ownership.AnnualHomeownersInsurance / 12
		pitia += ownership.floodInsuranceAnnual() / 12
	}
	if cfg.includeHOA() {
		pitia += ownership.monthlyHOA()
	}

	ratio := cfg.selfSufficiencyRatio()
	surplus := qualIncome - pitia*ratio
	pass := surplus >= 0

	return SelfSufficiencyResult{
		RentQualifyingPercent:   qualPct,
		QualifyingIncomeMonthly: round2(qualIncome),
		PITIAMonthly:            round2(pitia),
		SurplusDeficitMonthly:   round2(surplus),
		Pass:                    pass,
		Rule:                    cfg.selfSufficiencyRule(),
	}
}
