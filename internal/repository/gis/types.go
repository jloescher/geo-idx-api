// Package gisrepo provides PostGIS persistence for parcel and boundary layers.
// Persistent GIS tables + monthly parcel refresh deliver instant Leaflet map performance →
// higher visitor engagement before the 3-listing hard gate → more OTP registrations and
// qualified leads while keeping marginal cost at zero.
package gisrepo

import (
	"encoding/json"
	"time"
)

// ParcelRow is a row in gis_parcels.
type ParcelRow struct {
	ParcelID          string
	SourceKey         string
	County            string
	GeometryJSON      string
	Properties        json.RawMessage
	SiteAddress       *string
	OwnerName         *string
	City              *string
	ZipCode           *string
	JustValue         *float64
	AssessedValue     *float64
	LandValue         *float64
	LivingAreaSqft    *float64
	YearBuilt         *int
	Acres             *float64
	LandUseCode       *string
	LastSalePrice     *float64
	LastSaleDate      *time.Time
	SourceGeneration  int
	SourceFingerprint *string
}

// CityRow is a row in gis_cities.
type CityRow struct {
	CityName          string
	County            *string
	SourceKey         string
	GeometryJSON      string
	Properties        json.RawMessage
	SourceGeneration  int
	SourceFingerprint *string
}

// CountyRow is a row in gis_counties.
type CountyRow struct {
	CountyName        string
	FIPSCode          *string
	SourceKey         string
	GeometryJSON      string
	Properties        json.RawMessage
	SourceGeneration  int
	SourceFingerprint *string
}

// ZipRow is a row in gis_zips.
type ZipRow struct {
	ZipCode           string
	SourceKey         string
	GeometryJSON      string
	Properties        json.RawMessage
	SourceGeneration  int
	SourceFingerprint *string
}

// FeatureResult holds a GeoJSON feature built from a persistent row.
type FeatureResult struct {
	GeometryJSON string
	Properties   json.RawMessage
}
