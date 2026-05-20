package gis

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	gissvc "github.com/quantyralabs/idx-api/internal/service/gis"
)

// Handler serves GIS proxy routes.
type Handler struct {
	svc *gissvc.ProxyService
}

func NewHandler(cfg config.Config, db *repository.DB, _ *slog.Logger) *Handler {
	return &Handler{svc: gissvc.NewProxyService(cfg, db)}
}

func (h *Handler) Show(c *fiber.Ctx) error {
	return h.svc.Fetch(c, nil)
}

func (h *Handler) ShowForMLS(c *fiber.Ctx) error {
	code := c.Params("mlsCode")
	return h.svc.Fetch(c, &code)
}
