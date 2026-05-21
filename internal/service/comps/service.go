package comps

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/audit"
)

// Service runs comps/BPO/investor analysis.
type Service struct {
	engine *Engine
	db     *repository.DB
	log    *slog.Logger
}

func NewService(cfg config.Config, db *repository.DB, log *slog.Logger) *Service {
	return &Service{
		engine: NewEngine(cfg, db),
		db:     db,
		log:    log,
	}
}

func (s *Service) Run(c *fiber.Ctx) error {
	var req RunRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid comps request")
	}
	feed, _ := c.Locals(ctxkeys.MLSFeedCode).(string)
	resp, err := s.engine.Run(c.Context(), feed, req)
	if err != nil {
		if isValidationErr(err) {
			return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
		}
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	n := len(resp.SoldComps) + len(resp.CompetitionComps)
	audit.NewLogger(s.db).Log(c, "comps.run", &n, nil)
	return c.JSON(resp)
}

func isValidationErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "required") ||
		strings.Contains(msg, "must be") ||
		strings.Contains(msg, "unsupported") ||
		errors.Is(err, errValidation)
}

var errValidation = errors.New("validation")
