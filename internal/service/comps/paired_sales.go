package comps

// pairedSalesPPSF estimates $/sqft from comp pairs that differ primarily in living area.
func pairedSalesPPSF(sold []CompRecord) float64 {
	if len(sold) < 3 {
		return 0
	}
	var rates []float64
	for i := 0; i < len(sold); i++ {
		for j := i + 1; j < len(sold); j++ {
			a, b := sold[i], sold[j]
			if a.LivingArea <= 0 || b.LivingArea <= 0 {
				continue
			}
			areaDiff := abs(a.LivingArea - b.LivingArea)
			if areaDiff < 50 {
				continue
			}
			priceDiff := abs(compPrice(a) - compPrice(b))
			if priceDiff <= 0 {
				continue
			}
			// Require other attributes roughly aligned.
			if abs(a.Bedrooms-b.Bedrooms) > 1 || abs(a.Bathrooms-b.Bathrooms) > 0.5 {
				continue
			}
			rates = append(rates, priceDiff/areaDiff)
		}
	}
	if len(rates) == 0 {
		return 0
	}
	sum := 0.0
	for _, r := range rates {
		sum += r
	}
	return sum / float64(len(rates))
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
