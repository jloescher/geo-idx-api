package gis

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
	gissvc "github.com/quantyralabs/idx-api/internal/service/gis"
)

// Handler serves GIS proxy routes.
type Handler struct {
	svc   *gissvc.ProxyService
	auto  *gissvc.AutocompleteService
}

func NewHandler(cfg config.Config, db *repository.DB, _ *slog.Logger) *Handler {
	repo := gisrepo.New(db)
	return &Handler{
		svc:  gissvc.NewProxyService(cfg, db),
		auto: gissvc.NewAutocompleteService(repo),
	}
}

func (h *Handler) Show(c *fiber.Ctx) error {
	return h.svc.Fetch(c, nil)
}

func (h *Handler) ShowForMLS(c *fiber.Ctx) error {
	code := c.Params("mlsCode")
	return h.svc.Fetch(c, &code)
}

func (h *Handler) AutocompleteCities(c *fiber.Ctx) error {
	q := c.Query("q")
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	results, err := h.auto.Cities(c.Context(), q, limit)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	if results == nil {
		results = []gissvc.CitySuggestion{}
	}
	return c.JSON(results)
}

func (h *Handler) AutocompleteCounties(c *fiber.Ctx) error {
	q := c.Query("q")
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	results, err := h.auto.Counties(c.Context(), q, limit)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	if results == nil {
		results = []gissvc.CountySuggestion{}
	}
	return c.JSON(results)
}
