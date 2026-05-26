package mls

import "math"

// PostgreSQL NUMERIC column bounds for listings mirror indexed columns.
const (
	maxNumeric14_2 = 999999999999.99
	minNumeric14_2 = -999999999999.99
	maxNumeric12_2 = 9999999999.99
	minNumeric12_2 = -9999999999.99
	maxNumeric12_4 = 99999999.9999
	minNumeric12_4 = -99999999.9999
	maxBathroomsDecimal = 99.99
	maxBathroomInteger  = 30
)

// ResolveListPrice returns the mirror list_price from RESO fields (never omitted when ok).
// Order: ListPrice → PreviousListPrice → OriginalListPrice.
func ResolveListPrice(row map[string]any) (float64, bool) {
	for _, key := range []string{"ListPrice", "PreviousListPrice", "OriginalListPrice"} {
		if v, ok := numericValue(row[key]); ok {
			return clampNumeric14_2(v), true
		}
	}
	return 0, false
}

// ClampNumeric14_2Ptr clamps an optional price-like value to NUMERIC(14,2) bounds.
func ClampNumeric14_2Ptr(v any) *float64 {
	f, ok := numericValue(v)
	if !ok {
		return nil
	}
	out := clampNumeric14_2(f)
	return &out
}

// ClampNumeric12_4Ptr clamps lot size acres to NUMERIC(12,4).
func ClampNumeric12_4Ptr(v any) *float64 {
	f, ok := numericValue(v)
	if !ok {
		return nil
	}
	out := clampRange(f, minNumeric12_4, maxNumeric12_4)
	return &out
}

// ResolveLivingAreaSqft returns living area in square feet from LivingArea then BuildingAreaTotal.
func ResolveLivingAreaSqft(row map[string]any) *float64 {
	if v := clampNumeric12_2Ptr(row["LivingArea"]); v != nil {
		return v
	}
	return clampNumeric12_2Ptr(row["BuildingAreaTotal"])
}

// BathroomsTotal returns a plausible bathroom count for bathrooms_total_decimal NUMERIC(5,2).
func BathroomsTotal(row map[string]any) *float64 {
	if v := float64Ptr(row["BathroomsTotalDecimal"]); v != nil {
		return clampBathroomsPtr(*v)
	}
	if v := float64Ptr(row["BathroomsTotalInteger"]); v != nil {
		if *v < 0 || *v > maxBathroomInteger {
			return nil
		}
		return clampBathroomsPtr(*v)
	}
	return nil
}

// BoundedInt16Ptr parses a RESO integer field safe for PostgreSQL SMALLINT.
func BoundedInt16Ptr(v any) *int16 {
	f, ok := numericValue(v)
	if !ok {
		return nil
	}
	if f < float64(math.MinInt16) || f > float64(math.MaxInt16) {
		return nil
	}
	i := int16(math.Round(f))
	return &i
}

func clampNumeric14_2(v float64) float64 {
	return clampRange(v, minNumeric14_2, maxNumeric14_2)
}

func clampNumeric12_2Ptr(v any) *float64 {
	f, ok := numericValue(v)
	if !ok {
		return nil
	}
	out := clampRange(f, minNumeric12_2, maxNumeric12_2)
	return &out
}

func clampBathroomsPtr(v float64) *float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return nil
	}
	out := clampRange(v, 0, maxBathroomsDecimal)
	return &out
}

func clampRange(v, min, max float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
