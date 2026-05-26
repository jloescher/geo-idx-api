package comps

import "math"

func filterOutlierZScore(sold []CompRecord, threshold float64) []CompRecord {
	if threshold <= 0 || len(sold) < 3 {
		return sold
	}
	prices := make([]float64, 0, len(sold))
	for _, c := range sold {
		p := compPrice(c)
		if p > 0 {
			prices = append(prices, p)
		}
	}
	if len(prices) < 3 {
		return sold
	}
	mean, std := meanStd(prices)
	if std < 1e-9 {
		return sold
	}
	var out []CompRecord
	for _, c := range sold {
		p := compPrice(c)
		if p <= 0 {
			continue
		}
		z := math.Abs((p - mean) / std)
		if z <= threshold {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return sold
	}
	return out
}

func compPrice(c CompRecord) float64 {
	if c.AdjustedPrice > 0 {
		return c.AdjustedPrice
	}
	return c.ClosePrice
}

func meanStd(vals []float64) (mean, std float64) {
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	mean = sum / float64(len(vals))
	var varSum float64
	for _, v := range vals {
		d := v - mean
		varSum += d * d
	}
	std = math.Sqrt(varSum / float64(len(vals)))
	return mean, std
}
