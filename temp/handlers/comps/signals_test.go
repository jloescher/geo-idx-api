package comps

import (
	"math"
	"testing"
	"time"
)

func TestMean(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
	}{
		{"empty", nil, 0},
		{"single", []float64{5}, 5},
		{"odd", []float64{3, 1, 2}, 2},
		{"even", []float64{4, 2, 1, 3}, 2.5},
		{"with_outlier", []float64{1, 2, 3, 100}, 26.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mean(tt.values)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("mean() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestAggregate_Median(t *testing.T) {
	vals := []float64{3, 1, 4, 1, 5}
	got := aggregate(vals, AggMedian)
	// sorted: [1,1,3,4,5], median = 3
	if got != 3 {
		t.Errorf("aggregate(median) = %f, want 3", got)
	}
}

func TestAggregate_Average(t *testing.T) {
	vals := []float64{3, 1, 4, 1, 5}
	got := aggregate(vals, AggAverage)
	// mean = 14/5 = 2.8
	if math.Abs(got-2.8) > 0.001 {
		t.Errorf("aggregate(average) = %f, want 2.8", got)
	}
}

func TestAggregate_DifferentResults(t *testing.T) {
	// Skewed data where median != mean
	vals := []float64{1, 2, 3, 4, 100}
	med := aggregate(vals, AggMedian)
	avg := aggregate(vals, AggAverage)
	if med == avg {
		t.Errorf("expected median (%f) != average (%f) for skewed data", med, avg)
	}
	if med != 3 {
		t.Errorf("median = %f, want 3", med)
	}
	if math.Abs(avg-22) > 0.001 {
		t.Errorf("average = %f, want 22", avg)
	}
}

func TestParseAggregationMethod_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  AggregationMethod
	}{
		{"", AggMedian},
		{"median", AggMedian},
		{"average", AggAverage},
	}
	for _, tt := range tests {
		got, err := parseAggregationMethod(tt.input)
		if err != nil {
			t.Errorf("parseAggregationMethod(%q) unexpected error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("parseAggregationMethod(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseAggregationMethod_Invalid(t *testing.T) {
	invalids := []string{"mean", "mode", "percentile", "avg", "MEDIAN"}
	for _, s := range invalids {
		_, err := parseAggregationMethod(s)
		if err == nil {
			t.Errorf("parseAggregationMethod(%q) should have returned error", s)
		}
	}
}

func TestExtractMarketRates_Average(t *testing.T) {
	// With average, outliers pull the result differently than median.
	var rows []compRow
	// 3 pool comps: $300k, $300k, $500k (outlier)
	for _, p := range []float64{300000, 300000, 500000} {
		rows = append(rows, compRow{ClosePrice: &p, PoolPrivateYn: ptr(true), LivingArea: ptr(1500)})
	}
	// 3 no-pool comps: $280k, $280k, $280k
	for i := 0; i < 3; i++ {
		p := 280000.0
		rows = append(rows, compRow{ClosePrice: &p, PoolPrivateYn: ptr(false), LivingArea: ptr(1500)})
	}

	medRates := extractMarketRates(rows, 200.0, AggMedian)
	avgRates := extractMarketRates(rows, 200.0, AggAverage)

	// Median pool: 300k, median no-pool: 280k → diff = 20k
	if medRates.PoolValue != 20000 {
		t.Errorf("median PoolValue = %.0f, want 20000", medRates.PoolValue)
	}
	// Average pool: (300+300+500)/3 = 366.67k, avg no-pool: 280k → diff ≈ 86.67k, clamped to 75k
	if avgRates.PoolValue != clampPoolMax {
		t.Errorf("average PoolValue = %.0f, want %.0f (clamped)", avgRates.PoolValue, clampPoolMax)
	}
}

func TestComputeQuarterlyTrend_Average(t *testing.T) {
	now := time.Now()
	recent := now.AddDate(0, -1, 0)
	prior := now.AddDate(0, -5, 0)

	rows := []compRow{
		// Recent: $200/sqft and $300/sqft (mean=250, median=250)
		{ClosePrice: ptr(300000.0), LivingArea: ptr(1500), CloseDate: &recent},
		{ClosePrice: ptr(450000.0), LivingArea: ptr(1500), CloseDate: &recent},
		// Prior: $180/sqft
		{ClosePrice: ptr(270000.0), LivingArea: ptr(1500), CloseDate: &prior},
		{ClosePrice: ptr(270000.0), LivingArea: ptr(1500), CloseDate: &prior},
	}

	trendMedian := computeQuarterlyTrend(rows, AggMedian)
	trendAvg := computeQuarterlyTrend(rows, AggAverage)

	// Both should be positive (recent > prior)
	if trendMedian <= 0 {
		t.Errorf("median trend should be positive, got %.4f", trendMedian)
	}
	if trendAvg <= 0 {
		t.Errorf("average trend should be positive, got %.4f", trendAvg)
	}
}
