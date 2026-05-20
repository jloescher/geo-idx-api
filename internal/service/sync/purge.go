package sync

import (
	"context"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// PurgeClosed removes closed listings outside rolling window from mirror.
type PurgeClosed struct {
	cfg config.Config
	db  *repository.DB
}

func NewPurgeClosed(cfg config.Config, db *repository.DB) *PurgeClosed {
	return &PurgeClosed{cfg: cfg, db: db}
}

func (p *PurgeClosed) Run(ctx context.Context) error {
	cutoff := time.Now().AddDate(0, -p.cfg.MLS.LocalMirrorRollingMonths, 0)
	_, err := p.db.Pool.Exec(ctx, `
		DELETE FROM listings
		WHERE LOWER(TRIM(COALESCE(standard_status, ''))) = 'closed'
		   OR (close_date IS NOT NULL AND close_date < $1::date)
	`, cutoff)
	return err
}
