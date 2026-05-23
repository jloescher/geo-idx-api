package sync

import (
	"time"

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
	pool, err := s.db.ReadPool(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	rows, err := pool.Query(c.Context(), `
		SELECT COALESCE(c.dataset_slug, s.dataset_slug) AS dataset_slug,
		       COALESCE(s.active_pending, 0) AS active_pending,
		       s.latest_mod,
		       c.last_modification_timestamp,
		       c.last_sync_finished_at,
		       c.incremental_window_end,
		       COALESCE(c.replication_in_progress, false) AS replication_in_progress
		FROM listing_sync_cursors c
		FULL OUTER JOIN (
			SELECT dataset_slug,
			       COUNT(*) FILTER (WHERE LOWER(TRIM(COALESCE(standard_status,''))) IN ('active','pending')) AS active_pending,
			       MAX(modification_timestamp) AS latest_mod
			FROM listings
			GROUP BY dataset_slug
		) s ON s.dataset_slug = c.dataset_slug
		ORDER BY dataset_slug
	`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	type row struct {
		Dataset               string     `json:"dataset_slug"`
		ActivePending         int64      `json:"active_pending"`
		LatestMod             *time.Time `json:"latest_modification"`
		CursorLastMod         *time.Time `json:"cursor_last_modification_timestamp"`
		LastSyncFinished      *time.Time `json:"last_sync_finished_at"`
		IncrementalWindowEnd  *time.Time `json:"incremental_window_end"`
		ReplicationInProgress bool       `json:"replication_in_progress"`
	}
	var out []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.Dataset, &r.ActivePending, &r.LatestMod, &r.CursorLastMod,
			&r.LastSyncFinished, &r.IncrementalWindowEnd, &r.ReplicationInProgress); err != nil {
			return err
		}
		out = append(out, r)
	}
	if out == nil {
		out = []row{}
	}
	return c.JSON(fiber.Map{"datasets": out})
}
