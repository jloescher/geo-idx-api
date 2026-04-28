package comps

import (
	"sort"
	"time"
)

// productFamilies maps family names to their property sub-type members.
var productFamilies = map[string][]string{
	"detached":     {"Single Family Residence"},
	"attached":     {"Townhouse", "Villa", "Rowhouse"},
	"condo":        {"Condominium", "Condo-Hotel"},
	"manufactured": {"Manufactured", "Mobile"},
	"multi_2_4":    {"Duplex", "Triplex", "Quadruplex"},
}

// subTypeToFamily is the reverse lookup: sub-type → family name.
var subTypeToFamily map[string]string

func init() {
	subTypeToFamily = make(map[string]string, 20)
	for family, subTypes := range productFamilies {
		for _, st := range subTypes {
			subTypeToFamily[st] = family
		}
	}
}

// classifySubType returns the product family name for a given sub-type, or "" if unknown.
func classifySubType(subType string) string {
	if subType == "" {
		return ""
	}
	return subTypeToFamily[subType]
}

// scoreSubTypeMatch returns a match quality label and a factor (0.0–1.0) representing
// how well the comp's sub-type matches the subject's sub-type.
func scoreSubTypeMatch(subjectSubType, compSubType string) (matchQuality string, factor float64) {
	if subjectSubType == "" || compSubType == "" {
		return "family_match", 0.6
	}
	if subjectSubType == compSubType {
		return "exact_subtype", 1.0
	}
	subjectFamily := classifySubType(subjectSubType)
	compFamily := classifySubType(compSubType)
	if subjectFamily != "" && subjectFamily == compFamily {
		return "family_match", 0.6
	}
	return "cross_family_low_confidence", 0.2
}

// computeRecencyFactor returns a 0.0–1.0 factor based on how recently a lease closed.
func computeRecencyFactor(closeDate *time.Time, maxLookbackDays int) float64 {
	if closeDate == nil || maxLookbackDays <= 0 {
		return 0.5
	}
	daysSince := time.Since(*closeDate).Hours() / 24
	if daysSince < 0 {
		daysSince = 0
	}
	factor := 1.0 - daysSince/float64(maxLookbackDays)
	if factor < 0 {
		return 0
	}
	return factor
}

// computeFinalSimilarity combines SQL-side similarity with Go-side sub-type and recency.
func computeFinalSimilarity(sqlScore, subTypeFactor, recencyFactor float64) float64 {
	return sqlScore*0.65 + subTypeFactor*0.20 + recencyFactor*0.15
}

// --- Tiered filtering ---

const (
	tier1MaxMiles  = 1.0
	tier1MaxDays   = 180 // 6 months
	tier2MaxMiles  = 3.0
	tier2MaxDays   = 365 // 12 months
	minClosedComps = 3
)

// filterAndRankRentalComps scores all rows and applies tiered filtering for closed comps.
func filterAndRankRentalComps(
	closedRows []compRow,
	activeRows []compRow,
	subject *resolvedSubject,
	params *RentalParamsInput,
) (closed []rankedRentalComp, active []rankedRentalComp, warnings []string) {
	subjectSubType := ""
	if subject.PropertySubType != nil {
		subjectSubType = *subject.PropertySubType
	}

	// Score all closed rows.
	allClosed := make([]rankedRentalComp, 0, len(closedRows))
	for _, r := range closedRows {
		allClosed = append(allClosed, rankComp(r, subjectSubType, true))
	}

	// Tiered filtering for closed comps.
	closed, fallbackMsg := applyTieredFiltering(allClosed, params.allowCrossFamily())
	if fallbackMsg != "" {
		warnings = append(warnings, fallbackMsg)
	}

	// Score and filter active rows (simpler: sub-type filtering only).
	for _, r := range activeRows {
		rc := rankComp(r, subjectSubType, false)
		if rc.matchQuality == "cross_family_low_confidence" && !params.allowCrossFamily() {
			continue
		}
		active = append(active, rc)
	}

	// Sort both by final score descending.
	sort.Slice(closed, func(i, j int) bool { return closed[i].finalScore > closed[j].finalScore })
	sort.Slice(active, func(i, j int) bool { return active[i].finalScore > active[j].finalScore })

	return closed, active, warnings
}

// rankComp scores a single comp row and determines its rent source.
func rankComp(r compRow, subjectSubType string, isClosed bool) rankedRentalComp {
	compSubType := ""
	if r.PropertySubType != nil {
		compSubType = *r.PropertySubType
	}

	matchQuality, subTypeFactor := scoreSubTypeMatch(subjectSubType, compSubType)
	recencyFactor := computeRecencyFactor(r.CloseDate, tier2MaxDays)
	if !isClosed {
		recencyFactor = 0.8 // active listings get a fixed recency boost
	}
	finalScore := computeFinalSimilarity(r.SimilarityScore, subTypeFactor, recencyFactor)

	// Determine rent source.
	var rent *float64
	var rentSource string
	if isClosed {
		if r.ClosePrice != nil && *r.ClosePrice > 0 {
			rent = r.ClosePrice
			rentSource = "close_price"
		} else if r.MfrLeasePrice != nil && *r.MfrLeasePrice > 0 {
			rent = r.MfrLeasePrice
			rentSource = "mfr_lease_price"
		}
	} else {
		if r.ListPrice != nil && *r.ListPrice > 0 {
			rent = r.ListPrice
			rentSource = "list_price"
		} else if r.MfrLeasePrice != nil && *r.MfrLeasePrice > 0 {
			rent = r.MfrLeasePrice
			rentSource = "mfr_lease_price"
		}
	}

	return rankedRentalComp{
		row:            r,
		matchQuality:   matchQuality,
		subTypeFactor:  subTypeFactor,
		recencyFactor:  recencyFactor,
		finalScore:     finalScore,
		rent:           rent,
		rentSource:     rentSource,
		distanceMiles:  r.DistanceMeters / 1609.344,
		isClosedLeased: isClosed,
	}
}

// applyTieredFiltering applies cumulative tier expansion until minClosedComps is reached.
func applyTieredFiltering(all []rankedRentalComp, allowCrossFamily bool) ([]rankedRentalComp, string) {
	// Tier 1: exact_subtype + distance ≤ 1mi + close_date within 6mo.
	tier1 := filterTier(all, []string{"exact_subtype"}, tier1MaxMiles, tier1MaxDays)
	if len(tier1) >= minClosedComps {
		return tier1, ""
	}

	// Tier 2: exact_subtype + distance ≤ 3mi + close_date within 12mo.
	tier2 := filterTier(all, []string{"exact_subtype"}, tier2MaxMiles, tier2MaxDays)
	if len(tier2) >= minClosedComps {
		return tier2, "Expanded search to 3mi/12mo for sufficient exact sub-type matches"
	}

	// Tier 3: exact_subtype + family_match, any distance/time in scope.
	tier3 := filterTier(all, []string{"exact_subtype", "family_match"}, 0, 0)
	if len(tier3) >= minClosedComps {
		return tier3, "Included family-match comps for sufficient sample size"
	}

	// Tier 4: add cross_family if allowed.
	if allowCrossFamily {
		tier4 := filterTier(all, []string{"exact_subtype", "family_match", "cross_family_low_confidence"}, 0, 0)
		if len(tier4) > 0 {
			return tier4, "Included cross-family comps (low confidence) for sufficient sample size"
		}
	}

	// Return best available.
	if len(tier3) > 0 {
		return tier3, "Limited rental comps available; results may have lower confidence"
	}
	if len(tier2) > 0 {
		return tier2, "Limited rental comps available; results may have lower confidence"
	}
	return all, "Very limited rental comps available; results have low confidence"
}

// filterTier filters ranked comps by allowed match qualities, max distance, and max days.
// Pass maxMiles=0 or maxDays=0 to disable that constraint.
func filterTier(all []rankedRentalComp, qualities []string, maxMiles float64, maxDays int) []rankedRentalComp {
	qualitySet := make(map[string]bool, len(qualities))
	for _, q := range qualities {
		qualitySet[q] = true
	}

	var result []rankedRentalComp
	for _, rc := range all {
		if !qualitySet[rc.matchQuality] {
			continue
		}
		if maxMiles > 0 && rc.distanceMiles > maxMiles {
			continue
		}
		if maxDays > 0 && rc.row.CloseDate != nil {
			daysSince := int(time.Since(*rc.row.CloseDate).Hours() / 24)
			if daysSince > maxDays {
				continue
			}
		}
		result = append(result, rc)
	}
	return result
}
