package mls

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
	dom "github.com/quantyralabs/idx-api/internal/domain"
)

// FeedDefinition describes a catalog feed (see domain.FeedDefinition).
type FeedDefinition = dom.FeedDefinition

// Resolver resolves MLS feed from request (CheckMlsAccess parity).
type Resolver struct {
	cfg config.Config
}

func NewResolver(cfg config.Config) *Resolver {
	return &Resolver{cfg: cfg}
}

func (r *Resolver) Catalog() []FeedDefinition {
	var out []FeedDefinition
	for _, ds := range r.cfg.Bridge.Datasets {
		out = append(out, FeedDefinition{
			Code:     "bridge_" + ds,
			Provider: "bridge",
			Dataset:  ds,
		})
	}
	for _, ds := range r.cfg.Spark.Datasets {
		out = append(out, FeedDefinition{
			Code:     "spark_" + ds,
			Provider: "spark",
			Dataset:  ds,
		})
	}
	return out
}

func (r *Resolver) ResolveFeedCode(c *fiber.Ctx) (string, error) {
	if BypassGIS(c.Path()) {
		return "", nil
	}
	d, _ := c.Locals(ctxkeys.MLSDomain).(*dom.Domain)
	allowed := r.enabledForDomain(d)
	if len(allowed) == 0 {
		return "", fiber.NewError(fiber.StatusServiceUnavailable, "No MLS feeds enabled for this site.")
	}
	if ds := strings.TrimSpace(c.Query("dataset")); ds != "" {
		code := r.normalizeFeedCode(ds)
		if !contains(allowed, code) {
			return "", fiber.NewError(fiber.StatusForbidden, "MLS feed not enabled for this domain.")
		}
		return code, nil
	}
	if d != nil && d.MLSDataset != nil && *d.MLSDataset != "" {
		code := r.normalizeFeedCode(*d.MLSDataset)
		if contains(allowed, code) {
			return code, nil
		}
	}
	return allowed[0], nil
}

func (r *Resolver) FeedDefinition(code string) FeedDefinition {
	return FeedDefinitionFromCode(r.cfg, code)
}

// FeedDefinitionFromCode resolves catalog feed metadata from middleware feed code.
func FeedDefinitionFromCode(cfg config.Config, feedCode string) FeedDefinition {
	code := strings.ToLower(strings.TrimSpace(feedCode))
	for _, ds := range cfg.Bridge.Datasets {
		slug := strings.ToLower(ds)
		if code == "bridge_"+slug || code == slug {
			return FeedDefinition{Code: "bridge_" + slug, Provider: "bridge", Dataset: slug}
		}
	}
	for _, ds := range cfg.Spark.Datasets {
		slug := strings.ToLower(ds)
		if code == "spark_"+slug || code == slug {
			return FeedDefinition{Code: "spark_" + slug, Provider: "spark", Dataset: slug}
		}
	}
	if strings.HasPrefix(code, "spark_") {
		return FeedDefinition{Code: code, Provider: "spark", Dataset: strings.TrimPrefix(code, "spark_")}
	}
	if strings.HasPrefix(code, "bridge_") {
		return FeedDefinition{Code: code, Provider: "bridge", Dataset: strings.TrimPrefix(code, "bridge_")}
	}
	ds := strings.ToLower(cfg.Bridge.Dataset)
	return FeedDefinition{Code: "bridge_" + ds, Provider: "bridge", Dataset: ds}
}

// DatasetSlugFromFeedCode returns listings.dataset_slug (stellar, beaches, …).
func DatasetSlugFromFeedCode(feedCode string) string {
	code := strings.ToLower(strings.TrimSpace(feedCode))
	return strings.TrimPrefix(strings.TrimPrefix(code, "bridge_"), "spark_")
}

func (r *Resolver) normalizeFeedCode(ds string) string {
	ds = strings.TrimSpace(strings.ToLower(ds))
	if strings.HasPrefix(ds, "bridge_") || strings.HasPrefix(ds, "spark_") {
		return ds
	}
	return "bridge_" + ds
}

func (r *Resolver) enabledForDomain(d *dom.Domain) []string {
	catalog := r.Catalog()
	if d == nil {
		return codes(catalog)
	}
	var allowed []string
	if len(d.AllowedMLSDatasets) > 0 {
		_ = json.Unmarshal(d.AllowedMLSDatasets, &allowed)
	}
	if len(allowed) == 0 {
		return codes(catalog)
	}
	enabled := codes(catalog)
	var out []string
	for _, a := range allowed {
		a = r.normalizeFeedCode(a)
		if contains(enabled, a) {
			out = append(out, a)
		}
	}
	return out
}

func codes(defs []FeedDefinition) []string {
	out := make([]string, len(defs))
	for i, d := range defs {
		out[i] = d.Code
	}
	return out
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// BypassGIS returns true for GIS endpoints that skip feed resolution.
func BypassGIS(path string) bool {
	path = strings.TrimPrefix(path, "/")
	if path == "api/v1/gis" {
		return true
	}
	return strings.HasPrefix(path, "api/v1/mls/") && strings.HasSuffix(path, "/gis")
}
