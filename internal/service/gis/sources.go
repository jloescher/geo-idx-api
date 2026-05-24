package gis

// ArcGIS parcel layer endpoints (FDOR 2025 statewide + county mirrors).
const (
	FloridaStatewideCadastralQueryURL = "https://services9.arcgis.com/Gh9awoU677aKree0/ArcGIS/rest/services/Florida_Statewide_Cadastral/FeatureServer/0/query"
	PinellasParcelsQueryURL           = "https://egis.pinellascounty.org/arcgis/rest/services/PARCEL/MapServer/0/query"
	HillsboroughParcelsQueryURL       = "https://maps.hillsboroughcounty.org/arcgis/rest/services/InfoLayers/HC_ParcelsPublic/FeatureServer/0/query"

	FloridaStatewideCadastralMetaURL = "https://services9.arcgis.com/Gh9awoU677aKree0/ArcGIS/rest/services/Florida_Statewide_Cadastral/FeatureServer/0?f=json"
	PinellasParcelsMetaURL           = "https://egis.pinellascounty.org/arcgis/rest/services/PARCEL/MapServer/0?f=json"
	HillsboroughParcelsMetaURL       = "https://maps.hillsboroughcounty.org/arcgis/rest/services/InfoLayers/HC_ParcelsPublic/FeatureServer/0?f=json"
)

// Source describes an ArcGIS layer endpoint.
type Source struct {
	Key      string
	QueryURL string
	Tier     string
	CountyCO string // statewide CO_NO filter when using primary with county hint
}

var pinellasBBox = BBox{West: -82.9, South: 27.6, East: -82.6, North: 28.2}
var hillsboroughBBox = BBox{West: -82.7, South: 27.7, East: -82.2, North: 28.2}

func sourcesForBBox(b BBox) []Source {
	lat, lng := b.Centroid()
	hint := countyHint(lat, lng)
	out := []Source{{
		Key:      "florida_statewide_cadastral",
		QueryURL: FloridaStatewideCadastralQueryURL,
		Tier:     "statewide",
		CountyCO: coNoForCounty(hint),
	}}
	if intersects(b, pinellasBBox) {
		out = append(out, Source{
			Key:      "pinellas_enterprise_parcels",
			QueryURL: PinellasParcelsQueryURL,
			Tier:     "pinellas",
		})
	}
	if intersects(b, hillsboroughBBox) {
		out = append(out, Source{
			Key:      "hillsborough_hc_parcels",
			QueryURL: HillsboroughParcelsQueryURL,
			Tier:     "hillsborough",
		})
	}
	return out
}

// syncBBoxForCounty returns the WGS84 envelope used for full-layer parcel sync jobs.
func syncBBoxForCounty(county string) BBox {
	switch county {
	case "pinellas":
		return pinellasBBox
	case "hillsborough":
		return hillsboroughBBox
	default:
		return BBox{West: -87.6, South: 24.4, East: -79.8, North: 31.1}
	}
}

func countyHint(lat, lng float64) string {
	if lat >= pinellasBBox.South && lat <= pinellasBBox.North && lng >= pinellasBBox.West && lng <= pinellasBBox.East {
		return "pinellas"
	}
	if lat >= hillsboroughBBox.South && lat <= hillsboroughBBox.North && lng >= hillsboroughBBox.West && lng <= hillsboroughBBox.East {
		return "hillsborough"
	}
	return ""
}

func coNoForCounty(hint string) string {
	switch hint {
	case "pinellas":
		return "52"
	case "hillsborough":
		return "29"
	default:
		return ""
	}
}

func intersects(a, b BBox) bool {
	return !(a.East < b.West || a.West > b.East || a.North < b.South || a.South > b.North)
}
