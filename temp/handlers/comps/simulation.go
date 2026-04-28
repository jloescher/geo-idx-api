package comps

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/mls"
)

// --- Internal types ---

type rankedSimComp struct {
	row             compRow
	grid            *AdjustmentGrid
	bedroomAdj      *AdjustmentLine
	timeAdj         *AdjustmentLine
	fullAdjusted    float64
	fullGrossAdjPct float64
	distanceMiles   float64
	subTypeMatch    bool
	bedroomMismatch int
	monthsSinceSale float64
	reRankScore     float64
	reconcileWeight float64
}

type resolvedRates struct {
	glaPerSqft       float64
	poolValue        float64
	garagePerSpace   float64
	waterfrontValue  float64
	yearBuiltPerYear float64
	lotPerAcre       float64
	bedroomValue     float64
}

// --- Constants ---

const (
	simMaxSoldResults = 20
	simSoldLookback   = 12 // months
	simMaxCompResults = 10
	simPreferMiles    = 2.0
	simPreferMonths   = 6

	// Clamping bounds for market-derived rates.
	clampPoolMin       = 5000.0
	clampPoolMax       = 75000.0
	clampWaterfrontMin = 10000.0
	clampWaterfrontMax = 200000.0
	clampGarageMin     = 2000.0
	clampGarageMax     = 25000.0
	clampYearBuiltMin  = 100.0
	clampYearBuiltMax  = 3000.0
	clampLotMin        = 5000.0
	clampLotMax        = 100000.0
	clampBedroomMin    = 1000.0
	clampBedroomMax    = 25000.0

	// Fallback defaults when market data is insufficient.
	defaultGLAPerSqft       = 150.0
	defaultPoolValue        = 20000.0
	defaultWaterfrontValue  = 50000.0
	defaultGaragePerSpace   = 7500.0
	defaultYearBuiltPerYear = 500.0
	defaultLotPerAcre       = 25000.0
	defaultBedroomValue     = 5000.0
)

// --- Validation ---

func validateSimulationParams(req *RunCompsRequest) error {
	p := req.SimulationParams
	if p == nil {
		return nil // all fields optional
	}
	if p.ContractPrice != nil && *p.ContractPrice <= 0 {
		return fmt.Errorf("simulation_params.contract_price must be positive")
	}
	if p.ConcessionPercent != nil && (*p.ConcessionPercent < 0 || *p.ConcessionPercent > 1) {
		return fmt.Errorf("simulation_params.concession_percent must be between 0.0 and 1.0")
	}
	if p.MaxComps != nil && (*p.MaxComps < 1 || *p.MaxComps > 10) {
		return fmt.Errorf("simulation_params.max_comps must be between 1 and 10")
	}
	return nil
}

// --- Market-Derived Rates (3.9) ---

func extractMarketRates(rows []compRow, centralPPSF float64, method AggregationMethod) MarketDerivedRates {
	rates := MarketDerivedRates{
		GLAPerSqft:       defaultGLAPerSqft,
		PoolValue:        defaultPoolValue,
		GaragePerSpace:   defaultGaragePerSpace,
		WaterfrontValue:  defaultWaterfrontValue,
		YearBuiltPerYear: defaultYearBuiltPerYear,
		LotPerAcre:       defaultLotPerAcre,
		BedroomValue:     defaultBedroomValue,
		SampleSize:       len(rows),
	}

	if centralPPSF > 0 {
		rates.GLAPerSqft = centralPPSF
	}

	// Pool: price diff between pool=true and pool=false.
	if v := booleanFeatureDiff(rows, func(r compRow) *bool { return r.PoolPrivateYn }, method); v != nil {
		rates.PoolValue = clamp(math.Abs(*v), clampPoolMin, clampPoolMax)
	}

	// Waterfront: price diff.
	if v := booleanFeatureDiff(rows, func(r compRow) *bool { return r.WaterfrontYn }, method); v != nil {
		rates.WaterfrontValue = clamp(math.Abs(*v), clampWaterfrontMin, clampWaterfrontMax)
	}

	// Garage: pairwise regression on garage_spaces.
	if v := pairwiseRate(rows,
		func(r compRow) *float64 {
			if r.GarageSpaces == nil {
				return nil
			}
			f := float64(*r.GarageSpaces)
			return &f
		},
		func(r compRow) *float64 { return r.ClosePrice },
		3,
		method,
	); v != nil {
		rates.GaragePerSpace = clamp(math.Abs(*v), clampGarageMin, clampGarageMax)
	}

	// Year built: pairwise regression.
	if v := pairwiseRate(rows,
		func(r compRow) *float64 {
			if r.YearBuilt == nil {
				return nil
			}
			f := float64(*r.YearBuilt)
			return &f
		},
		func(r compRow) *float64 { return r.ClosePrice },
		3,
		method,
	); v != nil {
		rates.YearBuiltPerYear = clamp(math.Abs(*v), clampYearBuiltMin, clampYearBuiltMax)
	}

	// Lot size: pairwise regression on acres.
	if v := pairwiseRate(rows,
		func(r compRow) *float64 { return r.LotSizeAcres },
		func(r compRow) *float64 { return r.ClosePrice },
		3,
		method,
	); v != nil {
		rates.LotPerAcre = clamp(math.Abs(*v), clampLotMin, clampLotMax)
	}

	// Bedrooms: adjacent group diffs.
	if v := bedroomGroupDiff(rows, method); v != nil {
		rates.BedroomValue = clamp(math.Abs(*v), clampBedroomMin, clampBedroomMax)
	}

	// Confidence based on sample size.
	switch {
	case len(rows) >= 10:
		rates.Confidence = "high"
	case len(rows) >= 5:
		rates.Confidence = "moderate"
	default:
		rates.Confidence = "low"
	}

	return rates
}

// booleanFeatureDiff computes aggregate(price where feature=true) - aggregate(price where feature=false).
// Returns nil if either group has fewer than 2 comps.
func booleanFeatureDiff(rows []compRow, getFeature func(compRow) *bool, method AggregationMethod) *float64 {
	var withPrices, withoutPrices []float64
	for _, r := range rows {
		f := getFeature(r)
		if f == nil || r.ClosePrice == nil {
			continue
		}
		if *f {
			withPrices = append(withPrices, *r.ClosePrice)
		} else {
			withoutPrices = append(withoutPrices, *r.ClosePrice)
		}
	}
	if len(withPrices) < 2 || len(withoutPrices) < 2 {
		return nil
	}
	diff := aggregate(withPrices, method) - aggregate(withoutPrices, method)
	return &diff
}

// pairwiseRate computes the central tendency rate of change between all unique pairs.
// For each pair (i,j) where feature_i != feature_j: rate = (price_i - price_j) / (feature_i - feature_j).
// Returns nil if fewer than minPairs unique feature values exist.
func pairwiseRate(rows []compRow, getFeature func(compRow) *float64, getPrice func(compRow) *float64, minDistinct int, method AggregationMethod) *float64 {
	type fp struct {
		feature float64
		price   float64
	}
	var pairs []fp
	seen := make(map[float64]bool)
	for _, r := range rows {
		f := getFeature(r)
		p := getPrice(r)
		if f == nil || p == nil {
			continue
		}
		pairs = append(pairs, fp{*f, *p})
		seen[*f] = true
	}
	if len(seen) < minDistinct {
		return nil
	}

	var rates []float64
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			diff := pairs[i].feature - pairs[j].feature
			if diff == 0 {
				continue
			}
			rate := (pairs[i].price - pairs[j].price) / diff
			rates = append(rates, rate)
		}
	}
	if len(rates) == 0 {
		return nil
	}
	m := aggregate(rates, method)
	return &m
}

// bedroomGroupDiff groups comps by bedroom count, computes central tendency price per group,
// then returns the central tendency of all adjacent-group diffs.
func bedroomGroupDiff(rows []compRow, method AggregationMethod) *float64 {
	groups := make(map[int][]float64)
	for _, r := range rows {
		if r.BedroomsTotal == nil || r.ClosePrice == nil {
			continue
		}
		groups[*r.BedroomsTotal] = append(groups[*r.BedroomsTotal], *r.ClosePrice)
	}
	if len(groups) < 2 {
		return nil
	}

	// Sort bedroom counts.
	var beds []int
	for b := range groups {
		beds = append(beds, b)
	}
	sort.Ints(beds)

	// Compute adjacent diffs.
	var diffs []float64
	for i := 0; i < len(beds)-1; i++ {
		m1 := aggregate(groups[beds[i]], method)
		m2 := aggregate(groups[beds[i+1]], method)
		bedDiff := float64(beds[i+1] - beds[i])
		if bedDiff > 0 {
			diffs = append(diffs, (m2-m1)/bedDiff)
		}
	}
	if len(diffs) == 0 {
		return nil
	}
	m := aggregate(diffs, method)
	return &m
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// resolveRates merges market-derived rates with user FiltersInput overrides.
func resolveRates(derived MarketDerivedRates, filters *FiltersInput) resolvedRates {
	r := resolvedRates{
		glaPerSqft:       derived.GLAPerSqft,
		poolValue:        derived.PoolValue,
		garagePerSpace:   derived.GaragePerSpace,
		waterfrontValue:  derived.WaterfrontValue,
		yearBuiltPerYear: derived.YearBuiltPerYear,
		lotPerAcre:       derived.LotPerAcre,
		bedroomValue:     derived.BedroomValue,
	}
	if filters.AdjPoolValue != nil {
		r.poolValue = *filters.AdjPoolValue
	}
	if filters.AdjGaragePerSpace != nil {
		r.garagePerSpace = *filters.AdjGaragePerSpace
	}
	if filters.AdjWaterfrontValue != nil {
		r.waterfrontValue = *filters.AdjWaterfrontValue
	}
	if filters.AdjYearBuiltPerYear != nil {
		r.yearBuiltPerYear = *filters.AdjYearBuiltPerYear
	}
	if filters.AdjLotPerAcre != nil {
		r.lotPerAcre = *filters.AdjLotPerAcre
	}
	if filters.AdjBedroomValue != nil {
		r.bedroomValue = *filters.AdjBedroomValue
	}
	return r
}

// --- Simulation Adjustment Grid (3.9.3) ---

func computeSimAdjustmentGrid(subject *resolvedSubject, comp compRow, rates resolvedRates) *AdjustmentGrid {
	if comp.ClosePrice == nil {
		return nil
	}

	var lines []AdjustmentLine
	var absSum float64

	// GLA adjustment.
	if subject.LivingAreaSqft != nil && comp.LivingArea != nil && rates.glaPerSqft > 0 {
		diff := float64(*subject.LivingAreaSqft - *comp.LivingArea)
		adj := math.Round(diff * rates.glaPerSqft)
		sizeWord := "larger"
		if diff < 0 {
			sizeWord = "smaller"
		}
		reasoning := fmt.Sprintf("Subject is %s sqft %s (%s vs %s sqft); adjusted %s at $%.2f/sqft (market-derived)",
			itoa(int(math.Abs(diff))), sizeWord,
			itoa(*subject.LivingAreaSqft), itoa(*comp.LivingArea),
			formatDollarsSigned(adj), rates.glaPerSqft)
		lines = append(lines, AdjustmentLine{
			Feature:    "gla",
			SubjectVal: *subject.LivingAreaSqft,
			CompVal:    *comp.LivingArea,
			Adjustment: adj,
			Reasoning:  reasoning,
		})
		absSum += math.Abs(adj)
	}

	// Pool adjustment.
	if subject.Pool != nil && comp.PoolPrivateYn != nil && *subject.Pool != *comp.PoolPrivateYn {
		adj := rates.poolValue
		var reasoning string
		if *subject.Pool {
			reasoning = fmt.Sprintf("Subject has pool; comp does not; adjusted %s (market-derived)", formatDollarsSigned(adj))
		} else {
			adj = -adj
			reasoning = fmt.Sprintf("Comp has pool; subject does not; adjusted %s (market-derived)", formatDollarsSigned(adj))
		}
		lines = append(lines, AdjustmentLine{
			Feature:    "pool",
			SubjectVal: *subject.Pool,
			CompVal:    *comp.PoolPrivateYn,
			Adjustment: adj,
			Reasoning:  reasoning,
		})
		absSum += math.Abs(adj)
	}

	// Garage adjustment.
	if subject.GarageSpaces != nil && comp.GarageSpaces != nil {
		diff := *subject.GarageSpaces - *comp.GarageSpaces
		if diff != 0 {
			adj := float64(diff) * rates.garagePerSpace
			moreOrFewer := "more"
			if diff < 0 {
				moreOrFewer = "fewer"
			}
			reasoning := fmt.Sprintf("Subject has %s %s garage space(s) (%s vs %s); adjusted %s at %s/space (market-derived)",
				itoa(int(math.Abs(float64(diff)))), moreOrFewer,
				itoa(*subject.GarageSpaces), itoa(*comp.GarageSpaces),
				formatDollarsSigned(adj), formatDollars(rates.garagePerSpace))
			lines = append(lines, AdjustmentLine{
				Feature:    "garage",
				SubjectVal: *subject.GarageSpaces,
				CompVal:    *comp.GarageSpaces,
				Adjustment: adj,
				Reasoning:  reasoning,
			})
			absSum += math.Abs(adj)
		}
	}

	// Waterfront adjustment.
	if subject.Waterfront != nil && comp.WaterfrontYn != nil && *subject.Waterfront != *comp.WaterfrontYn {
		adj := rates.waterfrontValue
		var reasoning string
		if *subject.Waterfront {
			reasoning = fmt.Sprintf("Subject is waterfront; comp is not; adjusted %s (market-derived)", formatDollarsSigned(adj))
		} else {
			adj = -adj
			reasoning = fmt.Sprintf("Comp is waterfront; subject is not; adjusted %s (market-derived)", formatDollarsSigned(adj))
		}
		lines = append(lines, AdjustmentLine{
			Feature:    "waterfront",
			SubjectVal: *subject.Waterfront,
			CompVal:    *comp.WaterfrontYn,
			Adjustment: adj,
			Reasoning:  reasoning,
		})
		absSum += math.Abs(adj)
	}

	// Year built adjustment.
	if subject.YearBuilt != nil && comp.YearBuilt != nil {
		diff := *subject.YearBuilt - *comp.YearBuilt
		if diff != 0 {
			adj := float64(diff) * rates.yearBuiltPerYear
			ageWord := "newer"
			if diff < 0 {
				ageWord = "older"
			}
			reasoning := fmt.Sprintf("Subject is %s years %s (%s vs %s); adjusted %s at %s/year (market-derived)",
				itoa(int(math.Abs(float64(diff)))), ageWord,
				itoa(*subject.YearBuilt), itoa(*comp.YearBuilt),
				formatDollarsSigned(adj), formatDollars(rates.yearBuiltPerYear))
			lines = append(lines, AdjustmentLine{
				Feature:    "year_built",
				SubjectVal: *subject.YearBuilt,
				CompVal:    *comp.YearBuilt,
				Adjustment: adj,
				Reasoning:  reasoning,
			})
			absSum += math.Abs(adj)
		}
	}

	// Lot size adjustment.
	if subject.LotSizeAcres != nil && comp.LotSizeAcres != nil {
		diff := *subject.LotSizeAcres - *comp.LotSizeAcres
		if diff != 0 {
			adj := math.Round(diff * rates.lotPerAcre)
			sizeWord := "larger"
			if diff < 0 {
				sizeWord = "smaller"
			}
			reasoning := fmt.Sprintf("Subject lot is %.2f acres %s (%.2f vs %.2f acres); adjusted %s at %s/acre (market-derived)",
				math.Abs(diff), sizeWord,
				*subject.LotSizeAcres, *comp.LotSizeAcres,
				formatDollarsSigned(adj), formatDollars(rates.lotPerAcre))
			lines = append(lines, AdjustmentLine{
				Feature:    "lot_size",
				SubjectVal: *subject.LotSizeAcres,
				CompVal:    *comp.LotSizeAcres,
				Adjustment: adj,
				Reasoning:  reasoning,
			})
			absSum += math.Abs(adj)
		}
	}

	// Compute totals.
	net := 0.0
	for _, l := range lines {
		net += l.Adjustment
	}

	adjustedPrice := *comp.ClosePrice + net
	grossPct := 0.0
	if *comp.ClosePrice > 0 {
		grossPct = absSum / *comp.ClosePrice * 100
	}

	return &AdjustmentGrid{
		Lines:          lines,
		NetAdjustment:  math.Round(net),
		AdjustedPrice:  math.Round(adjustedPrice),
		GrossAdjPct:    math.Round(grossPct*10) / 10,
		HighAdjWarning: grossPct > 25,
	}
}

// --- Quarterly Trend (3.3) ---

func computeQuarterlyTrend(rows []compRow, method AggregationMethod) float64 {
	now := time.Now()
	cutoff := now.AddDate(0, -3, 0)

	var recentPPSFs, priorPPSFs []float64
	for _, r := range rows {
		if r.ClosePrice == nil || r.LivingArea == nil || *r.LivingArea == 0 || r.CloseDate == nil {
			continue
		}
		ppsf := *r.ClosePrice / float64(*r.LivingArea)
		if r.CloseDate.After(cutoff) {
			recentPPSFs = append(recentPPSFs, ppsf)
		} else {
			priorPPSFs = append(priorPPSFs, ppsf)
		}
	}
	if len(recentPPSFs) == 0 || len(priorPPSFs) == 0 {
		return 0
	}
	recentCentral := aggregate(recentPPSFs, method)
	priorCentral := aggregate(priorPPSFs, method)
	if priorCentral == 0 {
		return 0
	}
	return recentCentral/priorCentral - 1
}

// --- Time Adjustment (3.4) ---

func applyTimeAdjustment(soldPrice float64, closeDate time.Time, quarterlyTrend float64) *AdjustmentLine {
	if math.Abs(quarterlyTrend) <= 0.01 {
		return nil
	}
	monthsSinceSale := time.Since(closeDate).Hours() / (24 * 30.44)
	adj := math.Round(soldPrice * quarterlyTrend * (monthsSinceSale / 3.0))
	if adj == 0 {
		return nil
	}
	direction := "appreciating"
	if quarterlyTrend < 0 {
		direction = "depreciating"
	}
	reasoning := fmt.Sprintf("Market %s at %.1f%%/quarter; comp sold %.1f months ago; time adjustment %s",
		direction, quarterlyTrend*100, monthsSinceSale, formatDollarsSigned(adj))
	return &AdjustmentLine{
		Feature:    "time",
		SubjectVal: "current",
		CompVal:    closeDate.Format("2006-01-02"),
		Adjustment: adj,
		Reasoning:  reasoning,
	}
}

// --- Bedroom Adjustment (3.5) ---

func applyBedroomAdjustment(subjectBeds, compBeds *int, valuePerBedroom float64) *AdjustmentLine {
	if subjectBeds == nil || compBeds == nil {
		return nil
	}
	diff := *subjectBeds - *compBeds
	if diff == 0 {
		return nil
	}
	adj := float64(diff) * valuePerBedroom
	moreOrFewer := "more"
	if diff < 0 {
		moreOrFewer = "fewer"
	}
	reasoning := fmt.Sprintf("Subject has %d %s bedroom(s) (%d vs %d); adjusted %s at %s/bedroom (market-derived)",
		int(math.Abs(float64(diff))), moreOrFewer,
		*subjectBeds, *compBeds,
		formatDollarsSigned(adj), formatDollars(valuePerBedroom))
	return &AdjustmentLine{
		Feature:    "bedrooms",
		SubjectVal: *subjectBeds,
		CompVal:    *compBeds,
		Adjustment: adj,
		Reasoning:  reasoning,
	}
}

// --- Re-Ranking (3.6) ---

func reRankForSimulation(rows []compRow, subject *resolvedSubject, rates resolvedRates, params *SimulationParamsInput, quarterlyTrend float64) []rankedSimComp {
	now := time.Now()
	sixMonthsAgo := now.AddDate(0, -simPreferMonths, 0)

	var all []rankedSimComp
	for _, r := range rows {
		if r.ClosePrice == nil {
			continue
		}

		grid := computeSimAdjustmentGrid(subject, r, rates)
		bedAdj := applyBedroomAdjustment(subject.Bedrooms, r.BedroomsTotal, rates.bedroomValue)
		timeAdj := applyTimeAdjustment(*r.ClosePrice, safeTime(r.CloseDate), quarterlyTrend)

		gridNet := 0.0
		gridAbsSum := 0.0
		if grid != nil {
			gridNet = grid.NetAdjustment
			for _, l := range grid.Lines {
				gridAbsSum += math.Abs(l.Adjustment)
			}
		}

		bedAdjVal := 0.0
		if bedAdj != nil {
			bedAdjVal = bedAdj.Adjustment
		}
		timeAdjVal := 0.0
		if timeAdj != nil {
			timeAdjVal = timeAdj.Adjustment
		}

		fullAdjusted := *r.ClosePrice + gridNet + bedAdjVal + timeAdjVal
		totalAbsAdj := gridAbsSum + math.Abs(bedAdjVal) + math.Abs(timeAdjVal)
		fullGrossAdjPct := 0.0
		if *r.ClosePrice > 0 {
			fullGrossAdjPct = totalAbsAdj / *r.ClosePrice * 100
		}

		distMiles := r.DistanceMeters / 1609.344

		subTypeMatch := false
		if subject.PropertySubType != nil && r.PropertySubType != nil {
			subTypeMatch = *subject.PropertySubType == *r.PropertySubType
		}

		bedroomMismatch := 0
		if subject.Bedrooms != nil && r.BedroomsTotal != nil {
			bedroomMismatch = int(math.Abs(float64(*subject.Bedrooms - *r.BedroomsTotal)))
		}

		monthsSinceSale := 0.0
		if r.CloseDate != nil {
			monthsSinceSale = time.Since(*r.CloseDate).Hours() / (24 * 30.44)
		}

		// Re-rank score (lower = better).
		distScore := math.Min(distMiles/simPreferMiles, 1.0)
		adjScore := math.Min(fullGrossAdjPct/50.0, 1.0)
		subTypeScore := 1.0
		if subTypeMatch {
			subTypeScore = 0.0
		}
		glaScore := 0.0
		if subject.LivingAreaSqft != nil && r.LivingArea != nil && *subject.LivingAreaSqft > 0 {
			glaScore = math.Min(math.Abs(float64(*subject.LivingAreaSqft-*r.LivingArea))/float64(*subject.LivingAreaSqft), 1.0)
		}

		reRankScore := 0.30*distScore + 0.35*adjScore + 0.20*subTypeScore + 0.15*glaScore

		all = append(all, rankedSimComp{
			row:             r,
			grid:            grid,
			bedroomAdj:      bedAdj,
			timeAdj:         timeAdj,
			fullAdjusted:    math.Round(fullAdjusted),
			fullGrossAdjPct: math.Round(fullGrossAdjPct*10) / 10,
			distanceMiles:   round2(distMiles),
			subTypeMatch:    subTypeMatch,
			bedroomMismatch: bedroomMismatch,
			monthsSinceSale: monthsSinceSale,
			reRankScore:     reRankScore,
		})
	}

	// Prefer comps within 2 miles and 6 months.
	var preferred []rankedSimComp
	for _, c := range all {
		if c.distanceMiles <= simPreferMiles && c.row.CloseDate != nil && c.row.CloseDate.After(sixMonthsAgo) {
			preferred = append(preferred, c)
		}
	}

	candidates := preferred
	if len(candidates) < 3 {
		candidates = all
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].reRankScore < candidates[j].reRankScore
	})

	maxComps := params.maxComps()
	if len(candidates) > maxComps {
		candidates = candidates[:maxComps]
	}

	return candidates
}

func safeTime(t *time.Time) time.Time {
	if t != nil {
		return *t
	}
	return time.Time{}
}

// --- Reconciliation (3.7) ---

func reconcileSimComps(comps []rankedSimComp) (low, high, weightedMid float64, confidence string) {
	if len(comps) == 0 {
		return 0, 0, 0, "low"
	}

	// Weight by inverse gross adjustment.
	totalWeight := 0.0
	for i := range comps {
		w := 1.0 / math.Max(comps[i].fullGrossAdjPct, 1.0)
		comps[i].reconcileWeight = w
		totalWeight += w
	}

	// Normalize weights.
	if totalWeight > 0 {
		for i := range comps {
			comps[i].reconcileWeight /= totalWeight
		}
	}

	// Compute weighted midpoint, low, high.
	low = math.MaxFloat64
	high = -math.MaxFloat64
	weightedMid = 0
	var totalGrossAdj float64

	for _, c := range comps {
		weightedMid += c.fullAdjusted * c.reconcileWeight
		if c.fullAdjusted < low {
			low = c.fullAdjusted
		}
		if c.fullAdjusted > high {
			high = c.fullAdjusted
		}
		totalGrossAdj += c.fullGrossAdjPct
	}

	weightedMid = math.Round(weightedMid)
	avgGrossAdj := totalGrossAdj / float64(len(comps))

	switch {
	case len(comps) >= 3 && avgGrossAdj < 15:
		confidence = "high"
	case len(comps) >= 2 && avgGrossAdj < 25:
		confidence = "moderate"
	default:
		confidence = "low"
	}

	return low, high, weightedMid, confidence
}

// --- Commentary & Disclaimers (3.8) ---

func buildSimulationCommentary(comps []rankedSimComp, confidence string, timeAdjApplied bool, quarterlyTrend float64) string {
	if len(comps) == 0 {
		return "Insufficient comparable data to generate a valuation simulation."
	}

	var minDist, maxDist float64
	minDist = math.MaxFloat64
	for _, c := range comps {
		if c.distanceMiles < minDist {
			minDist = c.distanceMiles
		}
		if c.distanceMiles > maxDist {
			maxDist = c.distanceMiles
		}
	}

	commentary := fmt.Sprintf("Analysis based on %d comparable sales within %.1f-%.1f miles. ", len(comps), minDist, maxDist)
	commentary += fmt.Sprintf("Confidence level: %s. ", confidence)

	if timeAdjApplied {
		direction := "appreciation"
		if quarterlyTrend < 0 {
			direction = "depreciation"
		}
		commentary += fmt.Sprintf("Time adjustments applied reflecting %.1f%% quarterly %s.", math.Abs(quarterlyTrend)*100, direction)
	} else {
		commentary += "No significant market trend detected; time adjustments not applied."
	}

	return commentary
}

func buildBPODisclaimers() []string {
	return []string{
		"This Broker Price Opinion is not an appraisal and was not prepared by a state-licensed or certified appraiser. It is an independent market analysis prepared for informational and strategic pricing purposes only.",
		"The indicated market value is based on comparable sales data and analytical modeling. Actual appraised values may differ based on property condition, interior inspection, and appraiser judgment.",
	}
}

// --- Handler Pipeline (3.2) ---

func (h *Handler) handleAppraiserSimulation(w http.ResponseWriter, r *http.Request, req RunCompsRequest, subject *resolvedSubject, start time.Time) {
	ctx := r.Context()
	params := req.SimulationParams

	// Default 2-mile radius if not set.
	simScope := req.Scope
	if simScope.Type == "radius" && simScope.RadiusMiles == nil {
		defaultRadius := 2.0
		simScope.RadiusMiles = &defaultRadius
	}

	scope, err := buildScopeClause(subject, simScope)
	if err != nil {
		writeTimedErrorResponse(w, http.StatusBadRequest, err.Error(), start)
		return
	}

	normDistMeters := 16093.44 // 10 miles default
	if simScope.RadiusMiles != nil {
		normDistMeters = *simScope.RadiusMiles * 1609.344
	}
	simArgs := buildSimilarityArgs(subject, normDistMeters)

	filterClause, filterArgs := buildFilterClauses(subject, &req.Filters)
	soldSince := time.Now().AddDate(0, -simSoldLookback, 0)

	// Run sold + competition queries in parallel.
	g, gCtx := errgroup.WithContext(ctx)

	var soldRows []compRow
	var soldTotal int
	g.Go(func() error {
		var err error
		soldRows, soldTotal, err = h.queryComps(gCtx, "CompsSoldBase", simArgs, scope, filterClause, filterArgs, subject.ListingID, soldSince, simMaxSoldResults)
		return err
	})

	var compTotal int
	g.Go(func() error {
		var err error
		_, compTotal, err = h.queryComps(gCtx, "CompsCompetitionBase", simArgs, scope, filterClause, filterArgs, subject.ListingID, time.Time{}, simMaxCompResults)
		return err
	})

	if err := g.Wait(); err != nil {
		log.Printf("comps: appraiser_simulation query failed: %v", err)
		writeTimedErrorResponse(w, http.StatusInternalServerError, "Internal server error", start)
		return
	}

	// Flood zone partitioning.
	if req.Filters.matchFlood() && len(subject.FloodZoneCodes) > 0 {
		soldRows, _ = partitionByFloodZone(subject.FloodZoneCodes, soldRows)
	}

	var warnings []string

	// Step 6: Compute central PPSF.
	aggMethod, _ := parseAggregationMethod(req.AggregationMethod)
	medPPSF := ppsfFromRows(soldRows, aggMethod)

	// Step 7: Extract market-derived rates.
	derivedRates := extractMarketRates(soldRows, medPPSF, aggMethod)

	// Step 8: Resolve final rates (merge with user overrides).
	rates := resolveRates(derivedRates, &req.Filters)

	// Step 9: Market conditions.
	soldMonths := simSoldLookback
	if req.Filters.SoldMonthsBack != nil {
		soldMonths = *req.Filters.SoldMonthsBack
	}
	marketConditions := computeMarketConditions(soldRows, soldTotal, compTotal, soldMonths, aggMethod)

	// Step 10: Quarterly trend.
	quarterlyTrend := computeQuarterlyTrend(soldRows, aggMethod)

	// Step 11: Re-rank and select top comps.
	ranked := reRankForSimulation(soldRows, subject, rates, params, quarterlyTrend)

	if len(ranked) == 0 {
		warnings = append(warnings, "No comparable sales available for simulation")
	}

	// Step 12: Reconcile.
	low, high, weightedMid, confidence := reconcileSimComps(ranked)

	// Check if time adj was applied.
	timeAdjApplied := false
	for _, c := range ranked {
		if c.timeAdj != nil {
			timeAdjApplied = true
			break
		}
	}

	// Step 13: Build BPO + Simulation summaries.
	methodology := "Sales comparison approach using market-derived adjustment rates with inverse gross adjustment weighting for reconciliation."
	bpo := BPOSummary{
		ReportType:         "Broker Price Opinion",
		IndicatedValue:     weightedMid,
		ValueRangeLow:      math.Round(low),
		ValueRangeHigh:     math.Round(high),
		MethodologySummary: methodology,
		Confidence:         confidence,
	}

	// Gross adj range string.
	grossAdjRange := ""
	if len(ranked) > 0 {
		minAdj := ranked[0].fullGrossAdjPct
		maxAdj := ranked[0].fullGrossAdjPct
		for _, c := range ranked[1:] {
			if c.fullGrossAdjPct < minAdj {
				minAdj = c.fullGrossAdjPct
			}
			if c.fullGrossAdjPct > maxAdj {
				maxAdj = c.fullGrossAdjPct
			}
		}
		grossAdjRange = fmt.Sprintf("%.1f%% - %.1f%%", minAdj, maxAdj)
	}

	primaryComps := mapSimulationCompDetails(ranked, subject)
	commentary := buildSimulationCommentary(ranked, confidence, timeAdjApplied, quarterlyTrend)

	simSummary := SimulationSummary{
		SimulatedValue:    weightedMid,
		ValueRangeLow:     math.Round(low),
		ValueRangeHigh:    math.Round(high),
		PrimaryCompsUsed:  primaryComps,
		GrossAdjRange:     grossAdjRange,
		CommentarySummary: commentary,
		ConfidenceLevel:   confidence,
		TimeAdjApplied:    timeAdjApplied,
		QuarterlyTrendPct: math.Round(quarterlyTrend*10000) / 100, // e.g. 0.03 → 3.0
		MedianPPSF:        round2(medPPSF),
	}

	// Step 14: Valuation risk score (if contract price provided).
	var riskScore *ValuationRiskScore
	if params != nil && params.ContractPrice != nil {
		concessionPct := params.concessionPercent() * 100 // convert to percentage
		riskScore = computeValuationRisk(*params.ContractPrice, weightedMid, ranked, concessionPct, quarterlyTrend, subject)
	}

	// Step 15: Map supporting sold comps.
	supportingComps := mapSoldComps(soldRows, nil, "appraiser_simulation", subject, medPPSF, &req.Filters, subject.FloodZoneCodes)

	// Build subject response.
	subjectResp := SubjectResponse{
		ListingID:       mls.StripPrefix(subject.ListingID),
		Address:         subject.Address,
		Lat:             subject.Lat,
		Lng:             subject.Lng,
		Bedrooms:        subject.Bedrooms,
		Bathrooms:       subject.Bathrooms,
		LivingAreaSqft:  subject.LivingAreaSqft,
		LotSizeSqft:     subject.LotSizeSqft,
		YearBuilt:       subject.YearBuilt,
		ListPrice:       subject.ListPrice,
		PropertyType:    subject.PropertyType,
		PropertySubType: subject.PropertySubType,
		Waterfront:      subject.Waterfront,
		GarageSpaces:    subject.GarageSpaces,
		SeniorCommunity: subject.SeniorCommunity,
		FloodZoneCodes:  subject.FloodZoneCodes,
	}

	if warnings == nil {
		warnings = []string{}
	}

	processingMs := time.Since(start).Milliseconds()
	radiusMiles := 0.0
	if simScope.RadiusMiles != nil {
		radiusMiles = *simScope.RadiusMiles
	}

	h.trackCompsEvent(ctx, req.Mode, req.Scope.Type, len(soldRows), compTotal, 0, processingMs)

	writeJSON(w, http.StatusOK, RunCompsResponse{
		Success: true,
		Subject: &subjectResp,
		SimulationResult: &SimulationResult{
			BrokerPriceOpinion:  bpo,
			ValuationSimulation: simSummary,
			ValuationRisk:       riskScore,
			MarketDerivedRates:  derivedRates,
			SupportingComps:     supportingComps,
			MarketConditions:    marketConditions,
			Warnings:            warnings,
			Disclaimers:         buildBPODisclaimers(),
			Metadata: SimulationMetadata{
				TotalSoldCandidates: soldTotal,
				CompsSelected:       len(ranked),
				ScopeApplied:        req.Scope.Type,
				RadiusMiles:         radiusMiles,
				ProcessingMs:        processingMs,
			},
		},
		Metadata: Metadata{ProcessingMs: processingMs},
	})
}

// mapSimulationCompDetails maps ranked sim comps to the response detail type.
func mapSimulationCompDetails(comps []rankedSimComp, subject *resolvedSubject) []SimulationCompDetail {
	details := make([]SimulationCompDetail, 0, len(comps))
	for _, c := range comps {
		d := SimulationCompDetail{
			Address:           buildAddress(c.row.StreetNumber, c.row.StreetDirPrefix, c.row.StreetName, c.row.StreetSuffix, c.row.StreetDirSuffix, c.row.UnitNumber, c.row.City, c.row.State, c.row.PostalCode),
			DistanceMiles:     c.distanceMiles,
			Adjustments:       c.grid,
			BedroomAdj:        c.bedroomAdj,
			TimeAdj:           c.timeAdj,
			FullAdjustedPrice: c.fullAdjusted,
			GrossAdjPct:       c.fullGrossAdjPct,
			ReconcileWeight:   math.Round(c.reconcileWeight*1000) / 1000,
			SubTypeMatch:      c.subTypeMatch,
		}
		if c.row.ListingID != nil {
			d.ListingID = mls.StripPrefix(*c.row.ListingID)
		}
		if c.row.ClosePrice != nil {
			d.SoldPrice = *c.row.ClosePrice
		}
		if c.row.CloseDate != nil {
			d.SoldDate = c.row.CloseDate.Format("2006-01-02")
		}
		details = append(details, d)
	}
	return details
}
