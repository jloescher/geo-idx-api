package comps

import "math"

// computeARV computes the After-Repair Value from sold comps.
// Primary method: central tendency of adjusted prices. Fallback: PPSF * subject sqft.
func computeARV(soldRows []compRow, subject *resolvedSubject, filters *FiltersInput, method AggregationMethod) (arv float64, arvMethod string, compCount int) {
	if len(soldRows) == 0 {
		return 0, "no_comps", 0
	}

	// Compute PPSF for adjustment grid.
	var ppsfs []float64
	for _, r := range soldRows {
		if r.ClosePrice != nil && r.LivingArea != nil && *r.LivingArea > 0 {
			ppsfs = append(ppsfs, *r.ClosePrice/float64(*r.LivingArea))
		}
	}
	centralPPSF := aggregate(ppsfs, method)

	// Compute adjusted prices, excluding high-adjustment comps if enough remain.
	var adjustedPrices []float64
	var allAdjusted []float64
	for _, r := range soldRows {
		grid := computeAdjustmentGrid(subject, r, centralPPSF, filters)
		if grid == nil {
			continue
		}
		allAdjusted = append(allAdjusted, grid.AdjustedPrice)
		if !grid.HighAdjWarning {
			adjustedPrices = append(adjustedPrices, grid.AdjustedPrice)
		}
	}

	// Use filtered set if at least 3 comps, otherwise use all.
	useSet := adjustedPrices
	if len(useSet) < 3 {
		useSet = allAdjusted
	}

	methodSuffix := "median"
	if method == AggAverage {
		methodSuffix = "average"
	}

	if len(useSet) > 0 {
		return round2(aggregate(useSet, method)), methodSuffix + "_adjusted_price", len(useSet)
	}

	// Fallback: PPSF * subject sqft.
	if centralPPSF > 0 && subject.LivingAreaSqft != nil && *subject.LivingAreaSqft > 0 {
		return round2(centralPPSF * float64(*subject.LivingAreaSqft)), methodSuffix + "_ppsf_x_sqft", len(ppsfs)
	}

	return 0, "no_comps", 0
}

// computeFlipSummary computes the complete flip financial analysis.
func computeFlipSummary(arv float64, arvMethod string, arvCompCount int, fp *FlipParamsInput, rp *RentalParamsInput) FlipSummary {
	purchasePrice := rp.Financing.PurchasePrice
	totalRepairs := fp.RepairBudget * (1 + fp.rehabContingencyPercent())
	buyClosingCosts := purchasePrice * fp.BuyCosts.closingCostsPercent()

	// Monthly carrying costs.
	taxes := rp.Ownership.AnnualPropertyTaxes / 12
	insurance := rp.Ownership.AnnualHomeownersInsurance / 12
	hoa := rp.Ownership.monthlyHOA()
	utilities := rp.Ownership.utilitiesMonthly()

	var financingMonthly float64
	if fp.FlipFinancing != nil {
		loanAmt := purchasePrice - computeFlipDownPayment(purchasePrice, fp)
		financingMonthly = loanAmt * fp.FlipFinancing.InterestRate / 12
	}

	monthlyCarrying := FlipMonthlyCarrying{
		Taxes:     round2(taxes),
		Insurance: round2(insurance),
		HOA:       round2(hoa),
		Utilities: round2(utilities),
		Financing: round2(financingMonthly),
		Total:     round2(taxes + insurance + hoa + utilities + financingMonthly),
	}

	months := fp.holdingPeriodMonths()
	carryingCosts := monthlyCarrying.Total * float64(months)

	// Selling costs.
	sellPct := fp.SellingCosts.agentCommissionPercent() + fp.SellingCosts.closingCostsPercent()
	sellingCosts := arv*sellPct + fp.SellingCosts.sellerConcessionsAmount()

	salePrice := arv
	totalCostBasis := purchasePrice + buyClosingCosts + totalRepairs + carryingCosts

	// Points cost.
	var pointsCost float64
	if fp.FlipFinancing != nil {
		loanAmt := purchasePrice - computeFlipDownPayment(purchasePrice, fp)
		pointsCost = loanAmt * fp.FlipFinancing.pointsPercent()
	}

	netProfit := salePrice - sellingCosts - totalCostBasis - pointsCost

	// Cash invested = down payment + buy closing + repairs + points + carrying (if no financing for carry).
	cashInvested := buyClosingCosts + totalRepairs + pointsCost
	if fp.FlipFinancing != nil {
		cashInvested += computeFlipDownPayment(purchasePrice, fp)
		cashInvested += carryingCosts
	} else {
		cashInvested += purchasePrice + carryingCosts
	}

	var roi float64
	if cashInvested > 0 {
		roi = netProfit / cashInvested
	}

	var annualizedROI float64
	if months > 0 && roi > -1 {
		annualizedROI = math.Pow(1+roi, 12/float64(months)) - 1
	}

	maxOffer := computeMaxOfferPrice(arv, fp, monthlyCarrying)
	var marginOfSafety float64
	if arv > 0 && maxOffer > 0 {
		marginOfSafety = (arv - maxOffer) / arv
	}

	assumptions := []string{
		"ARV based on comparable sold properties within search scope",
		"Holding period assumes " + itoa(months) + " months",
		"Carrying costs are constant over holding period",
	}
	if fp.FlipFinancing != nil {
		assumptions = append(assumptions, "Flip financing is interest-only during hold")
	}

	return FlipSummary{
		ARV:             round2(arv),
		ARVMethod:       arvMethod,
		ARVCompCount:    arvCompCount,
		PurchasePrice:   round2(purchasePrice),
		TotalRepairs:    round2(totalRepairs),
		BuyClosingCosts: round2(buyClosingCosts),
		CarryingCosts:   round2(carryingCosts),
		SellingCosts:    round2(sellingCosts),
		TotalCostBasis:  round2(totalCostBasis),
		SalePrice:       round2(salePrice),
		NetProfit:       round2(netProfit),
		CashInvested:    round2(cashInvested),
		ROI:             round4(roi),
		AnnualizedROI:   round4(annualizedROI),
		MaxOfferPrice:   round2(maxOffer),
		MarginOfSafety:  round4(marginOfSafety),
		MonthlyCarrying: monthlyCarrying,
		KeyAssumptions:  assumptions,
	}
}

// computeMaxOfferPrice solves for the maximum purchase price that achieves
// the desired profit target. Closed-form since profit is linear in purchase price.
func computeMaxOfferPrice(arv float64, fp *FlipParamsInput, monthlyCarrying FlipMonthlyCarrying) float64 {
	targetProfit := 0.0
	if fp.DesiredProfitAmount != nil {
		targetProfit = *fp.DesiredProfitAmount
	} else if fp.DesiredProfitPercentARV != nil {
		targetProfit = arv * *fp.DesiredProfitPercentARV
	} else {
		return 0
	}

	totalRepairs := fp.RepairBudget * (1 + fp.rehabContingencyPercent())
	months := fp.holdingPeriodMonths()

	// Carrying costs that don't depend on purchase price.
	fixedCarry := (monthlyCarrying.Taxes + monthlyCarrying.Insurance +
		monthlyCarrying.HOA + monthlyCarrying.Utilities) * float64(months)

	sellPct := fp.SellingCosts.agentCommissionPercent() + fp.SellingCosts.closingCostsPercent()
	concessions := fp.SellingCosts.sellerConcessionsAmount()
	netSale := arv*(1-sellPct) - concessions

	buyClosingPct := fp.BuyCosts.closingCostsPercent()

	// P * (1 + buyClosingPct) + totalRepairs + fixedCarry + financeCarry(P) + points(P) + targetProfit = netSale
	// financeCarry(P) = P * finRate/12 * months (if financed, simplified)
	// points(P) = P * pointsPct (if financed)
	var financePctOfP float64
	var pointsPctOfP float64
	if fp.FlipFinancing != nil {
		financePctOfP = fp.FlipFinancing.InterestRate / 12 * float64(months)
		pointsPctOfP = fp.FlipFinancing.pointsPercent()
	}

	// P * (1 + buyClosingPct + financePctOfP + pointsPctOfP) = netSale - totalRepairs - fixedCarry - targetProfit
	denom := 1 + buyClosingPct + financePctOfP + pointsPctOfP
	if denom <= 0 {
		return 0
	}

	maxP := (netSale - totalRepairs - fixedCarry - targetProfit) / denom
	if maxP < 0 {
		return 0
	}
	return round2(maxP)
}

// computeFlipVsHoldComparison builds boolean comparison flags between flip and hold strategies.
func computeFlipVsHoldComparison(flip FlipSummary, hold HoldSummary) FlipVsHoldComparison {
	flipProfitable := flip.NetProfit > 0
	holdCFPositive := hold.AnnualCashFlow > 0
	flipROIAbove := flip.ROI > 0.15
	holdDSCRPass := hold.DSCR >= 1.10
	flipHigherROI := flip.AnnualizedROI > hold.CashOnCash
	holdSelfSuff := hold.SelfSufficient

	bothViable := flipProfitable && holdCFPositive
	neitherViable := !flipProfitable && !holdCFPositive

	var explanation []string
	if flipProfitable {
		explanation = append(explanation, "Flip net profit: "+formatDollars(flip.NetProfit))
	} else {
		explanation = append(explanation, "Flip produces a loss of "+formatDollars(flip.NetProfit))
	}
	if holdCFPositive {
		explanation = append(explanation, "Hold annual cash flow: "+formatDollars(hold.AnnualCashFlow))
	} else {
		explanation = append(explanation, "Hold annual cash flow is negative: "+formatDollars(hold.AnnualCashFlow))
	}
	if flipHigherROI {
		explanation = append(explanation, "Flip annualized ROI "+formatPct(flip.AnnualizedROI)+" exceeds hold cash-on-cash "+formatPct(hold.CashOnCash))
	} else {
		explanation = append(explanation, "Hold cash-on-cash "+formatPct(hold.CashOnCash)+" exceeds flip annualized ROI "+formatPct(flip.AnnualizedROI))
	}

	return FlipVsHoldComparison{
		FlipProfitable:         flipProfitable,
		HoldCashFlowPositive:   holdCFPositive,
		FlipROIAboveThreshold:  flipROIAbove,
		HoldDSCRPass:           holdDSCRPass,
		FlipHigherShortTermROI: flipHigherROI,
		HoldSelfSufficient:     holdSelfSuff,
		BothViable:             bothViable,
		NeitherViable:          neitherViable,
		Explanation:            explanation,
	}
}

// computeFlipDownPayment returns the flip down payment amount.
// If no financing, returns 0 (full cash purchase handled separately).
func computeFlipDownPayment(purchasePrice float64, fp *FlipParamsInput) float64 {
	if fp.FlipFinancing == nil {
		return 0
	}
	// For flip financing, assume full loan (no down payment) unless
	// purchase price structure suggests otherwise. Hard money typically
	// finances 100% of purchase.
	return 0
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}

func formatDollars(v float64) string {
	if v < 0 {
		return "-$" + formatAbsDollars(-v)
	}
	return "$" + formatAbsDollars(v)
}

func formatAbsDollars(v float64) string {
	rounded := int(math.Round(v))
	s := itoa(rounded)
	// Add comma separators.
	if len(s) <= 3 {
		return s
	}
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

func formatDollarsSigned(v float64) string {
	if v >= 0 {
		return "+$" + formatAbsDollars(v)
	}
	return "-$" + formatAbsDollars(-v)
}

func formatPct(v float64) string {
	rounded := math.Round(v*1000) / 10 // one decimal
	s := itoa(int(rounded))
	frac := int(math.Round(math.Abs(rounded)*10)) % 10
	return s + "." + itoa(frac) + "%"
}
