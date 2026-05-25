package gis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
)

// ProxyService implements GIS parcel proxy with edge cache and persistent PostGIS reads.
// Persistent GIS tables + monthly parcel refresh deliver instant Leaflet map performance →
// higher visitor engagement before the 3-listing hard gate → more OTP registrations and
// qualified leads while keeping marginal cost at zero.
type ProxyService struct {
	cfg        config.Config
	db         *repository.DB
	repo       *gisrepo.Repository
	persistent *PersistentQueryService
	arcgis     *ArcGISClient
}

func NewProxyService(cfg config.Config, db *repository.DB) *ProxyService {
	repo := gisrepo.New(db)
	return &ProxyService{
		cfg:        cfg,
		db:         db,
		repo:       repo,
		persistent: NewPersistentQueryService(cfg, repo),
		arcgis:     NewArcGISClient(cfg.GIS),
	}
}

func (s *ProxyService) Fetch(c *fiber.Ctx, mlsCode *string) error {
	if mlsCode != nil && !s.allowedMLS(*mlsCode) {
		return fiber.NewError(fiber.StatusNotFound, "MLS code not supported for GIS")
	}
	bbox, err := ParseBBox(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if bbox.SpanDeg() > s.cfg.GIS.MaxBboxSpanDeg {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "bbox span exceeds limit")
	}

	qtype := ParseQueryType(c.Query("type"))
	limit := ParseLimit(c.Query("limit"), s.cfg.GIS)
	fullAccess := requestFullAccess(c)

	hash := queryHash(c)
	if body, meta, ok := s.fromCache(c.Context(), hash, fullAccess, qtype); ok {
		body, _ = applyTeaser(body, s.cfg.GIS, fullAccess)
		c.Set("Content-Type", "application/geo+json")
		c.Set("X-IDX-Cache", "edge")
		return c.Send(wrapWithMeta(body, meta))
	}

	hasData, err := s.persistent.HasData(c.Context(), qtype, bbox)
	if err == nil && hasData {
		raw, _, qerr := s.persistent.Query(c.Context(), qtype, bbox, limit)
		if qerr == nil && len(raw) > 0 {
			meta := s.buildPersistentMeta(c, bbox, qtype, fullAccess)
			gen, _ := s.repo.SourceGeneration(c.Context(), meta.SourceUsed)
			_ = s.storeCache(c.Context(), hash, raw, meta.SourceUsed, gen)
			body, _ := applyTeaser(raw, s.cfg.GIS, fullAccess)
			c.Set("Content-Type", "application/geo+json")
			c.Set("X-IDX-Cache", "persistent")
			return c.Send(wrapWithMeta(body, meta))
		}
	}

	if qtype != QueryTypeParcel {
		raw := []byte(`{"type":"FeatureCollection","features":[]}`)
		meta := s.buildPersistentMeta(c, bbox, qtype, fullAccess)
		meta.Degraded = true
		warn := fmt.Sprintf("No persistent %s boundaries loaded for this area; run gis.annual_boundaries_refresh.", qtype)
		meta.Warnings = []string{warn}
		body, _ := applyTeaser(raw, s.cfg.GIS, fullAccess)
		c.Set("Content-Type", "application/geo+json")
		c.Set("X-IDX-Cache", "miss")
		return c.Send(wrapWithMeta(body, meta))
	}

	var lastErr error
	var used Source
	var raw []byte
	for _, src := range sourcesForBBox(bbox) {
		raw, lastErr = s.fetchSource(src, bbox)
		if lastErr == nil && len(raw) > 0 {
			used = src
			break
		}
	}
	meta := s.buildMeta(c, bbox, used, lastErr, fullAccess)
	if lastErr != nil && len(raw) == 0 {
		raw = []byte(`{"type":"FeatureCollection","features":[]}`)
		meta.Degraded = true
		fallback := "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
		meta.LeafletFallback = &fallback
	}
	gen, _ := s.sourceGeneration(c.Context(), meta.SourceUsed)
	_ = s.storeCache(c.Context(), hash, raw, meta.SourceUsed, gen)
	body, _ := applyTeaser(raw, s.cfg.GIS, fullAccess)
	c.Set("Content-Type", "application/geo+json")
	c.Set("X-IDX-Cache", "miss")
	return c.Send(wrapWithMeta(body, meta))
}

type responseMeta struct {
	SourceUsed      string   `json:"source_used"`
	SourceTier      string   `json:"source_tier"`
	CountyHint      string   `json:"county_hint"`
	QueryType       string   `json:"query_type,omitempty"`
	Teaser          bool     `json:"teaser"`
	FullAccess      bool     `json:"full_access"`
	Degraded        bool     `json:"degraded"`
	LeafletFallback *string  `json:"leaflet_fallback"`
	CacheGeneration int64    `json:"cache_generation"`
	Cached          bool     `json:"cached"`
	CacheHit        *string  `json:"cache_hit"`
	Warnings        []string `json:"warnings,omitempty"`
}

func (s *ProxyService) buildPersistentMeta(c *fiber.Ctx, bbox BBox, qtype QueryType, fullAccess bool) responseMeta {
	lat, lng := bbox.Centroid()
	hint := countyHint(lat, lng)
	sourceUsed := "gis_parcels"
	tier := "persistent"
	switch qtype {
	case QueryTypeCity:
		sourceUsed = "gis_cities"
	case QueryTypeCounty:
		sourceUsed = "gis_counties"
	case QueryTypeZip:
		sourceUsed = "gis_zips"
	}
	gen, _ := s.repo.SourceGeneration(c.Context(), sourceUsed)
	if gen == 0 {
		gen, _ = s.repo.SourceGeneration(c.Context(), FDOTAdminBoundariesKey)
	}
	return responseMeta{
		SourceUsed:      sourceUsed,
		SourceTier:      tier,
		CountyHint:      hint,
		QueryType:       string(qtype),
		Teaser:          !fullAccess,
		FullAccess:      fullAccess,
		CacheGeneration: gen,
		Cached:          false,
	}
}

func (s *ProxyService) buildMeta(c *fiber.Ctx, bbox BBox, src Source, fetchErr error, fullAccess bool) responseMeta {
	lat, lng := bbox.Centroid()
	hint := countyHint(lat, lng)
	gen, _ := s.sourceGeneration(c.Context(), src.Key)
	m := responseMeta{
		SourceUsed:      src.Key,
		SourceTier:      src.Tier,
		CountyHint:      hint,
		QueryType:       "parcel",
		Teaser:          !fullAccess,
		FullAccess:      fullAccess,
		Degraded:        fetchErr != nil,
		CacheGeneration: gen,
		Cached:          false,
	}
	if m.SourceUsed == "" {
		m.SourceUsed = "unknown"
		m.SourceTier = "none"
	}
	return m
}

func wrapWithMeta(geojson []byte, meta responseMeta) []byte {
	var fc map[string]any
	if err := json.Unmarshal(geojson, &fc); err != nil {
		return geojson
	}
	fc["meta"] = meta
	out, err := json.Marshal(fc)
	if err != nil {
		return geojson
	}
	return out
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

func (s *ProxyService) sourceGeneration(ctx context.Context, key string) (int64, error) {
	return s.repo.SourceGeneration(ctx, key)
}

func (s *ProxyService) fromCache(ctx context.Context, hash string, fullAccess bool, qtype QueryType) ([]byte, responseMeta, bool) {
	pool, err := s.db.ReadPool(ctx)
	if err != nil {
		return nil, responseMeta{}, false
	}
	var geojson, sourceUsed string
	var gen int64
	err = pool.QueryRow(ctx, `
		SELECT geojson, source_used, source_generation FROM gis_cache
		WHERE query_hash = $1 AND expires_at > NOW()
	`, hash).Scan(&geojson, &sourceUsed, &gen)
	if err != nil {
		return nil, responseMeta{}, false
	}
	hit := "edge"
	meta := responseMeta{
		SourceUsed:      sourceUsed,
		QueryType:       string(qtype),
		CacheGeneration: gen,
		Cached:          true,
		CacheHit:        &hit,
		Teaser:          !fullAccess,
		FullAccess:      fullAccess,
		SourceTier:      "cached",
	}
	return []byte(geojson), meta, true
}

func (s *ProxyService) storeCache(ctx context.Context, hash string, body []byte, sourceKey string, gen int64) error {
	exp := time.Now().Add(s.cfg.GIS.EdgeCacheTTL)
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO gis_cache (query_hash, geojson, expires_at, source_used, source_generation, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (query_hash) DO UPDATE SET geojson = EXCLUDED.geojson, expires_at = EXCLUDED.expires_at,
			source_used = EXCLUDED.source_used, source_generation = EXCLUDED.source_generation, updated_at = NOW()
	`, hash, string(body), exp, sourceKey, gen)
	return err
}

func (s *ProxyService) fetchSource(src Source, bbox BBox) ([]byte, error) {
	limit := s.cfg.GIS.MaxFeatures
	if limit <= 0 {
		limit = 500
	}
	return s.arcgis.FetchBBoxPage(src, bbox, 0, limit)
}
