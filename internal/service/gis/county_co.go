package gis

// Persistent statewide parcel sync (including Osceola) = instant complete Leaflet maps for all
// pilot counties → maximum engagement before hard gate → more OTP leads.
//
// FDOR CO_NO values for the 22 MLS pilot counties (Florida Department of Revenue county codes).

const (
	FloridaStatewideCadastralKey = "florida_statewide_cadastral"
)

// FloridaStatewideCadastralURL is FDOR Cadastral 2025 layer 0 (paginate-only sync; no WHERE/geometry).
const FloridaStatewideCadastralURL = "https://services9.arcgis.com/Gh9awoU677aKree0/arcgis/rest/services/Florida_Statewide_Cadastral/FeatureServer/0/query"

// coNoToCountySlug maps FDOR CO_NO to gis_parcels.county slugs for MLS pilot counties.
var coNoToCountySlug = map[int]string{
	1:  "alachua",
	6:  "broward",
	8:  "charlotte",
	14: "desoto",
	18: "flagler",
	23: "miami-dade",
	28: "hillsborough",
	34: "lake",
	40: "manatee",
	41: "marion",
	42: "martin",
	47: "okeechobee",
	48: "orange",
	50: "palm-beach",
	51: "pasco",
	52: "pinellas",
	53: "polk",
	56: "st-lucie",
	57: "sarasota",
	59: "osceola",
	60: "sumter",
	64: "volusia",
}

// CountySlugFromCONO maps an FDOR CO_NO attribute to a county slug.
func CountySlugFromCONO(coNo int) (string, bool) {
	slug, ok := coNoToCountySlug[coNo]
	return slug, ok
}

// CONOFromProperties reads CO_NO from ArcGIS feature properties.
func CONOFromProperties(props map[string]any) (int, bool) {
	v := firstFloat(props, "CO_NO")
	if v == 0 {
		return 0, false
	}
	return int(v), true
}

// IsMLSPilotCounty reports whether slug is in Stellar or Beaches MLS coverage.
func IsMLSPilotCounty(slug string) bool {
	return MLSStellarSlugs[slug] || MLSBeachesSlugs[slug]
}

// IsStatewideCadastralSource reports whether source_key is the FDOR statewide layer.
func IsStatewideCadastralSource(sourceKey string) bool {
	return sourceKey == FloridaStatewideCadastralKey
}
