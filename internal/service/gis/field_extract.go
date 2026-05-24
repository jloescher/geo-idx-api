package gis

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
)

// ExtractParcelRow maps an ArcGIS feature to a gis_parcels row.
// Revenue impact: structured parcel fields power richer map popups that drive lead capture.
func ExtractParcelRow(feat ArcGISFeature, sourceKey, county string, gen int, fingerprint *string) (gisrepo.ParcelRow, error) {
	geom := string(feat.Geometry)
	if len(geom) == 0 || string(geom) == "null" {
		return gisrepo.ParcelRow{}, errNoGeometry
	}
	propsJSON, _ := json.Marshal(feat.Properties)
	parcelID := firstString(feat.Properties,
		"PARCELID", "PARCEL_ID", "PARCELNO", "PARCEL_NO", "PC_PID", "PIN", "FOLIO", "OBJECTID")
	if parcelID == "" {
		parcelID = firstString(feat.Properties, "GlobalID", "FID")
	}
	if parcelID == "" {
		return gisrepo.ParcelRow{}, errNoParcelID
	}
	row := gisrepo.ParcelRow{
		ParcelID:          parcelID,
		SourceKey:         sourceKey,
		County:            county,
		GeometryJSON:      geom,
		Properties:        propsJSON,
		SiteAddress:       strPtr(firstString(feat.Properties, "PHY_ADDR1", "SITE_ADDR", "SITUS_ADDR", "ADDRESS", "LOC_ADDR")),
		OwnerName:         strPtr(firstString(feat.Properties, "OWN_NAME", "OWNER", "OWNER_NAME", "OWNERNME1")),
		City:              strPtr(firstString(feat.Properties, "PHY_CITY", "SITE_CITY", "CITY", "MUNICIPAL")),
		ZipCode:           strPtr(firstString(feat.Properties, "PHY_ZIPCD", "ZIP", "ZIP_CODE", "POSTAL")),
		JustValue:         floatPtr(firstFloat(feat.Properties, "JV", "JUST", "JUST_VALUE", "TOT_VAL")),
		AssessedValue:     floatPtr(firstFloat(feat.Properties, "AV_SD", "ASSESSED", "ASSESSED_VALUE", "TAXABLE")),
		LandValue:         floatPtr(firstFloat(feat.Properties, "LV", "LAND", "LAND_VALUE")),
		LivingAreaSqft:    floatPtr(firstFloat(feat.Properties, "TOT_LVG_AR", "LIVING_AREA", "BLDG_AREA", "HEATED_AREA")),
		YearBuilt:         intPtr(firstInt(feat.Properties, "ACT_YR_BLT", "YEAR_BUILT", "YR_BLT")),
		Acres:             floatPtr(firstFloat(feat.Properties, "ACRES", "GIS_ACRES", "CALC_ACRES")),
		LandUseCode:       strPtr(firstString(feat.Properties, "DOR_UC", "LAND_USE", "LU_CODE", "USE_CODE")),
		LastSalePrice:     floatPtr(firstFloat(feat.Properties, "SALE_PRC1", "SALE_PRICE", "LAST_SALE_PRICE")),
		SourceGeneration:  gen,
		SourceFingerprint: fingerprint,
	}
	if saleDate := firstString(feat.Properties, "SALE_DATE1", "SALE_DATE", "LAST_SALE_DATE"); saleDate != "" {
		if t, err := parseArcGISDate(saleDate); err == nil {
			row.LastSaleDate = &t
		}
	}
	return row, nil
}

// ExtractCityRow maps an ArcGIS feature to a gis_cities row.
func ExtractCityRow(feat ArcGISFeature, gen int, fingerprint *string) (gisrepo.CityRow, error) {
	geom := string(feat.Geometry)
	if len(geom) == 0 || string(geom) == "null" {
		return gisrepo.CityRow{}, errNoGeometry
	}
	name := firstString(feat.Properties, "NAME", "CITYNAME", "CITY_NAME", "MUNICIPAL")
	if name == "" {
		return gisrepo.CityRow{}, errNoName
	}
	propsJSON, _ := json.Marshal(feat.Properties)
	county := strPtr(firstString(feat.Properties, "COUNTY", "COUNTYNAME", "COUNTY_NAME"))
	return gisrepo.CityRow{
		CityName:          name,
		County:            county,
		SourceKey:         FDOTAdminBoundariesKey,
		GeometryJSON:      geom,
		Properties:        propsJSON,
		SourceGeneration:  gen,
		SourceFingerprint: fingerprint,
	}, nil
}

// ExtractCountyRow maps an ArcGIS feature to a gis_counties row.
func ExtractCountyRow(feat ArcGISFeature, gen int, fingerprint *string) (gisrepo.CountyRow, error) {
	geom := string(feat.Geometry)
	if len(geom) == 0 || string(geom) == "null" {
		return gisrepo.CountyRow{}, errNoGeometry
	}
	name := firstString(feat.Properties, "NAME", "COUNTY", "COUNTYNAME")
	fips := firstString(feat.Properties, "FIPS", "FIPS_CODE", "COUNTYFIPS", "GEOID")
	if name == "" && fips == "" {
		return gisrepo.CountyRow{}, errNoName
	}
	if name == "" {
		name = fips
	}
	propsJSON, _ := json.Marshal(feat.Properties)
	return gisrepo.CountyRow{
		CountyName:        name,
		FIPSCode:          strPtr(fips),
		SourceKey:         FDOTAdminBoundariesKey,
		GeometryJSON:      geom,
		Properties:        propsJSON,
		SourceGeneration:  gen,
		SourceFingerprint: fingerprint,
	}, nil
}

// ExtractZipRow maps an ArcGIS feature to a gis_zips row.
func ExtractZipRow(feat ArcGISFeature, gen int, fingerprint *string) (gisrepo.ZipRow, error) {
	geom := string(feat.Geometry)
	if len(geom) == 0 || string(geom) == "null" {
		return gisrepo.ZipRow{}, errNoGeometry
	}
	zip := firstString(feat.Properties, "ZIP", "ZIP_CODE", "ZCTA5CE10", "ZCTA5CE20", "GEOID10")
	if zip == "" {
		return gisrepo.ZipRow{}, errNoZip
	}
	propsJSON, _ := json.Marshal(feat.Properties)
	return gisrepo.ZipRow{
		ZipCode:           zip,
		SourceKey:         FDOTAdminBoundariesKey,
		GeometryJSON:      geom,
		Properties:        propsJSON,
		SourceGeneration:  gen,
		SourceFingerprint: fingerprint,
	}, nil
}

var (
	errNoGeometry = errExtract("missing geometry")
	errNoParcelID = errExtract("missing parcel id")
	errNoName     = errExtract("missing name")
	errNoZip      = errExtract("missing zip code")
)

type errExtract string

func (e errExtract) Error() string { return string(e) }

func firstString(props map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := props[k]; ok && v != nil {
			s := strings.TrimSpace(fmtAny(v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func firstFloat(props map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := props[k]; ok && v != nil {
			switch t := v.(type) {
			case float64:
				return t
			case int:
				return float64(t)
			case json.Number:
				f, _ := t.Float64()
				return f
			case string:
				f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
				if err == nil {
					return f
				}
			}
		}
	}
	return 0
}

func firstInt(props map[string]any, keys ...string) int {
	f := firstFloat(props, keys...)
	if f == 0 {
		return 0
	}
	return int(f)
}

func fmtAny(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	default:
		b, _ := json.Marshal(t)
		return strings.Trim(string(b), `"`)
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func floatPtr(v float64) *float64 {
	if v == 0 {
		return nil
	}
	return &v
}

func intPtr(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}

func parseArcGISDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.UnixMilli(ms), nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
		"1/2/2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errExtract("unparseable date")
}
