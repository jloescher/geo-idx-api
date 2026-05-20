package bridge

import (
	"log/slog"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mlspoxy"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/bridge"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/images"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/audit"
	"github.com/quantyralabs/idx-api/internal/service/cache"
	"github.com/quantyralabs/idx-api/internal/service/search"
	"github.com/quantyralabs/idx-api/internal/service/sync"
)

// Handler implements Bridge proxy routes.
type Handler struct {
	cfg      config.Config
	factory  *mlspoxy.Factory
	rewriter *images.Rewriter
	audit    *audit.Logger
	listings *cache.ListingsService
	search   *search.Service
	stats    *sync.StatsService
	logger   *slog.Logger
}

func NewHandler(cfg config.Config, db *repository.DB, auditor *audit.Logger, logger *slog.Logger) *Handler {
	return &Handler{
		cfg:      cfg,
		factory:  mlspoxy.NewFactory(cfg),
		rewriter: images.NewRewriter(cfg),
		audit:    auditor,
		listings: cache.NewListingsService(cfg, db),
		search:   search.NewService(cfg, db),
		stats:    sync.NewStatsService(db),
		logger:   logger,
	}
}

func (h *Handler) Listings(c *fiber.Ctx) error {
	return h.proxyWeb(c, "listings.collection", "listings")
}

func (h *Handler) Listing(c *fiber.Ctx) error {
	return h.proxyWeb(c, "listings.detail", "listings/"+c.Params("listingId"))
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
	return h.proxyReso(c, "properties.collection", "Property")
}

func (h *Handler) Property(c *fiber.Ctx) error {
	return h.proxyReso(c, "properties.detail", "Property('"+c.Params("listingKey")+"')")
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

func (h *Handler) Lookup(c *fiber.Ctx) error { return h.proxyReso(c, "lookup", "Lookup") }

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
	feed := mlspoxy.Feed(c)
	ds := bridge.DatasetFromFeed(feed, h.cfg.Bridge.Dataset)
	cli := h.factory.ForRequest(c)
	var upstream string
	if feed.Provider == "spark" {
		// Spark web paths use live API
		upstream = h.cfg.Spark.APIHost + "/" + h.cfg.Spark.APIVersion + "/" + path
	} else {
		bc := bridge.NewClient(h.cfg, feed)
		upstream = bc.WebURL(path, ds)
	}
	return h.finishProxy(c, auditType, cli, upstream, "")
}

func (h *Handler) proxyReso(c *fiber.Ctx, auditType, entity string) error {
	feed := mlspoxy.Feed(c)
	ds := bridge.DatasetFromFeed(feed, h.cfg.Bridge.Dataset)
	cli := h.factory.ForRequest(c)
	var upstream string
	if feed.Provider == "spark" {
		sc := mlspoxy.Feed(c)
		_ = sc
		upstream = h.cfg.Spark.APIHost + "/" + h.cfg.Spark.APIVersion + "/" + h.cfg.Spark.LiveResoRoot + "/" + entity
	} else {
		bc := bridge.NewClient(h.cfg, feed)
		upstream = bc.ResoURL(entity, ds)
	}
	return h.finishProxy(c, auditType, cli, upstream, "")
}

func (h *Handler) proxyPub(c *fiber.Ctx, auditType, path string) error {
	bc := bridge.NewClient(h.cfg, mlspoxy.Feed(c))
	ds := bridge.DatasetFromFeed(mlspoxy.Feed(c), h.cfg.Bridge.Dataset)
	upstream := bc.WebURL(path, ds)
	return h.finishProxy(c, auditType, h.factory.ForRequest(c), upstream, "")
}

func (h *Handler) finishProxy(c *fiber.Ctx, auditType string, cli mlspoxy.ProxyClient, upstream, listingKey string) error {
	status, body, hdr, err := cli.Proxy(c, upstream)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	if status >= 200 && status < 300 {
		body = images.RewriteBytes(h.rewriter, body, mlspoxy.Feed(c).Dataset, listingKey)
	}
	h.audit.Log(c, auditType, nil)
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
