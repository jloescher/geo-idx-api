package gis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// ProxyService implements GIS parcel proxy with edge + origin cache tiers.
// Revenue impact: cached parcels reduce ArcGIS load and improve map UX conversion.
type ProxyService struct {
	cfg  config.Config
	db   *repository.DB
	http *http.Client
}

func NewProxyService(cfg config.Config, db *repository.DB) *ProxyService {
	return &ProxyService{
		cfg:  cfg,
		db:   db,
		http: &http.Client{Timeout: 12 * time.Second},
	}
}

func (s *ProxyService) Fetch(c *fiber.Ctx, mlsCode *string) error {
	if mlsCode != nil && !s.allowedMLS(*mlsCode) {
		return fiber.NewError(fiber.StatusNotFound, "MLS code not supported for GIS")
	}
	hash := queryHash(c)
	if body, ok := s.fromCache(c.Context(), hash); ok {
		c.Set("Content-Type", "application/geo+json")
		c.Set("X-IDX-Cache", "edge")
		return c.Send(body)
	}
	body, err := s.fetchArcGIS(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	_ = s.storeCache(c.Context(), hash, body)
	c.Set("Content-Type", "application/geo+json")
	c.Set("X-IDX-Cache", "miss")
	return c.Send(body)
}

func (s *ProxyService) allowedMLS(code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	for _, c := range s.cfg.GIS.FloridaMLSCodes {
		if strings.ToLower(c) == code {
			return true
		}
	}
	return false
}

func queryHash(c *fiber.Ctx) string {
	return fmt.Sprintf("%x", c.Request().URI().QueryArgs().String())
}

func (s *ProxyService) fromCache(ctx context.Context, hash string) ([]byte, bool) {
	var geojson string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT geojson FROM gis_cache WHERE query_hash = $1 AND expires_at > NOW()
	`, hash).Scan(&geojson)
	if err != nil {
		return nil, false
	}
	return []byte(geojson), true
}

func (s *ProxyService) storeCache(ctx context.Context, hash string, body []byte) error {
	exp := time.Now().Add(s.cfg.GIS.EdgeCacheTTL)
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO gis_cache (query_hash, geojson, expires_at, source_used, source_generation, created_at, updated_at)
		VALUES ($1, $2, $3, 'arcgis', 0, NOW(), NOW())
		ON CONFLICT (query_hash) DO UPDATE SET geojson = EXCLUDED.geojson, expires_at = EXCLUDED.expires_at, updated_at = NOW()
	`, hash, string(body), exp)
	return err
}

func (s *ProxyService) fetchArcGIS(c *fiber.Ctx) ([]byte, error) {
	// Default Florida statewide layer when bbox present
	bbox := c.Query("bbox")
	if bbox == "" {
		return json.Marshal(map[string]any{"type": "FeatureCollection", "features": []any{}})
	}
	layerURL := "https://services.arcgis.com/HRPe58PVRWYor63Q/arcgis/rest/services/Florida_Statewide_Cadastral/FeatureServer/0/query"
	u, _ := url.Parse(layerURL)
	q := u.Query()
	q.Set("f", "geojson")
	q.Set("geometry", bbox)
	q.Set("geometryType", "esriGeometryEnvelope")
	q.Set("spatialRel", "esriSpatialRelIntersects")
	q.Set("outFields", "*")
	q.Set("returnGeometry", "true")
	u.RawQuery = q.Encode()
	resp, err := s.http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
