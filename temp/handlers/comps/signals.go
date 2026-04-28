package comps

import (
	"fmt"
	"math"
	"sort"
)

// computeOverpricedSignals generates overpriced indicators from the fetched results.
func computeOverpricedSignals(
	soldComps []SoldComp,
	competitionComps []CompetitionComp,
	failedListings []FailedListing,
	method AggregationMethod,
) []OverpricedSignal {
	var signals []OverpricedSignal

	// Compute central tendency DOM and PPSF from sold comps.
	medianDOM := computeCentralDOM(soldComps, method)
	medianPPSF := computeCentralPPSF(soldComps, method)

	// 1. DOM above median: flag competition comps with high DOM.
	if medianDOM > 0 {
		for _, c := range competitionComps {
			if c.DOM == nil {
				continue
			}
			dom := float64(*c.DOM)
			if dom > medianDOM {
				severity := "low"
				if dom > medianDOM*2 {
					severity = "high"
				} else if dom > medianDOM*1.5 {
					severity = "moderate"
				}
				signals = append(signals, OverpricedSignal{
					ListingID: c.ListingID,
					Indicator: "dom_above_median",
					Detail:    fmt.Sprintf("DOM %d vs median %d", *c.DOM, int(medianDOM)),
					Severity:  severity,
				})
			}
		}
	}

	// 2. Expired similar: flag failed listings with high similarity.
	for _, fl := range failedListings {
		detail := fmt.Sprintf("Similar property %s at", fl.StandardStatus)
		if fl.LastListPrice != nil {
			detail += fmt.Sprintf(" $%s", formatPrice(*fl.LastListPrice))
		}
		if fl.DOM != nil {
			detail += fmt.Sprintf(" after %d DOM", *fl.DOM)
		}
		if fl.PriceReductions > 0 {
			detail += fmt.Sprintf(" with %d price reduction(s)", fl.PriceReductions)
		}

		severity := "moderate"
		if fl.SimilarityScore > 85 {
			severity = "high"
		} else if fl.SimilarityScore < 70 {
			severity = "low"
		}

		signals = append(signals, OverpricedSignal{
			ListingID: fl.ListingID,
			Indicator: "expired_similar",
			Detail:    detail,
			Severity:  severity,
		})
	}

	// 3. Above median PPSF: flag competition comps priced above sold median.
	if medianPPSF > 0 {
		for _, c := range competitionComps {
			if c.PPSF == nil {
				continue
			}
			if *c.PPSF > medianPPSF*1.15 { // 15% above median
				severity := "low"
				pctAbove := (*c.PPSF - medianPPSF) / medianPPSF * 100
				if pctAbove > 30 {
					severity = "high"
				} else if pctAbove > 15 {
					severity = "moderate"
				}
				signals = append(signals, OverpricedSignal{
					ListingID: c.ListingID,
					Indicator: "above_median_ppsf",
					Detail:    fmt.Sprintf("PPSF $%.0f vs sold median $%.0f (%.0f%% above)", *c.PPSF, medianPPSF, pctAbove),
					Severity:  severity,
				})
			}
		}
	}

	if signals == nil {
		signals = []OverpricedSignal{}
	}
	return signals
}

func computeCentralDOM(comps []SoldComp, method AggregationMethod) float64 {
	var doms []float64
	for _, c := range comps {
		if c.DOM != nil {
			doms = append(doms, float64(*c.DOM))
		}
	}
	return aggregate(doms, method)
}

func computeCentralPPSF(comps []SoldComp, method AggregationMethod) float64 {
	var ppsfs []float64
	for _, c := range comps {
		if c.PPSF != nil {
			ppsfs = append(ppsfs, *c.PPSF)
		}
	}
	return aggregate(ppsfs, method)
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	n := len(values)
	if n%2 == 0 {
		return (values[n/2-1] + values[n/2]) / 2
	}
	return values[n/2]
}

// AggregationMethod controls whether market data uses median or mean aggregation.
type AggregationMethod string

const (
	AggMedian  AggregationMethod = "median"
	AggAverage AggregationMethod = "average"
)

// mean computes the arithmetic mean of a float64 slice.
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// aggregate dispatches to median or mean based on the method.
func aggregate(values []float64, method AggregationMethod) float64 {
	if method == AggAverage {
		return mean(values)
	}
	return median(values)
}

// parseAggregationMethod validates and returns the aggregation method.
func parseAggregationMethod(s string) (AggregationMethod, error) {
	switch s {
	case "", "median":
		return AggMedian, nil
	case "average":
		return AggAverage, nil
	default:
		return "", fmt.Errorf("aggregation_method must be 'median' or 'average'")
	}
}

func formatPrice(price float64) string {
	p := math.Round(price)
	if p >= 1_000_000 {
		return fmt.Sprintf("%.0fM", p/1_000_000)
	}
	if p >= 1_000 {
		return fmt.Sprintf("%.0fk", p/1_000)
	}
	return fmt.Sprintf("%.0f", p)
}
