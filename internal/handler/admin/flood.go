package admin

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/fema"
)

// FloodHandler exposes operator endpoints for FEMA flood enrichment.
type FloodHandler struct {
	cfg    config.Config
	db     *repository.DB
	logger *slog.Logger
}

// NewFloodHandler constructs the admin flood handler.
func NewFloodHandler(cfg config.Config, db *repository.DB, logger *slog.Logger) *FloodHandler {
	return &FloodHandler{cfg: cfg, db: db, logger: logger}
}

type floodEnrichRequest struct {
	Limit       int    `json:"limit"`
	DatasetSlug string `json:"dataset_slug"`
}

type floodEnrichResponse struct {
	Enqueued      bool  `json:"enqueued"`
	StaleEstimate int64 `json:"stale_estimate"`
}

// Enrich enqueues FEMA flood enrichment (kickoff or a single limited batch for staging).
func (h *FloodHandler) Enrich(c *fiber.Ctx) error {
	var req floodEnrichRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid JSON body"})
		}
	}

	q := queue.NewClient(h.db.Pool, h.cfg.Queue.Table, h.cfg.Queue.NotifyChannel, h.cfg.Queue.RetryAfter, h.cfg.Queue.ReservationTimeout)
	svc := fema.NewEnrichmentService(h.cfg, h.db, q, h.logger)

	stale, err := svc.CountStaleForAdmin(c.Context(), req.DatasetSlug)
	if err != nil {
		h.logger.Error("fema flood enrich stale count", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to count stale listings"})
	}

	if req.Limit > 0 {
		limit := req.Limit
		if limit > h.cfg.FEMA.EnrichBatchSize && h.cfg.FEMA.EnrichBatchSize > 0 {
			limit = h.cfg.FEMA.EnrichBatchSize
		}
		_, err = q.Enqueue(c.Context(), h.cfg.FEMA.EnrichQueue, queue.TypeFEMAFloodEnrichBatch, fema.FloodEnrichBatchArgs{
			CursorID:    0,
			Limit:       limit,
			DatasetSlug: req.DatasetSlug,
		}, 0)
		if err != nil {
			h.logger.Error("fema flood enrich batch enqueue", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to enqueue batch"})
		}
		return c.JSON(floodEnrichResponse{Enqueued: true, StaleEstimate: stale})
	}

	_, enqueued, err := svc.AdminKickoff(c.Context(), req.DatasetSlug)
	// #region agent log
	agentDebugLog("FEMA-B", "flood.go:Enrich", "kickoff result", map[string]any{
		"enqueued": enqueued, "stale": stale, "err": err != nil,
	})
	// #endregion
	if err != nil {
		h.logger.Error("fema flood enrich kickoff", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to enqueue kickoff"})
	}
	return c.JSON(floodEnrichResponse{Enqueued: enqueued, StaleEstimate: stale})
}
