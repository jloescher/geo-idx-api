package gis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/quantyralabs/idx-api/internal/config"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
)

// QueryType selects which persistent GIS table to query.
type QueryType string

const (
	QueryTypeParcel QueryType = "parcel"
	QueryTypeCity   QueryType = "city"
	QueryTypeCounty QueryType = "county"
	QueryTypeZip    QueryType = "zip"
)

// PersistentQueryService reads GeoJSON from PostGIS persistent tables.
// Persistent GIS tables + monthly parcel refresh deliver instant Leaflet map performance →
// higher visitor engagement before the 3-listing hard gate → more OTP registrations and
// qualified leads while keeping marginal cost at zero.
type PersistentQueryService struct {
	cfg  config.Config
	repo *gisrepo.Repository
}

// NewPersistentQueryService creates a persistent GIS query service.
func NewPersistentQueryService(cfg config.Config, repo *gisrepo.Repository) *PersistentQueryService {
	return &PersistentQueryService{cfg: cfg, repo: repo}
}

// ParseQueryType reads the type query param (default parcel).
func ParseQueryType(raw string) QueryType {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "city", "cities":
		return QueryTypeCity
	case "county", "counties":
		return QueryTypeCounty
	case "zip", "zips", "postal":
		return QueryTypeZip
	default:
		return QueryTypeParcel
	}
}

// HasData reports whether persistent rows exist for the query type and bbox.
func (s *PersistentQueryService) HasData(ctx context.Context, qtype QueryType, bbox BBox) (bool, error) {
	w, so, e, n := bbox.West, bbox.South, bbox.East, bbox.North
	switch qtype {
	case QueryTypeCity:
		return s.repo.HasCitiesInBBox(ctx, w, so, e, n)
	case QueryTypeCounty:
		return s.repo.HasCountiesInBBox(ctx, w, so, e, n)
	case QueryTypeZip:
		return s.repo.HasZipsInBBox(ctx, w, so, e, n)
	default:
		counties := countiesForBBox(bbox)
		return s.repo.HasParcelsInBBox(ctx, w, so, e, n, counties)
	}
}

// Query returns GeoJSON FeatureCollection bytes and feature count.
func (s *PersistentQueryService) Query(ctx context.Context, qtype QueryType, bbox BBox, limit int) ([]byte, int, error) {
	if limit <= 0 {
		limit = s.cfg.GIS.MaxFeatures
	}
	if limit <= 0 {
		limit = 500
	}
	w, so, e, n := bbox.West, bbox.South, bbox.East, bbox.North
	var feats []gisrepo.FeatureResult
	var err error
	switch qtype {
	case QueryTypeCity:
		feats, err = s.repo.QueryCitiesByBBox(ctx, w, so, e, n, limit)
	case QueryTypeCounty:
		feats, err = s.repo.QueryCountiesByBBox(ctx, w, so, e, n, limit)
	case QueryTypeZip:
		feats, err = s.repo.QueryZipsByBBox(ctx, w, so, e, n, limit)
	default:
		counties := countiesForBBox(bbox)
		feats, err = s.repo.QueryParcelsByBBox(ctx, w, so, e, n, counties, limit)
	}
	if err != nil {
		return nil, 0, err
	}
	body, err := featuresToGeoJSON(feats)
	return body, len(feats), err
}

func countiesForBBox(b BBox) []string {
	lat, lng := b.Centroid()
	hint := countyHint(lat, lng)
	if hint == "" {
		return nil
	}
	return []string{hint, "statewide"}
}

func featuresToGeoJSON(feats []gisrepo.FeatureResult) ([]byte, error) {
	features := make([]map[string]any, 0, len(feats))
	for _, f := range feats {
		var geom any
		if err := json.Unmarshal([]byte(f.GeometryJSON), &geom); err != nil {
			continue
		}
		var props any
		if len(f.Properties) > 0 {
			_ = json.Unmarshal(f.Properties, &props)
		}
		if props == nil {
			props = map[string]any{}
		}
		features = append(features, map[string]any{
			"type":       "Feature",
			"geometry":   geom,
			"properties": props,
		})
	}
	fc := map[string]any{
		"type":     "FeatureCollection",
		"features": features,
	}
	return json.Marshal(fc)
}

// ParseLimit reads limit query param capped by GIS_MAX_FEATURES.
func ParseLimit(raw string, cfg config.GISConfig) int {
	max := cfg.MaxFeatures
	if max <= 0 {
		max = 500
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return max
	}
	var n int
	if _, err := fmt.Sscanf(raw, "%d", &n); err != nil || n <= 0 {
		return max
	}
	if n > max {
		return max
	}
	return n
}
