package comps

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// urarGridFeatures is the fixed 14-line URAR sales comparison grid (Laravel BpoAdjustmentEngine parity).
var urarGridFeatures = []string{
	"time_of_sale", "location", "site_lot", "design_style", "quality",
	"age_condition", "gla", "bedrooms", "bathrooms", "garage",
	"pool", "waterfront", "porch_patio", "hvac",
}

func buildURARGrid(subject SubjectProfile, comp CompRecord, rates BpoMarketRates) []BpoGridLine {
	src := rates.Method
	if src == "" {
		src = bpoRateSourceMedian
	}
	lines := make([]BpoGridLine, 0, 14)
	for _, feat := range urarGridFeatures {
		line := bpoGridLine(subject, comp, rates, feat, src)
		lines = append(lines, line)
	}
	return lines
}

func bpoGridLine(subject SubjectProfile, comp CompRecord, rates BpoMarketRates, feat, src string) BpoGridLine {
	switch feat {
	case "time_of_sale":
		mo := monthsSinceClose(comp.CloseDate)
		monthly := rates.TimePerMonthPct
		if monthly == 0 {
			monthly = 0.0025
		}
		if monthly < 0 {
			monthly = -monthly
		}
		adj := comp.ClosePrice * monthly * mo
		return BpoGridLine{
			Feature: "time_of_sale", CompValue: fmt.Sprintf("%.1f months ago", mo), Unit: "months",
			RateSource: src, RatePerUnit: monthly,
			Adjustment: round2(adj), Reasoning: "Time adjustment to effective date (older sales adjusted up when market appreciates)",
		}
	case "location":
		adj := locationAdjustment(subject, comp)
		return BpoGridLine{
			Feature: "location", SubjectValue: subject.Subdivision, CompValue: compSubdivision(comp),
			RateSource: "qualitative_match", Adjustment: round2(adj),
			Reasoning: "Subdivision / MLS area match",
		}
	case "site_lot":
		delta := subject.LotSizeAcres - comp.LotSizeAcres
		return BpoGridLine{
			Feature: "site_lot", SubjectValue: subject.LotSizeAcres, CompValue: comp.LotSizeAcres, Unit: "acres",
			RateSource: src, RatePerUnit: rates.LotPerAcre,
			Adjustment: round2(delta * rates.LotPerAcre), Reasoning: "Lot size differential x $/acre",
		}
	case "design_style":
		return BpoGridLine{
			Feature: "design_style", SubjectValue: subject.PropertyType, CompValue: compPropertyType(comp),
			RateSource: "qualitative_match", Adjustment: 0,
			Reasoning: "Property type match (insufficient subtype variance in comp set)",
		}
	case "quality":
		adj := qualityPPSFAdjustment(subject, comp, rates)
		return BpoGridLine{
			Feature: "quality", RateSource: src, RatePerUnit: rates.GLAPerSF,
			Adjustment: round2(adj), Reasoning: "PPSF tercile quality tier delta",
		}
	case "age_condition":
		subEff := effectiveYearBuilt(subject)
		compEff := float64(comp.YearBuilt)
		delta := subEff - compEff
		return BpoGridLine{
			Feature: "age_condition", SubjectValue: subEff, CompValue: comp.YearBuilt, Unit: "year",
			RateSource: src, RatePerUnit: rates.AgePerYear,
			Adjustment: round2(delta * rates.AgePerYear), Reasoning: "Effective age (condition overlay) x $/year",
		}
	case "gla":
		delta := subject.LivingArea - comp.LivingArea
		return BpoGridLine{
			Feature: "gla", SubjectValue: subject.LivingArea, CompValue: comp.LivingArea, Unit: "sqft",
			RateSource: src, RatePerUnit: rates.GLAPerSF,
			Adjustment: round2(delta * rates.GLAPerSF), Reasoning: "GLA delta x market-derived $/sf",
		}
	case "bedrooms":
		delta := subject.Bedrooms - comp.Bedrooms
		return BpoGridLine{
			Feature: "bedrooms", SubjectValue: subject.Bedrooms, CompValue: comp.Bedrooms, Unit: "rooms",
			RateSource: src, RatePerUnit: rates.BedPerRoom,
			Adjustment: round2(delta * rates.BedPerRoom), Reasoning: "Bedroom count differential",
		}
	case "bathrooms":
		delta := subject.Bathrooms - comp.Bathrooms
		return BpoGridLine{
			Feature: "bathrooms", SubjectValue: subject.Bathrooms, CompValue: comp.Bathrooms, Unit: "baths",
			RateSource: src, RatePerUnit: rates.BathPerFull,
			Adjustment: round2(delta * rates.BathPerFull), Reasoning: "Bathroom count differential",
		}
	case "garage":
		delta := subject.GarageSpaces - comp.GarageSpaces
		return BpoGridLine{
			Feature: "garage", SubjectValue: subject.GarageSpaces, CompValue: comp.GarageSpaces, Unit: "spaces",
			RateSource: src, RatePerUnit: rates.GaragePerSpace,
			Adjustment: round2(delta * rates.GaragePerSpace), Reasoning: "Garage space differential",
		}
	case "pool":
		delta := bool01(subject.PoolPrivate) - bool01(comp.PoolPrivate)
		return BpoGridLine{
			Feature: "pool", SubjectValue: subject.PoolPrivate, CompValue: comp.PoolPrivate,
			RateSource: src, RatePerUnit: rates.PoolValue,
			Adjustment: round2(delta * rates.PoolValue), Reasoning: "Pool presence differential",
		}
	case "waterfront":
		delta := bool01(subject.Waterfront) - bool01(comp.Waterfront)
		return BpoGridLine{
			Feature: "waterfront", SubjectValue: subject.Waterfront, CompValue: comp.Waterfront,
			RateSource: src, RatePerUnit: rates.WaterfrontValue,
			Adjustment: round2(delta * rates.WaterfrontValue), Reasoning: "Waterfront differential",
		}
	case "porch_patio", "hvac":
		return BpoGridLine{
			Feature: feat, RateSource: "indeterminate", Adjustment: 0,
			Reasoning: "Insufficient paired-sales signal; no adjustment",
		}
	default:
		return BpoGridLine{Feature: feat, RateSource: src, Adjustment: 0}
	}
}

func applyURARGrid(subject SubjectProfile, sold []CompRecord, rates BpoMarketRates) ([]CompRecord, []BpoCompGrid) {
	out := make([]CompRecord, len(sold))
	grids := make([]BpoCompGrid, len(sold))
	for i, c := range sold {
		base := c.ClosePrice
		if base <= 0 {
			base = c.ListPrice
		}
		lines := buildURARGrid(subject, c, rates)
		net, gross := sumGridAdjustments(lines)
		cap := base * 0.35
		if cap > 0 {
			if net > cap {
				net = cap
			} else if net < -cap {
				net = -cap
			}
		}
		adj := base + net
		legacy := urarToLegacyLines(lines)
		c.Adjustments = legacy
		c.AdjustedPrice = round2(adj)
		out[i] = c
		pct := 0.0
		if base > 0 {
			pct = round2(gross / base * 100)
		}
		grids[i] = BpoCompGrid{
			CompIndex: i, ListingKey: c.ListingKey, SalePrice: base, SaleDate: c.CloseDate,
			Lines: lines, NetAdjustment: round2(net), GrossAdjustment: round2(gross),
			GrossAdjustmentPct: pct, AdjustedPrice: c.AdjustedPrice,
		}
	}
	return out, grids
}

func sumGridAdjustments(lines []BpoGridLine) (net, gross float64) {
	for _, l := range lines {
		net += l.Adjustment
		gross += math.Abs(l.Adjustment)
	}
	return net, gross
}

func urarToLegacyLines(lines []BpoGridLine) []AdjustmentLine {
	out := make([]AdjustmentLine, 0, len(lines))
	for _, l := range lines {
		if l.Adjustment == 0 {
			continue
		}
		out = append(out, AdjustmentLine{Feature: l.Feature, Amount: l.Adjustment})
	}
	return out
}

func effectiveYearBuilt(subject SubjectProfile) float64 {
	yb := float64(subject.YearBuilt)
	if yb <= 0 {
		return yb
	}
	switch strings.ToLower(strings.TrimSpace(subject.Condition)) {
	case "excellent":
		return yb + 10
	case "fair":
		return yb - 5
	case "poor":
		return yb - 15
	default:
		return yb
	}
}

func locationAdjustment(subject SubjectProfile, comp CompRecord) float64 {
	subDiv := strings.TrimSpace(strings.ToLower(subject.Subdivision))
	compDiv := strings.TrimSpace(strings.ToLower(compSubdivision(comp)))
	if subDiv == "" || compDiv == "" {
		return 0
	}
	if subDiv == compDiv {
		return 0
	}
	if subject.MLSAreaMajor != "" && strings.EqualFold(subject.MLSAreaMajor, compMLSArea(comp)) {
		return 0
	}
	return -comp.ClosePrice * 0.02
}

func qualityPPSFAdjustment(subject SubjectProfile, comp CompRecord, rates BpoMarketRates) float64 {
	if rates.GLAPerSF <= 0 || subject.LivingArea <= 0 || comp.LivingArea <= 0 {
		return 0
	}
	subPPSF := subject.ListPrice / subject.LivingArea
	if subPPSF <= 0 {
		subPPSF = rates.GLAPerSF
	}
	compPPSF := comp.ClosePrice / comp.LivingArea
	if compPPSF <= 0 {
		return 0
	}
	tierDelta := (subPPSF - compPPSF) * comp.LivingArea * 0.25
	return tierDelta
}

func compSubdivision(c CompRecord) string {
	return stringField(c.Property, "SubdivisionName")
}

func compMLSArea(c CompRecord) string {
	return stringField(c.Property, "MLSAreaMajor")
}

func compPropertyType(c CompRecord) string {
	return stringField(c.Property, "PropertySubType")
}

func stringField(raw json.RawMessage, key string) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
