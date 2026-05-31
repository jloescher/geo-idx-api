package bridge

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mlspoxy"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/images"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/upstream"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/audit"
	"github.com/quantyralabs/idx-api/internal/service/cache"
	"github.com/quantyralabs/idx-api/internal/service/crypto"
	"github.com/quantyralabs/idx-api/internal/service/mls"
	"github.com/quantyralabs/idx-api/internal/service/search"
	"github.com/quantyralabs/idx-api/internal/service/sync"
)

// proxyCacheStore is the cache surface used by finishProxy (implemented by *cache.ProxyCache).
type proxyCacheStore interface {
	Get(ctx context.Context, partition, fingerprint string) ([]byte, bool, error)
	Put(ctx context.Context, partition, fingerprint string, body []byte) error
}

// mlsClientFactory selects the upstream MLS client for the active feed.
type mlsClientFactory interface {
	ForRequest(c *fiber.Ctx) mlspoxy.ProxyClient
}

// Handler implements MLS RESO/web proxy routes (Bridge and Spark feeds).
type Handler struct {
	cfg        config.Config
	factory    mlsClientFactory
	resolver   *mlspoxy.UpstreamResolver
	rewriter   *images.Rewriter
	audit      *audit.Logger
	proxyCache proxyCacheStore
	pricing    *crypto.PricingReader
	search     *search.Service
	stats      *sync.StatsService
	logger     *slog.Logger
}

func NewHandler(cfg config.Config, db *repository.DB, auditor *audit.Logger, logger *slog.Logger) *Handler {
	return &Handler{
		cfg:        cfg,
		factory:    mlspoxy.NewFactory(cfg, sync.NewSparkClusterRateLimiter(db.Pool, cfg)),
		resolver:   mlspoxy.NewUpstreamResolver(cfg),
		rewriter:   images.NewRewriter(cfg),
		audit:      auditor,
		proxyCache: cache.NewProxyCache(cfg, db),
		pricing:    crypto.NewPricingReader(db),
		search:     search.NewService(cfg, db, cache.NewProxyCache(cfg, db)),
		stats:      sync.NewStatsService(db),
		logger:     logger,
	}
}

func (h *Handler) Listings(c *fiber.Ctx) error {
	return h.proxyWeb(c, "listings.collection", "listings")
}

func (h *Handler) Listing(c *fiber.Ctx) error {
	return h.proxyWebWithKey(c, "listings.detail", "listings/"+c.Params("listingId"), c.Params("listingId"))
}

func (h *Handler) Agents(c *fiber.Ctx) error { return h.proxyWeb(c, "agents.collection", "agents") }
func (h *Handler) Agent(c *fiber.Ctx) error {
	return h.proxyWeb(c, "agents.detail", "agents/"+c.Params("agentId"))
}
func (h *Handler) Offices(c *fiber.Ctx) error { return h.proxyWeb(c, "offices.collection", "offices") }
func (h *Handler) Office(c *fiber.Ctx) error {
	return h.proxyWeb(c, "offices.detail", "offices/"+c.Params("officeId"))
}
func (h *Handler) OpenHouses(c *fiber.Ctx) error {
	return h.proxyWeb(c, "openhouses.collection", "openhouses")
}
func (h *Handler) OpenHouse(c *fiber.Ctx) error {
	return h.proxyWeb(c, "openhouses.detail", "openhouses/"+c.Params("openhouseId"))
}

func (h *Handler) Properties(c *fiber.Ctx) error {
	return h.proxyPropertiesCollection(c)
}

func (h *Handler) Property(c *fiber.Ctx) error {
	return h.proxyResoWithKey(c, "properties.detail", "Property('"+c.Params("listingKey")+"')", c.Params("listingKey"))
}

func (h *Handler) Members(c *fiber.Ctx) error { return h.proxyReso(c, "members.collection", "Member") }
func (h *Handler) Member(c *fiber.Ctx) error {
	return h.proxyReso(c, "members.detail", "Member('"+c.Params("memberKey")+"')")
}

func (h *Handler) ResoOffices(c *fiber.Ctx) error {
	return h.proxyReso(c, "reso-offices.collection", "Office")
}
func (h *Handler) ResoOffice(c *fiber.Ctx) error {
	return h.proxyReso(c, "reso-offices.detail", "Office('"+c.Params("officeKey")+"')")
}

func (h *Handler) ResoOpenHouses(c *fiber.Ctx) error {
	return h.proxyReso(c, "reso-openhouses.collection", "OpenHouse")
}

func (h *Handler) ResoOpenHouse(c *fiber.Ctx) error {
	return h.proxyReso(c, "reso-openhouses.detail", "OpenHouse('"+c.Params("openHouseKey")+"')")
}

func (h *Handler) Lookup(c *fiber.Ctx) error { return h.proxyResoLookup(c, "lookup", "Lookup") }

func (h *Handler) PubParcels(c *fiber.Ctx) error {
	return h.proxyPub(c, "pub.parcels", "pub/parcels")
}

func (h *Handler) PubParcel(c *fiber.Ctx) error {
	return h.proxyPub(c, "pub.parcel", "pub/parcels/"+c.Params("parcelId"))
}

func (h *Handler) PubParcelAssessments(c *fiber.Ctx) error {
	return h.proxyPub(c, "pub.parcel.assessments", "pub/parcels/"+c.Params("parcelId")+"/assessments")
}

func (h *Handler) PubParcelTransactions(c *fiber.Ctx) error {
	return h.proxyPub(c, "pub.parcel.transactions", "pub/parcels/"+c.Params("parcelId")+"/transactions")
}

func (h *Handler) PubAssessments(c *fiber.Ctx) error {
	return h.proxyPub(c, "pub.assessments", "pub/assessments")
}
func (h *Handler) PubTransactions(c *fiber.Ctx) error {
	return h.proxyPub(c, "pub.transactions", "pub/transactions")
}

func (h *Handler) Search(c *fiber.Ctx) error {
	return h.search.Handle(c)
}

func (h *Handler) Stats(c *fiber.Ctx) error {
	return h.stats.Handle(c)
}

func (h *Handler) proxyWeb(c *fiber.Ctx, auditType, path string) error {
	return h.proxyWebWithKey(c, auditType, path, "")
}

func (h *Handler) proxyWebWithKey(c *fiber.Ctx, auditType, path, listingKey string) error {
	feed := mlspoxy.Feed(c)
	cli := h.factory.ForRequest(c)
	upstreamURL := h.resolver.WebURL(feed, path)
	return h.finishProxy(c, auditType, cli, upstream.SingleURL(upstreamURL, "web"), listingKey, cache.WebPartition(h.domainSlug(c), h.feedCode(c), auditType))
}

func (h *Handler) proxyReso(c *fiber.Ctx, auditType, entity string) error {
	return h.proxyResoWithKey(c, auditType, entity, "")
}

func (h *Handler) proxyResoWithKey(c *fiber.Ctx, auditType, entity, listingKey string) error {
	feed := mlspoxy.Feed(c)
	cli := h.factory.ForRequest(c)
	candidates := upstream.BuildResoCandidates(h.cfg, feed, entity)
	return h.finishProxy(c, auditType, cli, candidates, listingKey, cache.ResoPartition(h.domainSlug(c), h.feedCode(c), entity))
}

func (h *Handler) proxyResoLookup(c *fiber.Ctx, auditType, entity string) error {
	feed := mlspoxy.Feed(c)
	cli := h.factory.ForRequest(c)
	candidates := upstream.BuildResoCandidates(h.cfg, feed, entity)
	partition := cache.LookupPartition(h.domainSlug(c), h.feedCode(c))
	return h.finishProxy(c, auditType, cli, candidates, "", partition)
}

func (h *Handler) proxyPub(c *fiber.Ctx, auditType, path string) error {
	upstreamURL := h.resolver.PubURL(path)
	return h.finishProxy(c, auditType, h.factory.ForRequest(c), upstream.SingleURL(upstreamURL, "pub"), "", cache.WebPartition(h.domainSlug(c), h.feedCode(c), auditType))
}

func (h *Handler) domainSlug(c *fiber.Ctx) string {
	s, _ := c.Locals(ctxkeys.MLSDomainSlug).(string)
	return s
}

func (h *Handler) feedCode(c *fiber.Ctx) string {
	f, _ := c.Locals(ctxkeys.MLSFeedCode).(string)
	if f == "" {
		return "bridge_" + h.cfg.Bridge.Dataset
	}
	return f
}

func (h *Handler) finishProxy(c *fiber.Ctx, auditType string, cli mlspoxy.ProxyClient, candidates []upstream.Candidate, listingKey, partition string) error {
	return h.finishProxyMethod(c, auditType, cli, candidates, listingKey, partition, "")
}

func (h *Handler) finishProxyMethod(c *fiber.Ctx, auditType string, cli mlspoxy.ProxyClient, candidates []upstream.Candidate, listingKey, partition, upstreamMethod string) error {
	if len(candidates) == 0 {
		return fiber.NewError(fiber.StatusBadGateway, "no upstream candidates")
	}

	if partition != "" {
		for _, cand := range candidates {
			fp := cache.FingerprintWithLeg(c, cand.Leg)
			if body, ok, err := h.proxyCache.Get(c.Context(), partition, fp); err == nil && ok {
				c.Set("X-IDX-Cache", "HIT")
				c.Set("X-IDX-Upstream-Leg", cand.Leg)
				hit := "HIT"
				h.audit.Log(c, auditType, nil, &hit)
				c.Set("Content-Type", "application/json")
				if c.Query("include_pricing") == "1" {
					body = h.pricing.InjectIntoJSON(c.Context(), body)
				}
				return c.Status(fiber.StatusOK).Send(body)
			}
		}
	}

	result, err := upstream.FetchWithFallbackMethod(c, cli, candidates, upstreamMethod)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	status, body, hdr, leg := result.Status, result.Body, result.Header, result.Leg

	var cacheHit *string
	if status >= 200 && status < 300 {
		body = images.RewriteBytes(h.rewriter, body, mlspoxy.Feed(c).Dataset, listingKey)
		if listingKey != "" {
			body = mls.SanitizeUpstreamPropertyJSONWithDataset(body, mlspoxy.Feed(c).Dataset)
		}
		if c.Query("include_pricing") == "1" {
			body = h.pricing.InjectIntoJSON(c.Context(), body)
		}
		if partition != "" {
			fp := cache.FingerprintWithLeg(c, leg)
			_ = h.proxyCache.Put(c.Context(), partition, fp, body)
			c.Set("X-IDX-Cache", "MISS")
			miss := "MISS"
			cacheHit = &miss
		}
	}
	h.audit.Log(c, auditType, nil, cacheHit)
	c.Set("X-IDX-Upstream-Leg", leg)
	c.Set("Content-Type", "application/json")
	if etags := hdr["Etag"]; len(etags) > 0 {
		c.Set("ETag", etags[0])
	} else if etags := hdr["ETag"]; len(etags) > 0 {
		c.Set("ETag", etags[0])
	}
	return c.Status(status).Send(body)
}

// Ensure unused import
var _ = http.StatusOK
