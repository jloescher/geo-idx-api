package gis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
)

// ArcGISFeature is a parsed GeoJSON feature from an ArcGIS query response.
type ArcGISFeature struct {
	Geometry   json.RawMessage
	Properties map[string]any
}

// ArcGISPageResult holds parsed features and pagination state.
type ArcGISPageResult struct {
	Features              []ArcGISFeature
	ExceededTransferLimit bool
}

// ArcGISClient performs paginated ArcGIS FeatureServer queries.
// Persistent GIS tables + monthly parcel refresh deliver instant Leaflet map performance →
// higher visitor engagement before the 3-listing hard gate → more OTP registrations and
// qualified leads while keeping marginal cost at zero.
type ArcGISClient struct {
	http     *http.Client
	pageSize int
}

// NewArcGISClient creates an ArcGIS HTTP client with configured page size.
func NewArcGISClient(cfg config.GISConfig) *ArcGISClient {
	pageSize := cfg.SyncPageSize
	if pageSize <= 0 {
		pageSize = 2000
	}
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 12 * time.Second
	}
	return &ArcGISClient{
		http:     &http.Client{Timeout: timeout},
		pageSize: pageSize,
	}
}

// FetchParcelPage queries a parcel source using sync mode (bbox, paginate, where_filter).
func (c *ArcGISClient) FetchParcelPage(spec ParcelSourceSpec, bbox BBox, offset, pageSize int) ([]byte, error) {
	u, err := url.Parse(spec.QueryURL)
	if err != nil {
		return nil, err
	}
	if pageSize <= 0 {
		pageSize = c.pageSize
	}
	q := baseQueryParams(spec.ResponseFormat)
	switch spec.SyncMode {
	case SyncModePaginate:
		if spec.ArcGISWhere != "" {
			q.Set("where", spec.ArcGISWhere)
		} else {
			q.Set("where", "1=1")
		}
	default:
		q.Set("geometry", bbox.EsriEnvelope())
		q.Set("geometryType", "esriGeometryEnvelope")
		q.Set("inSR", "4326")
		q.Set("spatialRel", "esriSpatialRelIntersects")
		if spec.ArcGISWhere != "" {
			q.Set("where", spec.ArcGISWhere)
		}
	}
	q.Set("resultOffset", fmt.Sprintf("%d", offset))
	q.Set("resultRecordCount", fmt.Sprintf("%d", pageSize))
	u.RawQuery = q.Encode()
	return c.get(u.String())
}

// FetchBBoxPage queries a parcel source by bounding box with pagination (legacy proxy helper).
func (c *ArcGISClient) FetchBBoxPage(src Source, bbox BBox, offset, pageSize int) ([]byte, error) {
	spec := ParcelSourceSpec{
		QueryURL:       src.QueryURL,
		SyncMode:       SyncModeBBox,
		ArcGISWhere:    src.Where,
		ResponseFormat: FormatGeoJSON,
	}
	if src.CountyCO != "" {
		spec.ArcGISWhere = "CO_NO=" + src.CountyCO
	}
	return c.FetchParcelPage(spec, bbox, offset, pageSize)
}

// FetchLayerPage queries a layer with optional WHERE clause and pagination.
func (c *ArcGISClient) FetchLayerPage(endpoint, where string, offset, pageSize int) ([]byte, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if pageSize <= 0 {
		pageSize = c.pageSize
	}
	q := baseQueryParams(FormatGeoJSON)
	if where != "" {
		q.Set("where", where)
	} else {
		q.Set("where", "1=1")
	}
	q.Set("resultOffset", fmt.Sprintf("%d", offset))
	q.Set("resultRecordCount", fmt.Sprintf("%d", pageSize))
	u.RawQuery = q.Encode()
	return c.get(u.String())
}

func baseQueryParams(format string) url.Values {
	q := url.Values{}
	if format == FormatJSON {
		q.Set("f", "json")
	} else {
		q.Set("f", "geojson")
	}
	q.Set("outFields", "*")
	q.Set("returnGeometry", "true")
	return q
}

func (c *ArcGISClient) get(rawURL string) ([]byte, error) {
	resp, err := c.http.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("arcgis status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := arcGISResponseError(body); err != nil {
		return nil, err
	}
	return body, nil
}

func arcGISResponseError(body []byte) error {
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil
	}
	errObj, ok := doc["error"].(map[string]any)
	if !ok {
		return nil
	}
	msg, _ := errObj["message"].(string)
	if msg == "" {
		msg = "arcgis query failed"
	}
	if details, ok := errObj["details"].([]any); ok && len(details) > 0 {
		if d, ok := details[0].(string); ok && d != "" {
			msg = d
		}
	}
	return fmt.Errorf("arcgis error: %s", msg)
}

// ParseFeatureCollection parses GeoJSON FeatureCollection from ArcGIS response.
func ParseFeatureCollection(body []byte) (ArcGISPageResult, error) {
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		return ArcGISPageResult{}, err
	}
	result := ArcGISPageResult{
		ExceededTransferLimit: boolField(doc, "exceededTransferLimit"),
	}
	rawFeats, ok := doc["features"].([]any)
	if !ok {
		return result, nil
	}
	for _, rf := range rawFeats {
		feat, ok := rf.(map[string]any)
		if !ok {
			continue
		}
		geom, _ := json.Marshal(feat["geometry"])
		props, _ := feat["properties"].(map[string]any)
		if props == nil {
			props, _ = feat["attributes"].(map[string]any)
		}
		if props == nil {
			props = map[string]any{}
		}
		result.Features = append(result.Features, ArcGISFeature{
			Geometry:   geom,
			Properties: props,
		})
	}
	return result, nil
}

func boolField(doc map[string]any, key string) bool {
	v, ok := doc[key].(bool)
	return ok && v
}

// FDOT boundary layer query endpoints.
const (
	FDOTCountiesURL = "https://gis.fdot.gov/arcgis/rest/services/Admin_Boundaries/FeatureServer/6/query"
	FDOTCitiesURL   = "https://gis.fdot.gov/arcgis/rest/services/Admin_Boundaries/FeatureServer/7/query"
	FDOTZipsURL     = "https://gis.fdot.gov/arcgis/rest/services/Admin_Boundaries/FeatureServer/8/query"
)

const FDOTAdminBoundariesKey = "fdot_admin_boundaries"
