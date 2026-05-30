package fema

import "time"

// EffectiveFloodZoneCode returns the best display label: FEMA when enriched, else MLS.
// Used by API flood_zone.effective_code assembly in internal/service/mls/flood_zone_response.go.
func EffectiveFloodZoneCode(mls *string, fema *string, femaAt *time.Time) *string {
	if femaAt != nil && fema != nil {
		s := *fema
		if s != "" {
			return fema
		}
	}
	return mls
}
