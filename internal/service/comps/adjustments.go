package comps

import (
	"math"
	"sort"
)

func applyAdjustments(subject SubjectProfile, comps []CompRecord, f FiltersInput) []CompRecord {
	poolAdj := 15000.0
	if f.AdjPoolValue != nil {
		poolAdj = *f.AdjPoolValue
	}
	waterAdj := 50000.0
	if f.AdjWaterfrontValue != nil {
		waterAdj = *f.AdjWaterfrontValue
	}
	garageAdj := 5000.0
	if f.AdjGaragePerSpace != nil {
		garageAdj = *f.AdjGaragePerSpace
	}
	yearAdj := 1500.0
	if f.AdjYearBuiltPerYear != nil {
		yearAdj = *f.AdjYearBuiltPerYear
	}
	out := make([]CompRecord, len(comps))
	for i, c := range comps {
		base := c.ClosePrice
		if base <= 0 {
			base = c.ListPrice
		}
		var lines []AdjustmentLine
		if subject.LivingArea > 0 && c.LivingArea > 0 {
			diff := subject.LivingArea - c.LivingArea
			ppsf := base / c.LivingArea
			amt := diff * ppsf
			lines = append(lines, AdjustmentLine{Feature: "living_area_sqft", Amount: round2(amt)})
		}
		if subject.Bedrooms != c.Bedrooms {
			amt := (subject.Bedrooms - c.Bedrooms) * 8000
			lines = append(lines, AdjustmentLine{Feature: "bedrooms", Amount: round2(amt)})
		}
		if subject.Bathrooms != c.Bathrooms {
			amt := (subject.Bathrooms - c.Bathrooms) * 5000
			lines = append(lines, AdjustmentLine{Feature: "bathrooms", Amount: round2(amt)})
		}
		if subject.PoolPrivate != c.PoolPrivate {
			amt := 0.0
			if subject.PoolPrivate {
				amt = poolAdj
			} else {
				amt = -poolAdj
			}
			lines = append(lines, AdjustmentLine{Feature: "pool", Amount: round2(amt)})
		}
		if subject.Waterfront != c.Waterfront {
			amt := 0.0
			if subject.Waterfront {
				amt = waterAdj
			} else {
				amt = -waterAdj
			}
			lines = append(lines, AdjustmentLine{Feature: "waterfront", Amount: round2(amt)})
		}
		if subject.GarageSpaces != c.GarageSpaces {
			amt := (subject.GarageSpaces - c.GarageSpaces) * garageAdj
			lines = append(lines, AdjustmentLine{Feature: "garage", Amount: round2(amt)})
		}
		if subject.YearBuilt > 0 && c.YearBuilt > 0 {
			amt := float64(subject.YearBuilt-c.YearBuilt) * yearAdj
			lines = append(lines, AdjustmentLine{Feature: "year_built", Amount: round2(amt)})
		}
		adj := base
		for _, l := range lines {
			adj += l.Amount
		}
		c.Adjustments = lines
		c.AdjustedPrice = round2(adj)
		out[i] = c
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].DistanceMiles < out[j].DistanceMiles
	})
	return out
}

func marketConditions(sold []CompRecord, months int) map[string]any {
	if len(sold) == 0 {
		return map[string]any{"sold_count": 0, "months_back": months}
	}
	var sum, min, max float64
	min = sold[0].ClosePrice
	for _, c := range sold {
		p := c.ClosePrice
		if c.AdjustedPrice > 0 {
			p = c.AdjustedPrice
		}
		sum += p
		if p < min {
			min = p
		}
		if p > max {
			max = p
		}
	}
	return map[string]any{
		"sold_count":       len(sold),
		"months_back":      months,
		"median_close":     medianPrice(sold),
		"avg_adjusted":     round2(sum / float64(len(sold))),
		"min_close":        min,
		"max_close":        max,
	}
}

func medianPrice(sold []CompRecord) float64 {
	vals := make([]float64, 0, len(sold))
	for _, c := range sold {
		p := c.ClosePrice
		if p > 0 {
			vals = append(vals, p)
		}
	}
	if len(vals) == 0 {
		return 0
	}
	sort.Float64s(vals)
	mid := len(vals) / 2
	if len(vals)%2 == 0 {
		return round2((vals[mid-1] + vals[mid]) / 2)
	}
	return round2(vals[mid])
}

func overpricedSignals(competition []CompRecord, sold []CompRecord) []map[string]any {
	med := medianPrice(sold)
	if med <= 0 {
		return nil
	}
	var out []map[string]any
	for _, c := range competition {
		if c.ListPrice > med*1.1 {
			out = append(out, map[string]any{
				"listing_key": c.ListingKey,
				"list_price":  c.ListPrice,
				"threshold":   round2(med * 1.1),
				"premium_pct": round2((c.ListPrice/med - 1) * 100),
			})
		}
	}
	return out
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
