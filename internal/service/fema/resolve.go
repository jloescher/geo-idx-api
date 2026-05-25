package fema

import "time"

// EffectiveFloodZoneCode returns the best display label: FEMA when enriched, else MLS.
// Not used for low_risk_flood_zone_yn (that column is set from fema_flood_zone_code only).
func EffectiveFloodZoneCode(mls *string, fema *string, femaAt *time.Time) *string {
	if femaAt != nil && fema != nil {
		s := *fema
		if s != "" {
			return fema
		}
	}
	return mls
}
