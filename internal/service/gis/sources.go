package gis

// Persistent statewide parcel sync (including Osceola) = instant complete Leaflet maps for all
// pilot counties → maximum engagement before hard gate → more OTP leads.

// ArcGIS parcel layer endpoints for county-specific mirrors.
const (
	PinellasParcelsQueryURL     = "https://egis.pinellascounty.org/arcgis/rest/services/PARCEL/MapServer/0/query"
	HillsboroughParcelsQueryURL = "https://maps.hillsboroughcounty.org/arcgis/rest/services/InfoLayers/HC_ParcelsPublic/FeatureServer/0/query"

	PinellasParcelsMetaURL     = "https://egis.pinellascounty.org/arcgis/rest/services/PARCEL/MapServer/0?f=json"
	HillsboroughParcelsMetaURL = "https://maps.hillsboroughcounty.org/arcgis/rest/services/InfoLayers/HC_ParcelsPublic/FeatureServer/0?f=json"
)

// Source describes an ArcGIS layer endpoint for live proxy fallback.
type Source struct {
	Key      string
	QueryURL string
	Tier     string
	Where    string
	CountyCO string
}

func sourcesForBBox(b BBox) []Source {
	var out []Source
	for _, spec := range FailoverSourcesForBBox(b) {
		out = append(out, Source{
			Key:      spec.SourceKey,
			QueryURL: spec.QueryURL,
			Tier:     spec.CountySlug,
			Where:    spec.ArcGISWhere,
		})
	}
	return out
}

func countyHint(lat, lng float64) string {
	for slug, bbox := range countySyncBBoxes {
		if lat >= bbox.South && lat <= bbox.North && lng >= bbox.West && lng <= bbox.East {
			return slug
		}
	}
	return ""
}

func intersects(a, b BBox) bool {
	return !(a.East < b.West || a.West > b.East || a.North < b.South || a.South > b.North)
}
