package comps

import (
	"math"
	"sort"
	"time"
)

const (
	bpoRateSourceOLS    = "ols_regression"
	bpoRateSourcePaired = "paired_sales"
	bpoRateSourceMedian = "median_fallback"
)

// extractMarketRates derives adjustment rates from sold comps (OLS, paired-sales, or median PPSF).
func extractMarketRates(sold []CompRecord, minOLS int) BpoMarketRates {
	rates := BpoMarketRates{
		CompCount: len(sold),
		Method:    bpoRateSourceMedian,
		Warnings:  []string{},
	}
	if minOLS <= 0 {
		minOLS = 6
	}
	if len(sold) == 0 {
		rates.Warnings = append(rates.Warnings, "no sold comps")
		return rates
	}
	rates.MedianGLA = medianGLA(sold)
	if len(sold) >= minOLS {
		if ols := fitBpoOLS(sold); ols.ok {
			rates = ols.rates
			sanitizeMarketRates(&rates)
			rates.CompCount = len(sold)
			rates.MedianGLA = medianGLA(sold)
			return rates
		}
		rates.Warnings = append(rates.Warnings, "OLS regression failed; using paired-sales or median fallback")
	}
	if len(sold) >= 3 {
		paired := extractPairedRates(sold)
		rates.GLAPerSF = paired.gla
		rates.BedPerRoom = paired.bed
		rates.BathPerFull = paired.bath
		rates.AgePerYear = paired.age
		rates.LotPerAcre = paired.lot
		rates.GaragePerSpace = paired.garage
		rates.PoolValue = paired.pool
		rates.WaterfrontValue = paired.waterfront
		rates.Method = bpoRateSourcePaired
		if rates.GLAPerSF <= 0 {
			rates.GLAPerSF = medianPPSF(sold)
		}
		fillDefaultRates(&rates)
		sanitizeMarketRates(&rates)
		return rates
	}
	rates.GLAPerSF = medianPPSF(sold)
	fillDefaultRates(&rates)
	sanitizeMarketRates(&rates)
	rates.Warnings = append(rates.Warnings, "insufficient comps for regression; median PPSF only")
	return rates
}

type olsFitResult struct {
	ok    bool
	rates BpoMarketRates
}

func fitBpoOLS(sold []CompRecord) olsFitResult {
	// ClosePrice ~ LivingArea + Beds + Baths + YearBuilt + Lot + Garage + Pool + Waterfront + monthsSinceClose
	const nFeat = 9
	var rows [][]float64
	var y []float64
	meanMonths := meanCloseMonths(sold)
	for _, c := range sold {
		price := c.ClosePrice
		if price <= 0 || c.LivingArea <= 0 {
			continue
		}
		mo := monthsSinceClose(c.CloseDate) - meanMonths
		rows = append(rows, []float64{
			c.LivingArea,
			c.Bedrooms,
			c.Bathrooms,
			float64(c.YearBuilt),
			c.LotSizeAcres,
			c.GarageSpaces,
			bool01(c.PoolPrivate),
			bool01(c.Waterfront),
			mo,
		})
		y = append(y, price)
	}
	if len(rows) < 6 {
		return olsFitResult{}
	}
	coef, ok := solveOLS(rows, y)
	if !ok || len(coef) != nFeat+1 {
		return olsFitResult{}
	}
	gla := coef[1]
	if gla <= 0 {
		return olsFitResult{}
	}
	r2 := olsRSquared(rows, y, coef)
	return olsFitResult{
		ok: true,
		rates: BpoMarketRates{
			Intercept:       coef[0],
			GLAPerSF:        gla,
			BedPerRoom:      coef[2],
			BathPerFull:     coef[3],
			AgePerYear:      math.Abs(coef[4]),
			LotPerAcre:      coef[5],
			GaragePerSpace:  coef[6],
			PoolValue:       coef[7],
			WaterfrontValue: coef[8],
			TimePerMonthPct: coef[9] / 100.0,
			RSquared:        r2,
			Method:          bpoRateSourceOLS,
		},
	}
}

func solveOLS(X [][]float64, y []float64) ([]float64, bool) {
	n := len(X)
	if n == 0 {
		return nil, false
	}
	p := len(X[0]) + 1
	// Design matrix with intercept column.
	xtX := make([][]float64, p)
	for i := range xtX {
		xtX[i] = make([]float64, p)
	}
	xtY := make([]float64, p)
	for i := 0; i < n; i++ {
		row := make([]float64, p)
		row[0] = 1
		copy(row[1:], X[i])
		for a := 0; a < p; a++ {
			for b := 0; b < p; b++ {
				xtX[a][b] += row[a] * row[b]
			}
			xtY[a] += row[a] * y[i]
		}
	}
	inv, ok := invertMatrix(xtX)
	if !ok {
		return nil, false
	}
	coef := make([]float64, p)
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			coef[i] += inv[i][j] * xtY[j]
		}
	}
	return coef, true
}

func invertMatrix(m [][]float64) ([][]float64, bool) {
	n := len(m)
	aug := make([][]float64, n)
	for i := 0; i < n; i++ {
		aug[i] = make([]float64, 2*n)
		copy(aug[i], m[i])
		aug[i][n+i] = 1
	}
	for col := 0; col < n; col++ {
		pivot := col
		for r := col + 1; r < n; r++ {
			if math.Abs(aug[r][col]) > math.Abs(aug[pivot][col]) {
				pivot = r
			}
		}
		if math.Abs(aug[pivot][col]) < 1e-12 {
			return nil, false
		}
		aug[col], aug[pivot] = aug[pivot], aug[col]
		div := aug[col][col]
		for j := 0; j < 2*n; j++ {
			aug[col][j] /= div
		}
		for r := 0; r < n; r++ {
			if r == col {
				continue
			}
			f := aug[r][col]
			for j := 0; j < 2*n; j++ {
				aug[r][j] -= f * aug[col][j]
			}
		}
	}
	inv := make([][]float64, n)
	for i := 0; i < n; i++ {
		inv[i] = aug[i][n:]
	}
	return inv, true
}

func olsRSquared(X [][]float64, y []float64, coef []float64) float64 {
	if len(y) == 0 {
		return 0
	}
	mean := 0.0
	for _, v := range y {
		mean += v
	}
	mean /= float64(len(y))
	var ssTot, ssRes float64
	for i, row := range X {
		pred := coef[0]
		for j, v := range row {
			pred += coef[j+1] * v
		}
		ssRes += (y[i] - pred) * (y[i] - pred)
		ssTot += (y[i] - mean) * (y[i] - mean)
	}
	if ssTot < 1e-9 {
		return 0
	}
	return math.Max(0, 1-ssRes/ssTot)
}

type pairedRates struct {
	gla, bed, bath, age, lot, garage, pool, waterfront float64
}

func extractPairedRates(sold []CompRecord) pairedRates {
	var pr pairedRates
	pr.gla = pairedSalesPPSF(sold)
	pr.pool = pairedBinaryDelta(sold, func(c CompRecord) bool { return c.PoolPrivate })
	pr.waterfront = pairedBinaryDelta(sold, func(c CompRecord) bool { return c.Waterfront })
	pr.garage = pairedContinuousDelta(sold, func(c CompRecord) float64 { return c.GarageSpaces }, 0.5)
	pr.bed = pairedContinuousDelta(sold, func(c CompRecord) float64 { return c.Bedrooms }, 0.25)
	pr.bath = pairedContinuousDelta(sold, func(c CompRecord) float64 { return c.Bathrooms }, 0.25)
	pr.age = pairedContinuousDelta(sold, func(c CompRecord) float64 { return float64(c.YearBuilt) }, 3)
	pr.lot = pairedContinuousDelta(sold, func(c CompRecord) float64 { return c.LotSizeAcres }, 0.1)
	return pr
}

func pairedBinaryDelta(sold []CompRecord, feat func(CompRecord) bool) float64 {
	var deltas []float64
	for i := 0; i < len(sold); i++ {
		for j := i + 1; j < len(sold); j++ {
			a, b := sold[i], sold[j]
			if !compsAlignedForPair(a, b) {
				continue
			}
			fa, fb := feat(a), feat(b)
			if fa == fb {
				continue
			}
			d := math.Abs(compPrice(a) - compPrice(b))
			if fa {
				d = compPrice(a) - compPrice(b)
			} else {
				d = compPrice(b) - compPrice(a)
			}
			if d > 0 {
				deltas = append(deltas, d)
			}
		}
	}
	return clamp(medianOf(deltas), 0, 80000)
}

func pairedContinuousDelta(sold []CompRecord, feat func(CompRecord) float64, minDiff float64) float64 {
	var rates []float64
	for i := 0; i < len(sold); i++ {
		for j := i + 1; j < len(sold); j++ {
			a, b := sold[i], sold[j]
			if !compsAlignedForPair(a, b) {
				continue
			}
			d := feat(a) - feat(b)
			if math.Abs(d) < minDiff {
				continue
			}
			pd := compPrice(a) - compPrice(b)
			if pd == 0 {
				continue
			}
			rates = append(rates, pd/d)
		}
	}
	return medianOf(rates)
}

func compsAlignedForPair(a, b CompRecord) bool {
	if a.LivingArea <= 0 || b.LivingArea <= 0 {
		return false
	}
	if math.Abs(a.LivingArea-b.LivingArea) > 200 {
		return false
	}
	if abs(a.Bedrooms-b.Bedrooms) > 1 || abs(a.Bathrooms-b.Bathrooms) > 0.5 {
		return false
	}
	return true
}

func sanitizeMarketRates(r *BpoMarketRates) {
	r.GLAPerSF = clamp(r.GLAPerSF, 40, 900)
	r.BedPerRoom = clamp(r.BedPerRoom, 0, 25000)
	r.BathPerFull = clamp(r.BathPerFull, 0, 20000)
	r.AgePerYear = clamp(r.AgePerYear, 0, 6000)
	r.LotPerAcre = clamp(r.LotPerAcre, 0, 150000)
	r.GaragePerSpace = clamp(r.GaragePerSpace, 0, 35000)
	r.PoolValue = clamp(r.PoolValue, -50000, 80000)
	r.WaterfrontValue = clamp(r.WaterfrontValue, -100000, 200000)
	if r.TimePerMonthPct < 0 {
		r.TimePerMonthPct = -r.TimePerMonthPct
	}
	if r.TimePerMonthPct > 0.02 {
		r.TimePerMonthPct = 0.02
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func fillDefaultRates(r *BpoMarketRates) {
	if r.BedPerRoom == 0 {
		r.BedPerRoom = 8000
	}
	if r.BathPerFull == 0 {
		r.BathPerFull = 5000
	}
	if r.AgePerYear == 0 {
		r.AgePerYear = 1500
	}
	if r.LotPerAcre == 0 {
		r.LotPerAcre = 25000
	}
	if r.GaragePerSpace == 0 {
		r.GaragePerSpace = 10000
	}
	if r.PoolValue == 0 {
		r.PoolValue = 20000
	}
	if r.WaterfrontValue == 0 {
		r.WaterfrontValue = 50000
	}
}

func medianPPSF(sold []CompRecord) float64 {
	var vals []float64
	for _, c := range sold {
		if c.LivingArea > 0 && c.ClosePrice > 0 {
			vals = append(vals, c.ClosePrice/c.LivingArea)
		}
	}
	return medianOf(vals)
}

func medianGLA(sold []CompRecord) float64 {
	var v []float64
	for _, c := range sold {
		if c.LivingArea > 0 {
			v = append(v, c.LivingArea)
		}
	}
	return medianOf(v)
}

func medianOf(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sort.Float64s(vals)
	mid := len(vals) / 2
	if len(vals)%2 == 0 {
		return (vals[mid-1] + vals[mid]) / 2
	}
	return vals[mid]
}

func meanCloseMonths(sold []CompRecord) float64 {
	var sum float64
	var n int
	for _, c := range sold {
		m := monthsSinceClose(c.CloseDate)
		sum += m
		n++
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func monthsSinceClose(closeDate string) float64 {
	if closeDate == "" {
		return 0
	}
	t, err := time.Parse("2006-01-02", closeDate[:min(10, len(closeDate))])
	if err != nil {
		return 0
	}
	days := time.Since(t).Hours() / 24
	return days / 30.437
}

func bool01(v bool) float64 {
	if v {
		return 1
	}
	return 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
