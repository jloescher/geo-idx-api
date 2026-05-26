package gis

import "github.com/quantyralabs/idx-api/internal/config"

const (
	SyncModeBBox        = "bbox"
	SyncModePaginate    = "paginate"
	SyncModeWhereFilter = "where_filter"
	SyncModeShapefile   = "shapefile"

	FormatGeoJSON = "geojson"
	FormatJSON    = "json"
)

// ParcelSourceSpec describes one parcel ArcGIS sync/proxy source.
type ParcelSourceSpec struct {
	SourceKey      string
	CountySlug     string
	QueryURL       string
	SyncMode       string
	ArcGISWhere    string
	SyncBBox       BBox
	ResponseFormat string
	MLSFeed        string // stellar | beaches
	Enabled        bool
	Priority       int
	HTTPTimeoutSec int
	ParcelIDFields []string
	Notes          string
}

// ParcelSourceCatalog returns sync sources: statewide primary + Pinellas/Hillsborough failovers.
// Osceola (CO_NO=59) is covered by the statewide layer; no county-specific Osceola source.
func ParcelSourceCatalog() []ParcelSourceSpec {
	return []ParcelSourceSpec{
		{
			SourceKey:      FloridaStatewideCadastralKey,
			CountySlug:     "statewide",
			QueryURL:       FloridaStatewideCadastralURL,
			SyncMode:       SyncModePaginate,
			ResponseFormat: FormatGeoJSON,
			MLSFeed:        "stellar",
			Enabled:        true,
			Priority:       1,
			ParcelIDFields: []string{"PARCEL_ID_", "PARCELNO", "ALT_KEY", "PARCELID", "PARCEL_ID", "STATE_PAR_"},
			Notes:          "FDOR Cadastral 2025; paginate where=1=1; CO_NO filter in Go",
		},
		{
			SourceKey:      "pinellas_enterprise_parcels",
			CountySlug:     "pinellas",
			QueryURL:       PinellasParcelsQueryURL,
			SyncMode:       SyncModeBBox,
			SyncBBox:       countyBBox("pinellas"),
			ResponseFormat: FormatGeoJSON,
			MLSFeed:        "stellar",
			Enabled:        true,
			Priority:       2,
			HTTPTimeoutSec: 120,
			ParcelIDFields: []string{"PARCEL_ID", "PARCELID", "PIN"},
		},
		{
			SourceKey:      "hillsborough_hc_parcels",
			CountySlug:     "hillsborough",
			QueryURL:       HillsboroughParcelsQueryURL,
			SyncMode:       SyncModeBBox,
			SyncBBox:       countyBBox("hillsborough"),
			ResponseFormat: FormatGeoJSON,
			MLSFeed:        "stellar",
			Enabled:        true,
			Priority:       3,
			ParcelIDFields: []string{"PARCEL_ID", "PARCELID", "PIN", "FOLIO"},
		},
	}
}

// EnabledParcelSourcesForConfig returns catalog entries enabled for background sync.
// Pinellas enterprise is opt-in (GIS_SYNC_PINELLAS_ENTERPRISE) because the county host often
// times out from datacenter networks; live API fallback still uses FailoverSourcesForBBox.
func EnabledParcelSourcesForConfig(cfg config.GISConfig) []ParcelSourceSpec {
	var out []ParcelSourceSpec
	for _, s := range ParcelSourceCatalog() {
		if !s.Enabled {
			continue
		}
		if s.SourceKey == "pinellas_enterprise_parcels" && !cfg.SyncPinellasEnterprise {
			continue
		}
		out = append(out, s)
	}
	sortParcelSourcesByPriority(out)
	return out
}

// ParcelSourceByCounty returns the enabled failover source for a county slug, if any.
func ParcelSourceByCounty(slug string) (ParcelSourceSpec, bool) {
	for _, s := range EnabledParcelSources() {
		if s.CountySlug == slug {
			return s, true
		}
	}
	return ParcelSourceSpec{}, false
}

// EnabledParcelSources returns catalog entries enabled for sync/proxy.
func EnabledParcelSources() []ParcelSourceSpec {
	return EnabledParcelSourcesForConfig(config.GISConfig{})
}

// FailoverSourcesForBBox returns Pinellas/Hillsborough sources intersecting the bbox (live proxy).
func FailoverSourcesForBBox(b BBox) []ParcelSourceSpec {
	var out []ParcelSourceSpec
	for _, s := range ParcelSourceCatalog() {
		if s.CountySlug == "" || s.CountySlug == "statewide" || !s.Enabled {
			continue
		}
		if intersects(b, s.SyncBBox) {
			out = append(out, s)
		}
	}
	sortParcelSourcesByPriority(out)
	return out
}

// ParcelSourceSpecByKey returns a catalog entry by source_key.
func ParcelSourceSpecByKey(sourceKey string) (ParcelSourceSpec, bool) {
	for _, s := range ParcelSourceCatalog() {
		if s.SourceKey == sourceKey {
			return s, true
		}
	}
	return ParcelSourceSpec{}, false
}

// SourcesForBBox is an alias for FailoverSourcesForBBox (live proxy + county hints).
func SourcesForBBox(b BBox) []ParcelSourceSpec {
	return FailoverSourcesForBBox(b)
}

func sortParcelSourcesByPriority(sources []ParcelSourceSpec) {
	for i := 0; i < len(sources); i++ {
		for j := i + 1; j < len(sources); j++ {
			if sources[j].Priority < sources[i].Priority {
				sources[i], sources[j] = sources[j], sources[i]
			}
		}
	}
}

func countyBBox(slug string) BBox {
	if b, ok := countySyncBBoxes[slug]; ok {
		return b
	}
	return BBox{West: -87.6, South: 24.4, East: -79.8, North: 31.1}
}

// countySyncBBoxes are WGS84 envelopes for failover county parcel sync jobs.
var countySyncBBoxes = map[string]BBox{
	"alachua":      {West: -82.65, South: 29.35, East: -81.95, North: 29.95},
	"charlotte":    {West: -82.35, South: 26.75, East: -81.75, North: 27.15},
	"desoto":       {West: -82.05, South: 27.05, East: -81.65, North: 27.55},
	"flagler":      {West: -81.55, South: 29.25, East: -81.05, North: 29.65},
	"hillsborough": {West: -82.85, South: 27.65, East: -82.05, North: 28.20},
	"lake":         {West: -82.05, South: 28.55, East: -81.35, North: 29.05},
	"manatee":      {West: -82.85, South: 27.25, East: -82.15, North: 27.65},
	"marion":       {West: -82.45, South: 28.85, East: -81.75, North: 29.45},
	"okeechobee":   {West: -81.15, South: 27.15, East: -80.55, North: 27.75},
	"orange":       {West: -81.75, South: 28.25, East: -80.95, North: 28.85},
	"osceola":      {West: -81.55, South: 27.85, East: -80.95, North: 28.35},
	"pasco":        {West: -82.85, South: 28.05, East: -82.15, North: 28.55},
	"pinellas":     {West: -82.98, South: 27.60, East: -82.55, North: 28.25},
	"polk":         {West: -82.15, South: 27.65, East: -81.35, North: 28.35},
	"sarasota":     {West: -82.85, South: 26.95, East: -82.05, North: 27.45},
	"sumter":       {West: -82.35, South: 28.45, East: -81.85, North: 28.95},
	"volusia":      {West: -81.55, South: 28.75, East: -80.85, North: 29.45},
	"broward":      {West: -80.45, South: 25.95, East: -79.95, North: 26.45},
	"palm-beach":   {West: -80.35, South: 26.35, East: -79.85, North: 26.95},
	"martin":       {West: -80.65, South: 26.85, East: -80.05, North: 27.35},
	"st-lucie":     {West: -80.75, South: 27.05, East: -80.15, North: 27.65},
	"miami-dade":   {West: -80.85, South: 25.35, East: -80.05, North: 25.95},
}
