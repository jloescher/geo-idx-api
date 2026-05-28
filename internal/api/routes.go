package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/quantyralabs/idx-api/internal/api/middleware"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/handler/admin"
	"github.com/quantyralabs/idx-api/internal/handler/auth"
	"github.com/quantyralabs/idx-api/internal/handler/bridge"
	"github.com/quantyralabs/idx-api/internal/handler/comps"
	"github.com/quantyralabs/idx-api/internal/handler/dashboard"
	"github.com/quantyralabs/idx-api/internal/handler/gis"
	"github.com/quantyralabs/idx-api/internal/handler/images"
	"github.com/quantyralabs/idx-api/internal/handler/marketing"
	"github.com/quantyralabs/idx-api/internal/openapi"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/audit"
)

// RegisterRoutes mounts all HTTP routes.
func RegisterRoutes(app *fiber.App, cfg config.Config, db *repository.DB, logger *slog.Logger) {
	if err := mountStatic(app); err != nil {
		logger.Error("static assets", "error", err)
	}

	openapi.Register(app)

	app.Get("/healthz", healthz)
	app.Get("/readyz", readyz(db))
	app.Get("/health/replicas", healthReplicas(db))
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	domains := repository.NewDomainRepo(db)
	tokens := repository.NewTokenRepo(db)
	auditor := audit.NewLogger(db)

	domainAuth := middleware.DomainToken(cfg, domains, tokens)
	mlsAccess := middleware.MLSAccess(cfg, domains)

	bridgeH := bridge.NewHandler(cfg, db, auditor, logger)
	gisH := gis.NewHandler(cfg, db, logger)
	imgH := images.NewHandler(cfg, db, logger)
	compsH := comps.NewHandler(cfg, db, logger)
	authH := auth.NewHandler(cfg, db, logger)
	dashH := dashboard.NewHandler(cfg, db, logger)
	mktH := marketing.NewHandler(cfg)

	// Image proxy (API host)
	app.Get("/images/:listingKey/:photoId", domainAuth, imgH.Show)

	// API group
	api := app.Group("/api")

	api.Post("/auth/token", authH.Token)
	api.Get("/auth/user", authH.User)

	v1 := api.Group("/v1", domainAuth, mlsAccess)

	// GIS (mls.access bypasses feed check inside middleware)
	v1.Get("/gis", gisH.Show)
	v1.Get("/mls/:mlsCode/gis", gisH.ShowForMLS)
	v1.Get("/gis/autocomplete/cities", gisH.AutocompleteCities)
	v1.Get("/gis/autocomplete/counties", gisH.AutocompleteCounties)

	// MLS web API (dataset-agnostic paths)
	v1.Get("/listings", bridgeH.Listings)
	v1.Get("/listings/:listingId", bridgeH.Listing)
	v1.Get("/agents", bridgeH.Agents)
	v1.Get("/agents/:agentId", bridgeH.Agent)
	v1.Get("/offices", bridgeH.Offices)
	v1.Get("/offices/:officeId", bridgeH.Office)
	v1.Get("/openhouses", bridgeH.OpenHouses)
	v1.Get("/openhouses/:openhouseId", bridgeH.OpenHouse)

	// RESO
	v1.Get("/properties", bridgeH.Properties)
	v1.Post("/properties", bridgeH.Properties)
	v1.Get("/properties/:listingKey", bridgeH.Property)
	v1.Get("/members", bridgeH.Members)
	v1.Get("/members/:memberKey", bridgeH.Member)
	v1.Get("/reso-offices", bridgeH.ResoOffices)
	v1.Get("/reso-offices/:officeKey", bridgeH.ResoOffice)
	v1.Get("/reso-openhouses", bridgeH.ResoOpenHouses)
	v1.Get("/reso-openhouses/:openHouseKey", bridgeH.ResoOpenHouse)
	v1.Get("/lookup", bridgeH.Lookup)

	// Pub parcels
	v1.Get("/pub/parcels", bridgeH.PubParcels)
	v1.Get("/pub/parcels/:parcelId", bridgeH.PubParcel)
	v1.Get("/pub/parcels/:parcelId/assessments", bridgeH.PubParcelAssessments)
	v1.Get("/pub/parcels/:parcelId/transactions", bridgeH.PubParcelTransactions)
	v1.Get("/pub/assessments", bridgeH.PubAssessments)
	v1.Get("/pub/transactions", bridgeH.PubTransactions)

	// Search, stats, comps
	v1.Post("/search", bridgeH.Search)
	v1.Get("/bridge/stats", bridgeH.Stats)
	v1.Post("/comps/run", compsH.Run)

	// Platform routes (host-checked in middleware or separate mount)
	app.Get("/", mktH.Home)
	dashH.Register(app)

	floodH := admin.NewFloodHandler(cfg, db, logger)
	gisAdminH := admin.NewGISHandler(cfg, db, logger)
	adminAPI := api.Group("/v1/admin", dashH.SessionAuthMiddleware)
	adminAPI.Get("/monitoring", dashH.MonitoringJSON)
	adminAPI.Post("/flood-enrich", floodH.Enrich)
	adminGIS := adminAPI.Group("/gis", dashH.RequireAdmin)
	admin.RegisterGISRoutes(adminGIS, gisAdminH)
}

func healthz(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func readyz(db *repository.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"ready": false, "error": err.Error()})
		}
		ver, _ := db.PostGISVersion(ctx)
		return c.JSON(fiber.Map{"ready": true, "postgis": ver})
	}
}

func healthReplicas(db *repository.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if db.Selector == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "read replica selector not configured",
			})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		snap := db.Selector.Snapshot()
		if err := db.Ping(ctx); err != nil && snap.FallbackActive {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":           err.Error(),
				"replicas":        snap.Replicas,
				"selected":        snap.Selected,
				"fallback_active": snap.FallbackActive,
			})
		}
		return c.JSON(snap)
	}
}
