package mls

import (
	"strings"
)

// BeachesSparkFloodZoneField is the RESO field for flood zone on BeachesMLS (Spark replication).
const BeachesSparkFloodZoneField = "Location_sp_and_sp_Legal_co_Flood_sp_Zone2"

// ResoFieldResolver maps provider-specific RESO fields to mirror columns.
type ResoFieldResolver struct {
	fees *AssociationFeeMonthlyNormalizer
}

func NewResoFieldResolver() *ResoFieldResolver {
	return &ResoFieldResolver{fees: NewAssociationFeeMonthlyNormalizer()}
}

func (r *ResoFieldResolver) ResolveFloodZoneCode(row map[string]any, provider MirrorProvider, datasetUpper string) *string {
	spark := stringValue(row[BeachesSparkFloodZoneField])
	bridgeExt := stringValue(row[strings.ToUpper(datasetUpper)+"_FloodZoneCode"])
	generic := stringValue(row["FloodZoneCode"])

	var value string
	switch provider {
	case MirrorProviderSpark:
		value = firstNonEmpty(spark, generic)
	default:
		value = firstNonEmpty(bridgeExt, generic, spark)
	}
	if value == "" {
		return nil
	}
	return &value
}

func (r *ResoFieldResolver) ResolveEstimatedTotalMonthlyFees(row map[string]any, provider MirrorProvider, datasetUpper string) *float64 {
	if provider == MirrorProviderBridge {
		extField := strings.ToUpper(datasetUpper) + "_TotalMonthlyFees"
		if v, ok := numericValue(row[extField]); ok && v > 0 {
			f := round2(v)
			return &f
		}
	}
	allowFallback := provider == MirrorProviderSpark
	return r.fees.FromResoRow(row, allowFallback)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
