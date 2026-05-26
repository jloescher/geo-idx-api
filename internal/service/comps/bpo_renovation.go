package comps

import "time"

const (
	renovationSourceMarket  = "market_derived"
	renovationSourceDefault = "default_floor"
)

// renovationCredits returns subject-side value adds from recent kitchen/bath/HVAC work.
func renovationCredits(subject SubjectProfile, rates BpoMarketRates) []RenovationCredit {
	typicalValue := typicalHomeValue(rates)
	kitchenBase := marketRenovationAmount(typicalValue, 0.02, 8000)
	bathBase := marketRenovationAmount(typicalValue, 0.015, 6000)
	hvacBase := marketRenovationAmount(typicalValue, 0.01, 4000)
	if rates.AgePerYear > 500 {
		hvacBase = marketRenovationAmount(typicalValue, rates.AgePerYear/75000, 4000)
	}

	var credits []RenovationCredit
	if y := subject.RenovatedKitchenYear; y > 0 {
		if amt, ok := depreciatedCredit(y, kitchenBase); ok {
			credits = append(credits, RenovationCredit{
				Feature: "renovated_kitchen", Year: y, Amount: amt,
				RateSource: renovationSourceFor(amt, kitchenBase),
				Reasoning:  "Kitchen renovation credit (market-derived, age-depreciated)",
			})
		}
	}
	if y := subject.RenovatedBathroomsYear; y > 0 {
		if amt, ok := depreciatedCredit(y, bathBase); ok {
			credits = append(credits, RenovationCredit{
				Feature: "renovated_bathrooms", Year: y, Amount: amt,
				RateSource: renovationSourceFor(amt, bathBase),
				Reasoning:  "Bathroom renovation credit (market-derived, age-depreciated)",
			})
		}
	}
	if y := subject.RenovatedHVACYear; y > 0 {
		if amt, ok := depreciatedCredit(y, hvacBase); ok {
			credits = append(credits, RenovationCredit{
				Feature: "renovated_hvac", Year: y, Amount: amt,
				RateSource: renovationSourceFor(amt, hvacBase),
				Reasoning:  "HVAC replacement credit (market-derived, age-depreciated)",
			})
		}
	}
	return credits
}

func typicalHomeValue(rates BpoMarketRates) float64 {
	if rates.MedianGLA > 0 && rates.GLAPerSF > 0 {
		return rates.MedianGLA * rates.GLAPerSF
	}
	if rates.Intercept > 0 {
		return rates.Intercept
	}
	return 350000
}

func marketRenovationAmount(typicalValue, pct, floorDefault float64) float64 {
	derived := typicalValue * pct
	floor := floorDefault * 0.5
	if derived < floor {
		return floor
	}
	return derived
}

func depreciatedCredit(renovationYear int, fullCredit float64) (float64, bool) {
	age := time.Now().Year() - renovationYear
	switch {
	case age < 0:
		return 0, false
	case age <= 5:
		return round2(fullCredit), true
	case age <= 10:
		return round2(fullCredit * 0.5), true
	default:
		return 0, false
	}
}

func renovationSourceFor(amt, base float64) string {
	if amt >= base*0.45 {
		return renovationSourceMarket
	}
	return renovationSourceDefault
}

func sumRenovationCredits(credits []RenovationCredit) float64 {
	sum := 0.0
	for _, c := range credits {
		sum += c.Amount
	}
	return sum
}
