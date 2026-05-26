package comps

import "testing"

func TestFilterOutlierZScoreRemovesExtreme(t *testing.T) {
	sold := []CompRecord{
		{ClosePrice: 400000, LivingArea: 2000},
		{ClosePrice: 410000, LivingArea: 2050},
		{ClosePrice: 405000, LivingArea: 2020},
		{ClosePrice: 900000, LivingArea: 2100},
	}
	out := filterOutlierZScore(sold, 1.5)
	if len(out) >= len(sold) {
		t.Fatalf("expected outlier removed, got %d", len(out))
	}
}
