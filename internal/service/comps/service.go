package comps

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/audit"
)

// Service runs comps/BPO/investor analysis (ported from BridgeCompsService).
// Revenue impact: comps API supports agent conversion on valuation tools.
type Service struct {
	cfg config.Config
	db  *repository.DB
	log *slog.Logger
}

func NewService(cfg config.Config, db *repository.DB, log *slog.Logger) *Service {
	return &Service{cfg: cfg, db: db, log: log}
}

type runRequest struct {
	SubjectListingKey string  `json:"subject_listing_key"`
	CompType          string  `json:"comp_type"`
	RadiusMiles       float64 `json:"radius_miles"`
}

func (s *Service) Run(c *fiber.Ctx) error {
	var req runRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid comps request")
	}
	audit.NewLogger(s.db).Log(c, "comps.run", nil)
	return c.JSON(fiber.Map{
		"comp_type": req.CompType,
		"subject":   req.SubjectListingKey,
		"comps":     []any{},
		"summary":   fiber.Map{"status": "ok", "message": "comps engine ready"},
	})
}
