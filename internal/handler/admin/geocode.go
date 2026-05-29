package admin

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/geocode"
)

// GeocodeHandler exposes operator endpoints for listings geocode enrichment.
type GeocodeHandler struct {
	cfg    config.Config
	db     *repository.DB
	logger *slog.Logger
}

// NewGeocodeHandler constructs the admin geocode handler.
func NewGeocodeHandler(cfg config.Config, db *repository.DB, logger *slog.Logger) *GeocodeHandler {
	return &GeocodeHandler{cfg: cfg, db: db, logger: logger}
}

type geocodeEnrichResponse struct {
	Enqueued        bool  `json:"enqueued"`
	PendingEstimate int64 `json:"pending_estimate"`
}

// Kickoff enqueues geocode listings enrichment when absent.
func (h *GeocodeHandler) Kickoff(c *fiber.Ctx) error {
	q := queue.NewClient(h.db.Pool, h.cfg.Queue.Table, h.cfg.Queue.NotifyChannel, h.cfg.Queue.RetryAfter, h.cfg.Queue.ReservationTimeout)
	svc := geocode.NewEnrichmentService(h.cfg, h.db, q, h.logger)
	pending, err := svc.CountPendingForAdmin(c.Context(), "")
	if err != nil {
		h.logger.Error("geocode enrich pending count", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to count pending listings"})
	}
	if err := svc.EnqueueKickoffIfAbsent(c.Context(), ""); err != nil {
		h.logger.Error("geocode enrich kickoff", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(geocodeEnrichResponse{Enqueued: true, PendingEstimate: pending})
}
