package comps

import (
	"math"
	"sort"
	"time"
)

// kernelSimilarity computes a weighted similarity score across multiple dimensions
// using exponential decay. Returns a value in [0,1].
func kernelSimilarity(subject *resolvedSubject, comp compRow, subTypeFactor float64) float64 {
	type dim struct {
		score  float64
		weight float64
	}

	var dims []dim

	// Bedrooms: exp(-2.0 * |delta|), weight 0.25.
	if subject.Bedrooms != nil && comp.BedroomsTotal != nil {
		delta := math.Abs(float64(*comp.BedroomsTotal - *subject.Bedrooms))
		dims = append(dims, dim{math.Exp(-2.0 * delta), 0.25})
	} else {
		dims = append(dims, dim{0.5, 0.25})
	}

	// Bathrooms: exp(-1.5 * |delta|), weight 0.15.
	if subject.Bathrooms != nil && comp.BathroomsTotal != nil {
		delta := math.Abs(float64(*comp.BathroomsTotal - *subject.Bathrooms))
		dims = append(dims, dim{math.Exp(-1.5 * delta), 0.15})
	} else {
		dims = append(dims, dim{0.5, 0.15})
	}

	// Living area sqft: exp(-3.0 * |delta|/subject_sqft), weight 0.30.
	if subject.LivingAreaSqft != nil && *subject.LivingAreaSqft > 0 && comp.LivingArea != nil {
		delta := math.Abs(float64(*comp.LivingArea-*subject.LivingAreaSqft)) / float64(*subject.LivingAreaSqft)
		dims = append(dims, dim{math.Exp(-3.0 * delta), 0.30})
	} else {
		dims = append(dims, dim{0.5, 0.30})
	}

	// Year built: exp(-0.03 * |delta|), weight 0.15.
	if subject.YearBuilt != nil && comp.YearBuilt != nil {
		delta := math.Abs(float64(*comp.YearBuilt - *subject.YearBuilt))
		dims = append(dims, dim{math.Exp(-0.03 * delta), 0.15})
	} else {
		dims = append(dims, dim{0.5, 0.15})
	}

	// Sub-type factor (from existing scoreSubTypeMatch), weight 0.15.
	dims = append(dims, dim{subTypeFactor, 0.15})

	var sumWS, sumW float64
	for _, d := range dims {
		sumWS += d.score * d.weight
		sumW += d.weight
	}
	if sumW == 0 {
		return 0.5
	}
	return sumWS / sumW
}

// distanceDecayFactor computes a Gaussian distance decay.
// D = exp(-(miles / decayMiles)^2).
func distanceDecayFactor(distanceMiles, decayMiles float64) float64 {
	if decayMiles <= 0 {
		return 1.0
	}
	ratio := distanceMiles / decayMiles
	return math.Exp(-ratio * ratio)
}

// recencyDecayFactor computes an exponential half-life decay based on close date.
// R = 0.5^(days_old / halfLife). Returns 0.5 for nil closeDate (active listings).
func recencyDecayFactor(closeDate *time.Time, halfLifeDays int) float64 {
	if closeDate == nil || halfLifeDays <= 0 {
		return 0.5
	}
	daysOld := time.Since(*closeDate).Hours() / 24
	if daysOld < 0 {
		daysOld = 0
	}
	return math.Pow(0.5, daysOld/float64(halfLifeDays))
}

// winsorizeRentPerSqft clamps rent_per_sqft outliers at the given percentile bounds.
// Returns the number of values clamped.
func winsorizeRentPerSqft(comps []rankedRentalComp, winsorPct float64) int {
	// Collect indices of comps with valid rent and sqft.
	type entry struct {
		idx         int
		rentPerSqft float64
	}
	var entries []entry
	for i := range comps {
		if comps[i].rent == nil || comps[i].row.LivingArea == nil || *comps[i].row.LivingArea <= 0 {
			continue
		}
		entries = append(entries, entry{i, *comps[i].rent / float64(*comps[i].row.LivingArea)})
	}

	if len(entries) < 5 {
		return 0
	}

	// Sort by rent_per_sqft.
	sort.Slice(entries, func(i, j int) bool { return entries[i].rentPerSqft < entries[j].rentPerSqft })

	vals := make([]float64, len(entries))
	for i, e := range entries {
		vals[i] = e.rentPerSqft
	}
	lowerBound := percentile(vals, winsorPct*100)
	upperBound := percentile(vals, (1-winsorPct)*100)

	clamped := 0
	for _, e := range entries {
		idx := e.idx
		sqft := float64(*comps[idx].row.LivingArea)
		rpsf := *comps[idx].rent / sqft
		if rpsf < lowerBound {
			newRent := lowerBound * sqft
			comps[idx].rent = &newRent
			comps[idx].rentPerSqft = lowerBound
			clamped++
		} else if rpsf > upperBound {
			newRent := upperBound * sqft
			comps[idx].rent = &newRent
			comps[idx].rentPerSqft = upperBound
			clamped++
		}
	}
	return clamped
}

// weightedMedian computes the weighted median from paired value/weight slices.
// Both slices must be the same length.
func weightedMedian(values, weights []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}

	// Build index pairs and sort by value.
	type vw struct {
		val    float64
		weight float64
	}
	pairs := make([]vw, len(values))
	var totalWeight float64
	for i := range values {
		pairs[i] = vw{values[i], weights[i]}
		totalWeight += weights[i]
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].val < pairs[j].val })

	// Walk forward accumulating weight, return value where cumulative crosses 50%.
	half := totalWeight / 2
	var cum float64
	for _, p := range pairs {
		cum += p.weight
		if cum >= half {
			return p.val
		}
	}
	return pairs[len(pairs)-1].val
}

// weightedPercentile computes the weighted percentile at pct (0-100).
func weightedPercentile(values, weights []float64, pct float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}

	type vw struct {
		val    float64
		weight float64
	}
	pairs := make([]vw, len(values))
	var totalWeight float64
	for i := range values {
		pairs[i] = vw{values[i], weights[i]}
		totalWeight += weights[i]
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].val < pairs[j].val })

	target := totalWeight * pct / 100
	var cum float64
	for i, p := range pairs {
		cum += p.weight
		if cum >= target {
			if i == 0 {
				return p.val
			}
			// Linear interpolation between this and previous.
			prevCum := cum - p.weight
			frac := (target - prevCum) / p.weight
			return pairs[i-1].val + frac*(p.val-pairs[i-1].val)
		}
	}
	return pairs[len(pairs)-1].val
}

// estimateRentV2 computes a blended weighted-mean/weighted-median rent estimate
// using kernel similarity, distance/recency decay, and winsorization.
func estimateRentV2(
	closedComps, activeComps []rankedRentalComp,
	subject *resolvedSubject,
	cfg *RentWeightingConfig,
) RentEstimate {
	subjectSubType := ""
	if subject.PropertySubType != nil {
		subjectSubType = *subject.PropertySubType
	}

	allComps := make([]rankedRentalComp, 0, len(closedComps)+len(activeComps))
	for i := range closedComps {
		allComps = append(allComps, closedComps[i])
	}
	for i := range activeComps {
		allComps = append(allComps, activeComps[i])
	}

	// Step 1: Compute per-comp weights.
	floor := cfg.minWeightFloor()
	decayMiles := cfg.distanceDecayMiles()
	halfLife := cfg.recencyHalfLifeDays()
	closedMult := cfg.closedWeightMultiplier()
	activeMult := cfg.activeWeightMultiplier()

	for i := range allComps {
		c := &allComps[i]
		compSubType := ""
		if c.row.PropertySubType != nil {
			compSubType = *c.row.PropertySubType
		}
		_, stFactor := scoreSubTypeMatch(subjectSubType, compSubType)

		c.kernelSim = kernelSimilarity(subject, c.row, stFactor)
		c.distanceDecay = distanceDecayFactor(c.distanceMiles, decayMiles)
		c.recencyDecay = recencyDecayFactor(c.row.CloseDate, halfLife)

		if c.isClosedLeased {
			c.statusMult = closedMult
		} else {
			c.statusMult = activeMult
		}

		raw := c.kernelSim * c.distanceDecay * c.recencyDecay * c.statusMult
		if raw < floor {
			raw = floor
		}
		c.rawWeight = raw

		// Compute rent per sqft for winsorization reference.
		if c.rent != nil && c.row.LivingArea != nil && *c.row.LivingArea > 0 {
			c.rentPerSqft = *c.rent / float64(*c.row.LivingArea)
		}
	}

	// Step 2: Winsorize rent_per_sqft tails.
	winsorized := winsorizeRentPerSqft(allComps, cfg.winsorizePercent())

	// Step 3: Collect (rent, weight) pairs from comps with valid rent.
	var rents, weights []float64
	var closedWeightSum, totalWeightSum float64

	for i := range allComps {
		c := &allComps[i]
		if c.rent == nil {
			continue
		}
		rents = append(rents, *c.rent)
		weights = append(weights, c.rawWeight)
		totalWeightSum += c.rawWeight
		if c.isClosedLeased {
			closedWeightSum += c.rawWeight
		}
	}

	if len(rents) == 0 {
		return RentEstimate{MethodVersion: "rent_weighting_v2_kernel_winsor_blend"}
	}

	// Step 4: Normalize weights.
	for i := range allComps {
		if totalWeightSum > 0 {
			allComps[i].normWeight = allComps[i].rawWeight / totalWeightSum
		}
	}

	// Step 5: Weighted mean.
	var sumRW float64
	for i, r := range rents {
		sumRW += r * weights[i]
	}
	wMean := sumRW / totalWeightSum

	// Step 6: Weighted median.
	wMedian := weightedMedian(rents, weights)

	// Step 7: Blend.
	blend := cfg.medianBlend()
	recommended := blend*wMedian + (1-blend)*wMean

	// Step 8: Low/high from weighted percentiles.
	low := weightedPercentile(rents, weights, cfg.rangeLowPct())
	high := weightedPercentile(rents, weights, cfg.rangeHighPct())

	// Step 9: Closed effective share.
	closedShare := 0.0
	if totalWeightSum > 0 {
		closedShare = closedWeightSum / totalWeightSum
	}

	// Step 10: Active median.
	var activeRents []float64
	for i := range allComps {
		if !allComps[i].isClosedLeased && allComps[i].rent != nil {
			activeRents = append(activeRents, *allComps[i].rent)
		}
	}
	var activeMedian *float64
	if len(activeRents) > 0 {
		sort.Float64s(activeRents)
		m := round2(percentile(activeRents, 50))
		activeMedian = &m
	}

	// Count closed comps with valid rent.
	closedCount := 0
	for i := range allComps {
		if allComps[i].isClosedLeased && allComps[i].rent != nil {
			closedCount++
		}
	}

	// Copy normalized weights back to the input slices.
	closedIdx, activeIdx := 0, 0
	for i := range allComps {
		if allComps[i].isClosedLeased {
			if closedIdx < len(closedComps) {
				closedComps[closedIdx].kernelSim = allComps[i].kernelSim
				closedComps[closedIdx].distanceDecay = allComps[i].distanceDecay
				closedComps[closedIdx].recencyDecay = allComps[i].recencyDecay
				closedComps[closedIdx].statusMult = allComps[i].statusMult
				closedComps[closedIdx].rawWeight = allComps[i].rawWeight
				closedComps[closedIdx].normWeight = allComps[i].normWeight
				closedComps[closedIdx].rentPerSqft = allComps[i].rentPerSqft
				closedComps[closedIdx].rent = allComps[i].rent
				closedIdx++
			}
		} else {
			if activeIdx < len(activeComps) {
				activeComps[activeIdx].kernelSim = allComps[i].kernelSim
				activeComps[activeIdx].distanceDecay = allComps[i].distanceDecay
				activeComps[activeIdx].recencyDecay = allComps[i].recencyDecay
				activeComps[activeIdx].statusMult = allComps[i].statusMult
				activeComps[activeIdx].rawWeight = allComps[i].rawWeight
				activeComps[activeIdx].normWeight = allComps[i].normWeight
				activeComps[activeIdx].rentPerSqft = allComps[i].rentPerSqft
				activeComps[activeIdx].rent = allComps[i].rent
				activeIdx++
			}
		}
	}

	return RentEstimate{
		Recommended:          round2(recommended),
		Low:                  round2(low),
		High:                 round2(high),
		ActiveMedian:         activeMedian,
		CompCount:            closedCount,
		ActiveCompCount:      len(activeRents),
		WeightedMean:         round2(wMean),
		WeightedMedian:       round2(wMedian),
		MedianBlendRatio:     blend,
		ClosedEffectiveShare: round4(closedShare),
		WinsorizedCount:      winsorized,
		ActiveCompMedian:     activeMedian,
		MethodVersion:        "rent_weighting_v2_kernel_winsor_blend",
	}
}
