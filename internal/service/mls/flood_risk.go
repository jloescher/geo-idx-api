package mls

import "strings"

// ComputeLowRiskFloodZoneYN derives low_risk_flood_zone_yn from a normalized flood_zone_code.
//
// false when code is empty; false when code contains A or V (case-insensitive substring);
// true when code contains X500, "no", or X (case-insensitive); else false.
func ComputeLowRiskFloodZoneYN(floodZoneCode *string) bool {
	if floodZoneCode == nil {
		return false
	}
	code := strings.TrimSpace(*floodZoneCode)
	if code == "" {
		return false
	}
	upper := strings.ToUpper(code)
	if strings.Contains(upper, "A") || strings.Contains(upper, "V") {
		return false
	}
	if strings.Contains(upper, "X500") {
		return true
	}
	if strings.Contains(strings.ToLower(code), "no") {
		return true
	}
	if strings.Contains(upper, "X") {
		return true
	}
	return false
}
