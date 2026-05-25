package sync

import (
	"context"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// activePendingMirrorAgeExpr is the effective timestamp for rolling-window purge of Active/Pending rows.
const activePendingMirrorAgeExpr = "COALESCE(mirror_persisted_at, modification_timestamp)"

// PurgeClosed removes Closed listings from the mirror and, when a rolling window is configured,
// Active/Pending rows older than the window (by mirror persist time, not MLS modification alone).
type PurgeClosed struct {
	cfg config.Config
	db  *repository.DB
}

func NewPurgeClosed(cfg config.Config, db *repository.DB) *PurgeClosed {
	return &PurgeClosed{cfg: cfg, db: db}
}

func (p *PurgeClosed) Run(ctx context.Context) error {
	months := p.cfg.MLS.LocalMirrorRollingMonths
	if months <= 0 {
		_, err := p.db.Pool.Exec(ctx, `
			DELETE FROM listings
			WHERE LOWER(TRIM(COALESCE(standard_status, ''))) = 'closed'
		`)
		return err
	}

	cutoff := time.Now().UTC().AddDate(0, -months, 0)
	_, err := p.db.Pool.Exec(ctx, `
		DELETE FROM listings
		WHERE LOWER(TRIM(COALESCE(standard_status, ''))) = 'closed'
		   OR (
		     LOWER(TRIM(COALESCE(standard_status, ''))) IN ('active', 'pending')
		     AND `+activePendingMirrorAgeExpr+` < $1
		   )
		   OR (close_date IS NOT NULL AND close_date < $1::date)
	`, cutoff)
	return err
}
