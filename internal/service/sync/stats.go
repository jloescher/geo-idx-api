package sync

import (
	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// StatsService serves GET /api/v1/bridge/stats.
type StatsService struct {
	db *repository.DB
}

func NewStatsService(db *repository.DB) *StatsService {
	return &StatsService{db: db}
}

func (s *StatsService) Handle(c *fiber.Ctx) error {
	rows, err := s.db.Pool.Query(c.Context(), `
		SELECT dataset_slug,
		       COUNT(*) FILTER (WHERE LOWER(TRIM(COALESCE(standard_status,''))) IN ('active','pending')) AS active_pending,
		       MAX(modification_timestamp) AS latest_mod
		FROM listings GROUP BY dataset_slug
	`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()
	type row struct {
		Dataset       string  `json:"dataset_slug"`
		ActivePending int64   `json:"active_pending"`
		LatestMod     *string `json:"latest_modification"`
	}
	var out []row
	for rows.Next() {
		var r row
		var latest *string
		if err := rows.Scan(&r.Dataset, &r.ActivePending, &latest); err != nil {
			return err
		}
		r.LatestMod = latest
		out = append(out, r)
	}
	return c.JSON(fiber.Map{"datasets": out})
}
