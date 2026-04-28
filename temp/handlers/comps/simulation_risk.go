package comps

import (
	"fmt"
	"math"
)

// computeValuationRisk computes the 7-component valuation risk score.
// Only called when a contract price is provided.
func computeValuationRisk(contractPrice, simulatedValue float64, comps []rankedSimComp, concessionPct, quarterlyTrend float64, subject *resolvedSubject) *ValuationRiskScore {
	if contractPrice <= 0 {
		return nil
	}

	gapAmount := contractPrice - simulatedValue
	gapPercent := 0.0
	if contractPrice > 0 {
		gapPercent = gapAmount / contractPrice * 100
	}

	components := make([]RiskComponent, 7)

	// 1. Gap Magnitude (weight 0.40).
	components[0] = gapMagnitudeComponent(gapPercent)

	// 2. Adjustment Dispersion (weight 0.15).
	components[1] = adjustmentDispersionComponent(comps)

	// 3. Concession Risk (weight 0.10).
	components[2] = concessionRiskComponent(concessionPct)

	// 4. Bedroom Mismatch (weight 0.10).
	components[3] = bedroomMismatchComponent(comps)

	// 5. Market Volatility (weight 0.10).
	components[4] = marketVolatilityComponent(quarterlyTrend)

	// 6. Limited Comps (weight 0.10).
	components[5] = limitedCompsComponent(len(comps))

	// 7. Subtype Mismatch (weight 0.05).
	components[6] = subtypeMismatchComponent(comps)

	// Composite score.
	composite := 0.0
	for _, c := range components {
		composite += c.Weighted
	}
	composite = clamp(composite, 0, 100)

	band := riskBand(composite)

	actions := recommendedActions(components)

	return &ValuationRiskScore{
		Score:              math.Round(composite*10) / 10,
		RiskBand:           band,
		GapAmount:          math.Round(gapAmount),
		GapPercent:         math.Round(gapPercent*10) / 10,
		Components:         components,
		RecommendedActions: actions,
	}
}

func gapMagnitudeComponent(gapPct float64) RiskComponent {
	var raw float64
	var detail string
	switch {
	case gapPct <= 0:
		raw = 0
		detail = "Simulated value meets or exceeds contract price"
	case gapPct <= 3:
		raw = 15
		detail = fmt.Sprintf("Gap of %.1f%% is within normal tolerance", gapPct)
	case gapPct <= 5:
		raw = 35
		detail = fmt.Sprintf("Gap of %.1f%% may trigger lender review", gapPct)
	case gapPct <= 10:
		raw = 65
		detail = fmt.Sprintf("Gap of %.1f%% likely to result in below-contract valuation", gapPct)
	default:
		raw = 90
		detail = fmt.Sprintf("Gap of %.1f%% is significantly above simulated value", gapPct)
	}
	w := 0.40
	return RiskComponent{
		Name:     "Gap Magnitude",
		RawScore: raw,
		Weight:   w,
		Weighted: raw * w,
		Detail:   detail,
	}
}

func adjustmentDispersionComponent(comps []rankedSimComp) RiskComponent {
	w := 0.15
	if len(comps) == 0 {
		return RiskComponent{Name: "Adjustment Dispersion", RawScore: 100, Weight: w, Weighted: 100 * w, Detail: "No comps available"}
	}
	var totalAdj float64
	for _, c := range comps {
		totalAdj += c.fullGrossAdjPct
	}
	avg := totalAdj / float64(len(comps))
	raw := math.Min(avg*4, 100)
	detail := fmt.Sprintf("Average gross adjustment: %.1f%%", avg)
	return RiskComponent{
		Name:     "Adjustment Dispersion",
		RawScore: raw,
		Weight:   w,
		Weighted: raw * w,
		Detail:   detail,
	}
}

func concessionRiskComponent(concessionPct float64) RiskComponent {
	w := 0.10
	var raw float64
	var detail string
	switch {
	case concessionPct <= 2:
		raw = 10
		detail = fmt.Sprintf("Concessions at %.1f%% are within typical range", concessionPct)
	case concessionPct <= 6:
		raw = 40
		detail = fmt.Sprintf("Concessions at %.1f%% may draw appraiser scrutiny", concessionPct)
	default:
		raw = 80
		detail = fmt.Sprintf("Concessions at %.1f%% are above typical market norms", concessionPct)
	}
	return RiskComponent{
		Name:     "Concession Risk",
		RawScore: raw,
		Weight:   w,
		Weighted: raw * w,
		Detail:   detail,
	}
}

func bedroomMismatchComponent(comps []rankedSimComp) RiskComponent {
	w := 0.10
	if len(comps) == 0 {
		return RiskComponent{Name: "Bedroom Mismatch", RawScore: 0, Weight: w, Weighted: 0, Detail: "No comps to evaluate"}
	}
	maxDiff := 0
	for _, c := range comps {
		if c.bedroomMismatch > maxDiff {
			maxDiff = c.bedroomMismatch
		}
	}
	var raw float64
	switch {
	case maxDiff == 0:
		raw = 0
	case maxDiff == 1:
		raw = 25
	case maxDiff == 2:
		raw = 60
	default:
		raw = 95
	}
	detail := fmt.Sprintf("Maximum bedroom difference: %d", maxDiff)
	return RiskComponent{
		Name:     "Bedroom Mismatch",
		RawScore: raw,
		Weight:   w,
		Weighted: raw * w,
		Detail:   detail,
	}
}

func marketVolatilityComponent(quarterlyTrend float64) RiskComponent {
	w := 0.10
	absQ := math.Abs(quarterlyTrend) * 100 // convert to percentage
	var raw float64
	switch {
	case absQ <= 1:
		raw = 5
	case absQ <= 3:
		raw = 20
	case absQ <= 5:
		raw = 50
	default:
		raw = 80
	}
	detail := fmt.Sprintf("Quarterly trend: %.1f%%", quarterlyTrend*100)
	return RiskComponent{
		Name:     "Market Volatility",
		RawScore: raw,
		Weight:   w,
		Weighted: raw * w,
		Detail:   detail,
	}
}

func limitedCompsComponent(count int) RiskComponent {
	w := 0.10
	var raw float64
	switch {
	case count >= 4:
		raw = 5
	case count == 3:
		raw = 20
	case count == 2:
		raw = 55
	case count == 1:
		raw = 85
	default:
		raw = 100
	}
	detail := fmt.Sprintf("%d comparable(s) selected", count)
	return RiskComponent{
		Name:     "Limited Comps",
		RawScore: raw,
		Weight:   w,
		Weighted: raw * w,
		Detail:   detail,
	}
}

func subtypeMismatchComponent(comps []rankedSimComp) RiskComponent {
	w := 0.05
	if len(comps) == 0 {
		return RiskComponent{Name: "Subtype Mismatch", RawScore: 0, Weight: w, Weighted: 0, Detail: "No comps to evaluate"}
	}
	matchCount := 0
	for _, c := range comps {
		if c.subTypeMatch {
			matchCount++
		}
	}
	raw := (1.0 - float64(matchCount)/float64(len(comps))) * 100
	detail := fmt.Sprintf("%d of %d comps match property sub-type", matchCount, len(comps))
	return RiskComponent{
		Name:     "Subtype Mismatch",
		RawScore: raw,
		Weight:   w,
		Weighted: raw * w,
		Detail:   detail,
	}
}

func riskBand(score float64) string {
	switch {
	case score <= 25:
		return "Low"
	case score <= 50:
		return "Moderate"
	case score <= 75:
		return "Elevated"
	default:
		return "High"
	}
}

func recommendedActions(components []RiskComponent) []string {
	var actions []string
	threshold := 30.0 // only recommend for components scoring above this

	for _, c := range components {
		if c.RawScore < threshold {
			continue
		}
		switch c.Name {
		case "Gap Magnitude":
			actions = append(actions, "Consider price reduction or renegotiation")
		case "Adjustment Dispersion":
			actions = append(actions, "Request additional comparable sales data")
		case "Concession Risk":
			actions = append(actions, "Verify concession structure with lender requirements")
		case "Bedroom Mismatch":
			actions = append(actions, "Source comps with matching bedroom count")
		case "Market Volatility":
			actions = append(actions, "Obtain current market trend data")
		case "Limited Comps":
			actions = append(actions, "Expand geographic search area")
		case "Subtype Mismatch":
			actions = append(actions, "Ensure comps match property sub-type")
		}
	}

	if len(actions) == 0 {
		actions = []string{}
	}
	return actions
}
