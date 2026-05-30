package comps

import (
	"encoding/json"
	"math"
	"strings"
)

func parseProperty(raw json.RawMessage) CompRecord {
	var m map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &m) != nil {
		return CompRecord{Property: raw}
	}
	c := CompRecord{Property: raw}
	c.ListingKey = str(m["ListingKey"])
	c.StandardStatus = str(m["StandardStatus"])
	c.ClosePrice = num(m["ClosePrice"])
	c.ListPrice = num(m["ListPrice"])
	if c.ListPrice == 0 {
		c.ListPrice = num(m["ListPrice"])
	}
	c.Bedrooms = num(m["BedroomsTotal"])
	c.Bathrooms = num(m["BathroomsTotalDecimal"])
	if c.Bathrooms == 0 {
		c.Bathrooms = num(m["BathroomsTotal"])
	}
	c.LivingArea = num(m["LivingArea"])
	c.LotSizeAcres = num(m["LotSizeAcres"])
	c.YearBuilt = int(num(m["YearBuilt"]))
	c.GarageSpaces = num(m["GarageSpaces"])
	c.PoolPrivate = boolVal(m["PoolPrivateYN"])
	c.Waterfront = boolVal(m["WaterfrontYN"])
	c.CloseDate = str(m["CloseDate"])
	c.MonthlyFees = num(m["STELLAR_TotalMonthlyFees"])
	if c.MonthlyFees == 0 {
		c.MonthlyFees = num(m["TotalMonthlyFees"])
	}
	c.FloodZone = floodZoneFromProperty(m)
	if c.FloodZone == "" {
		c.FloodZone = str(m["STELLAR_FloodZoneCode"])
	}
	if c.FloodZone == "" {
		c.FloodZone = str(m["FloodZoneCode"])
	}
	lat, lng := coordsFromMap(m)
	c.Lat, c.Lng = lat, lng
	return c
}

func coordsFromMap(m map[string]any) (lat, lng float64) {
	if v, ok := m["Latitude"].(float64); ok {
		lat = v
	}
	if v, ok := m["Longitude"].(float64); ok {
		lng = v
	}
	if lat != 0 && lng != 0 {
		return lat, lng
	}
	if c, ok := m["Coordinates"].(map[string]any); ok {
		if coords, ok := c["coordinates"].([]any); ok && len(coords) >= 2 {
			if lngV, ok := coords[0].(float64); ok {
				lng = lngV
			}
			if latV, ok := coords[1].(float64); ok {
				lat = latV
			}
		}
	}
	return lat, lng
}

func floodZoneFromProperty(m map[string]any) string {
	fz, ok := m["flood_zone"].(map[string]any)
	if !ok {
		return ""
	}
	return str(fz["effective_code"])
}

func str(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		b, _ := json.Marshal(t)
		return strings.Trim(strings.TrimSpace(string(b)), `"`)
	}
}

func num(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
	}
}

func boolVal(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(t, "true") || t == "1" || strings.EqualFold(t, "yes")
	default:
		return false
	}
}

func haversineMiles(lat1, lng1, lat2, lng2 float64) float64 {
	const earth = 3958.7613
	rad := math.Pi / 180
	la1, lo1 := lat1*rad, lng1*rad
	la2, lo2 := lat2*rad, lng2*rad
	dla := la2 - la1
	dlo := lo2 - lo1
	a := math.Sin(dla/2)*math.Sin(dla/2) + math.Cos(la1)*math.Cos(la2)*math.Sin(dlo/2)*math.Sin(dlo/2)
	return earth * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
