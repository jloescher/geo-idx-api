package gis

import "github.com/quantyralabs/idx-api/internal/config"

const (
	SyncModeBBox        = "bbox"
	SyncModePaginate    = "paginate"
	SyncModeWhereFilter = "where_filter"
	SyncModeShapefile   = "shapefile"

	FormatGeoJSON = "geojson"
	FormatJSON    = "json"

	sfwmdNormalizedParcelsURL = "https://geoweb.sfwmd.gov/agsext2/rest/services/LandOwnershipAndInterests/NormalizedParcels/FeatureServer/0/query"
)

// ParcelSourceSpec describes one county parcel ArcGIS sync/proxy source.
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

// ParcelSourceCatalog returns all MLS county parcel sources (22 enabled + Osceola stub).
func ParcelSourceCatalog() []ParcelSourceSpec {
	return []ParcelSourceSpec{
		{SourceKey: "alachua_ehwater_parcels", CountySlug: "alachua", QueryURL: "https://gis.floridahealth.gov/server/rest/services/EHWATER/Parcels/MapServer/0/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("alachua"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"PARCEL_ID", "PARCELID", "PIN", "OBJECTID"}},
		{SourceKey: "charlotte_ccgis_parcels", CountySlug: "charlotte", QueryURL: "https://agis3.charlottecountyfl.gov/arcgis/rest/services/Essentials/CCGISLayers/MapServer/27/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("charlotte"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"ACCOUNT", "PARCEL_ID", "PARCELID"}},
		{SourceKey: "desoto_swfwmd_parcels", CountySlug: "desoto", QueryURL: "https://www45.swfwmd.state.fl.us/arcgis12/rest/services/BaseVector/parcel_search/MapServer/3/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("desoto"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"PARCEL_ID", "PARCELID", "PARCELNO"}},
		{SourceKey: "flagler_palmcoast_parcels", CountySlug: "flagler", QueryURL: "https://gis.palmcoast.gov/hosting/rest/services/External/FlaglerCountyParcels/MapServer/1/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("flagler"), ResponseFormat: FormatJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"PARCELNO", "PARCEL_ID", "PARCELID"}},
		{SourceKey: "hillsborough_hc_parcels", CountySlug: "hillsborough", QueryURL: HillsboroughParcelsQueryURL, SyncMode: SyncModeBBox, SyncBBox: countyBBox("hillsborough"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 5, ParcelIDFields: []string{"PARCEL_ID", "PARCELID", "PIN", "FOLIO"}},
		{SourceKey: "lake_opendata_parcels", CountySlug: "lake", QueryURL: "https://gis.lakecountyfl.gov/lakegis/rest/services/OpenData/OpenData1/FeatureServer/12/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("lake"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"AltKey", "PARCEL_ID", "PARCELID"}},
		{SourceKey: "manatee_parcellines", CountySlug: "manatee", QueryURL: "https://www.mymanatee.org/gisits/rest/services/commonoperational/parcellines/MapServer/0/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("manatee"), ResponseFormat: FormatJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"PARCEL_ID", "PARCELID", "PIN"}},
		{SourceKey: "marion_gis_parcels", CountySlug: "marion", QueryURL: "https://gis.marionfl.org/public/rest/services/General/Parcels/MapServer/0/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("marion"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"PARCEL_ID", "PARCELID", "PIN"}},
		{SourceKey: "okeechobee_sfwmd_parcels", CountySlug: "okeechobee", QueryURL: sfwmdNormalizedParcelsURL, SyncMode: SyncModeWhereFilter, ArcGISWhere: "CNTYNAME='Okeechobee'", SyncBBox: countyBBox("okeechobee"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"PARCEL_ID", "PARCELID", "PARCELNO"}},
		{SourceKey: "orange_ocpa_parcels", CountySlug: "orange", QueryURL: "https://services2.arcgis.com/N4cKzJ9dzXmsPNRs/ArcGIS/rest/services/orange_county_parcels/FeatureServer/0/query", SyncMode: SyncModePaginate, SyncBBox: countyBBox("orange"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"PARCELID", "PARCEL_ID", "PID"}},
		{SourceKey: "osceola_pa_parcels", CountySlug: "osceola", QueryURL: "https://services2.arcgis.com/V2PQwgZMTFfgM0Xu/arcgis/rest/services/Osceola_County_FL_WFL1/FeatureServer/0/query", SyncMode: SyncModeShapefile, SyncBBox: countyBBox("osceola"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: false, Priority: 99, Notes: "REST layer incomplete; enable after shapefile ingest"},
		{SourceKey: "pasco_parcels", CountySlug: "pasco", QueryURL: "https://maps.pascopa.com/arcgis/rest/services/Parcels/MapServer/3/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("pasco"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"ParcelID", "PARCEL_ID", "PARCELID"}},
		{SourceKey: "pinellas_swfwmd_parcels", CountySlug: "pinellas", QueryURL: "https://www45.swfwmd.state.fl.us/arcgis12/rest/services/BaseVector/parcel_search/MapServer/13/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("pinellas"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 8, HTTPTimeoutSec: 120, ParcelIDFields: []string{"PARCEL_ID", "PARCELID", "PIN"}},
		{SourceKey: "polk_tpo_parcels", CountySlug: "polk", QueryURL: "https://gis.polk-county.net/hosting/rest/services/TPO/TPO_Parcel_and_Permit_Map/MapServer/1/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("polk"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"PARCELID", "PARCEL_ID"}},
		{SourceKey: "sarasota_scpa_parcels", CountySlug: "sarasota", QueryURL: "https://services3.arcgis.com/icrWMv7eBkctFu1f/arcgis/rest/services/ParcelHosted/FeatureServer/0/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("sarasota"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"PARCEL_ID", "PARCELID", "PIN"}},
		{SourceKey: "sumter_ecfrpc_parcels", CountySlug: "sumter", QueryURL: "https://gis.ecfrpc.org/arcgis/rest/services/Basemap/MapServer/4/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("sumter"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"PARCELNO", "PARCEL_ID", "PARCELID"}},
		{SourceKey: "volusia_open_data_parcels", CountySlug: "volusia", QueryURL: "https://maps5.vcgov.org/arcgis/rest/services/Open_Data/Open_Data_3/FeatureServer/36/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("volusia"), ResponseFormat: FormatGeoJSON, MLSFeed: "stellar", Enabled: true, Priority: 10, ParcelIDFields: []string{"PID", "PARCEL_ID", "PARCELID"}},
		{SourceKey: "broward_parcel_boundary", CountySlug: "broward", QueryURL: "https://services5.arcgis.com/wI5GZmCtnUU8ueya/ArcGIS/rest/services/Broward_County_Parcel_Boundary/FeatureServer/1/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("broward"), ResponseFormat: FormatGeoJSON, MLSFeed: "beaches", Enabled: true, Priority: 10, HTTPTimeoutSec: 120, ParcelIDFields: []string{"PARCELNO", "PARCEL_ID", "PARCELID"}},
		{SourceKey: "palm_beach_opendata_parcels", CountySlug: "palm-beach", QueryURL: "https://maps.co.palm-beach.fl.us/arcgis/rest/services/OpenData/open_data_v2/MapServer/0/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("palm-beach"), ResponseFormat: FormatGeoJSON, MLSFeed: "beaches", Enabled: true, Priority: 10, ParcelIDFields: []string{"PCN", "PARCEL_ID", "PARCELID"}},
		{SourceKey: "martin_sfwmd_parcels", CountySlug: "martin", QueryURL: sfwmdNormalizedParcelsURL, SyncMode: SyncModeWhereFilter, ArcGISWhere: "CNTYNAME='Martin'", SyncBBox: countyBBox("martin"), ResponseFormat: FormatGeoJSON, MLSFeed: "beaches", Enabled: true, Priority: 10, ParcelIDFields: []string{"PARCEL_ID", "PARCELID", "PARCELNO"}},
		{SourceKey: "st_lucie_sfwmd_parcels", CountySlug: "st-lucie", QueryURL: sfwmdNormalizedParcelsURL, SyncMode: SyncModeWhereFilter, ArcGISWhere: "CNTYNAME='St Lucie'", SyncBBox: countyBBox("st-lucie"), ResponseFormat: FormatGeoJSON, MLSFeed: "beaches", Enabled: true, Priority: 10, ParcelIDFields: []string{"PARCEL_ID", "PARCELID", "PARCELNO"}},
		{SourceKey: "miami_dade_pa_parcels", CountySlug: "miami-dade", QueryURL: "https://gisweb.miamidade.gov/arcgis/rest/services/MD_LandInformation/MapServer/26/query", SyncMode: SyncModeBBox, SyncBBox: countyBBox("miami-dade"), ResponseFormat: FormatJSON, MLSFeed: "beaches", Enabled: true, Priority: 10, ParcelIDFields: []string{"PARCELNO", "PARCEL_ID", "PARCELID", "FOLIO"}},
	}
}

// EnabledParcelSourcesForConfig returns enabled catalog entries plus optional overrides.
func EnabledParcelSourcesForConfig(cfg config.GISConfig) []ParcelSourceSpec {
	var out []ParcelSourceSpec
	for _, s := range ParcelSourceCatalog() {
		if s.Enabled {
			out = append(out, s)
		}
	}
	if cfg.SyncPinellasEnterprise {
		out = append(out, ParcelSourceSpec{
			SourceKey:      "pinellas_enterprise_parcels",
			CountySlug:     "pinellas",
			QueryURL:       PinellasParcelsQueryURL,
			SyncMode:       SyncModeBBox,
			SyncBBox:       countyBBox("pinellas"),
			ResponseFormat: FormatGeoJSON,
			MLSFeed:        "stellar",
			Enabled:        true,
			Priority:       3,
			ParcelIDFields: []string{"PARCEL_ID", "PARCELID", "PIN"},
		})
	}
	sortParcelSourcesByPriority(out)
	return out
}

// ParcelSourceByCounty returns the enabled source for a county slug, if any.
func ParcelSourceByCounty(slug string) (ParcelSourceSpec, bool) {
	for _, s := range EnabledParcelSources() {
		if s.CountySlug == slug {
			return s, true
		}
	}
	return ParcelSourceSpec{}, false
}

// EnabledParcelSources returns catalog entries that are enabled for sync/proxy (default config).
func EnabledParcelSources() []ParcelSourceSpec {
	return EnabledParcelSourcesForConfig(config.GISConfig{})
}

// SourcesForBBox returns enabled parcel sources whose sync bbox intersects the query bbox, sorted by priority.
func SourcesForBBox(b BBox) []ParcelSourceSpec {
	var out []ParcelSourceSpec
	for _, s := range EnabledParcelSources() {
		if intersects(b, s.SyncBBox) {
			out = append(out, s)
		}
	}
	sortParcelSourcesByPriority(out)
	return out
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

// countySyncBBoxes are WGS84 envelopes for full-county parcel sync jobs.
var countySyncBBoxes = map[string]BBox{
	"alachua":    {West: -82.65, South: 29.35, East: -81.95, North: 29.95},
	"charlotte":  {West: -82.35, South: 26.75, East: -81.75, North: 27.15},
	"desoto":     {West: -82.05, South: 27.05, East: -81.65, North: 27.55},
	"flagler":    {West: -81.55, South: 29.25, East: -81.05, North: 29.65},
	"hillsborough": {West: -82.85, South: 27.65, East: -82.05, North: 28.20},
	"lake":       {West: -82.05, South: 28.55, East: -81.35, North: 29.05},
	"manatee":    {West: -82.85, South: 27.25, East: -82.15, North: 27.65},
	"marion":     {West: -82.45, South: 28.85, East: -81.75, North: 29.45},
	"okeechobee": {West: -81.15, South: 27.15, East: -80.55, North: 27.75},
	"orange":     {West: -81.75, South: 28.25, East: -80.95, North: 28.85},
	"osceola":    {West: -81.55, South: 27.85, East: -80.95, North: 28.35},
	"pasco":      {West: -82.85, South: 28.05, East: -82.15, North: 28.55},
	"pinellas":   {West: -82.98, South: 27.60, East: -82.55, North: 28.25},
	"polk":       {West: -82.15, South: 27.65, East: -81.35, North: 28.35},
	"sarasota":   {West: -82.85, South: 26.95, East: -82.05, North: 27.45},
	"sumter":     {West: -82.35, South: 28.45, East: -81.85, North: 28.95},
	"volusia":    {West: -81.55, South: 28.75, East: -80.85, North: 29.45},
	"broward":    {West: -80.45, South: 25.95, East: -79.95, North: 26.45},
	"palm-beach": {West: -80.35, South: 26.35, East: -79.85, North: 26.95},
	"martin":     {West: -80.65, South: 26.85, East: -80.05, North: 27.35},
	"st-lucie":   {West: -80.75, South: 27.05, East: -80.15, North: 27.65},
	"miami-dade": {West: -80.85, South: 25.35, East: -80.05, North: 25.95},
}
