package comps

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/comps"
)

// Handler serves POST /api/v1/comps/run.
type Handler struct {
	svc *comps.Service
}

func NewHandler(cfg config.Config, db *repository.DB, logger *slog.Logger) *Handler {
	return &Handler{svc: comps.NewService(cfg, db, logger)}
}

func (h *Handler) Run(c *fiber.Ctx) error {
	return h.svc.Run(c)
}
