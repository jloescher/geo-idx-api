package comps

import (
	"fmt"
	"math"
	"strings"
)

type bpoReconcileResult struct {
	PointEstimate          float64
	Low                    float64
	High                   float64
	Confidence             float64
	ConfidenceBand         string
	ReconciliationSummary  string
}

// reconcileBPO computes weighted indicated value from URAR-adjusted comps.
func reconcileBPO(subject SubjectProfile, sold []CompRecord, grids []BpoCompGrid, rates BpoMarketRates) bpoReconcileResult {
	if len(grids) == 0 {
		return bpoReconcileResult{}
	}
	weights := make([]float64, len(grids))
	var wSum float64
	for i, g := range grids {
		c := sold[i]
		w := compReconcileWeight(subject, c, g, rates)
		weights[i] = w
		wSum += w
	}
	if wSum <= 0 {
		for i := range weights {
			weights[i] = 1
		}
		wSum = float64(len(weights))
	}
	var indicated float64
	for i, g := range grids {
		indicated += (weights[i] / wSum) * g.AdjustedPrice
	}
	indicated = round2(indicated)
	conf := reconcileConfidence(grids, rates)
	spread := spreadPct(conf)
	low := round2(indicated * (1 - spread))
	high := round2(indicated * (1 + spread))
	band := confidenceBand(conf)
	summary := fmt.Sprintf(
		"Weighted average of %d adjusted comps (method: %s, R²: %.2f). Strongest weight to lowest gross-adjustment / closest GLA comps.",
		len(grids), rates.Method, rates.RSquared,
	)
	return bpoReconcileResult{
		PointEstimate:         indicated,
		Low:                   low,
		High:                  high,
		Confidence:            conf,
		ConfidenceBand:        band,
		ReconciliationSummary: summary,
	}
}

func compReconcileWeight(subject SubjectProfile, comp CompRecord, g BpoCompGrid, rates BpoMarketRates) float64 {
	const (
		wProx = 0.25
		wGLA  = 0.25
		wRec  = 0.20
		wGross = 0.20
		wFeat = 0.10
	)
	prox := proximityScore(comp.DistanceMiles)
	gla := glaSimilarity(subject.LivingArea, comp.LivingArea)
	rec := recencyScore(comp.CloseDate)
	gross := 1 - math.Min(g.GrossAdjustmentPct/25, 1)
	feat := featureMatchScore(subject, comp)
	return wProx*prox + wGLA*gla + wRec*rec + wGross*gross + wFeat*feat
}

func proximityScore(miles float64) float64 {
	if miles <= 0.25 {
		return 1
	}
	if miles >= 3 {
		return 0.1
	}
	return math.Max(0.1, 1-miles/3)
}

func glaSimilarity(subGLA, compGLA float64) float64 {
	if subGLA <= 0 || compGLA <= 0 {
		return 0.5
	}
	diff := math.Abs(subGLA-compGLA) / subGLA
	return math.Max(0, 1-diff)
}

func recencyScore(closeDate string) float64 {
	mo := monthsSinceClose(closeDate)
	if mo <= 3 {
		return 1
	}
	if mo >= 18 {
		return 0.2
	}
	return math.Max(0.2, 1-mo/18)
}

func featureMatchScore(subject SubjectProfile, comp CompRecord) float64 {
	score := 0.0
	n := 0.0
	if subject.Subdivision != "" {
		n++
		if strings.EqualFold(subject.Subdivision, compSubdivision(comp)) {
			score++
		}
	}
	if subject.PoolPrivate == comp.PoolPrivate {
		score += 0.5
		n += 0.5
	}
	if subject.Waterfront == comp.Waterfront {
		score += 0.5
		n += 0.5
	}
	if n == 0 {
		return 0.7
	}
	return score / n
}

func reconcileConfidence(grids []BpoCompGrid, rates BpoMarketRates) float64 {
	n := float64(len(grids))
	base := 40.0
	if n >= 6 {
		base = 70
	} else if n >= 3 {
		base = 55
	}
	base += rates.RSquared * 20
	if rates.Method == bpoRateSourceOLS {
		base += 5
	}
	var avgGross float64
	for _, g := range grids {
		avgGross += g.GrossAdjustmentPct
	}
	if len(grids) > 0 {
		avgGross /= float64(len(grids))
	}
	if avgGross > 20 {
		base -= 10
	}
	return math.Min(95, math.Max(20, round2(base)))
}

func spreadPct(confidence float64) float64 {
	if confidence >= 80 {
		return 0.03
	}
	if confidence >= 60 {
		return 0.05
	}
	return 0.08
}

func confidenceBand(conf float64) string {
	switch {
	case conf >= 75:
		return "high"
	case conf >= 50:
		return "moderate"
	default:
		return "low"
	}
}
