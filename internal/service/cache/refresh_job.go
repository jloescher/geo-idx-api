package cache

import (
	"context"
	"log/slog"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/comps"
)

// RefreshJob purges stale mls_search_cache rows (on-demand proxy cache retention).
// Active/Pending data lives in the listings mirror; this job does not pre-warm collections.
type RefreshJob struct {
	cfg    config.Config
	db     *repository.DB
	cache  *ProxyCache
	logger *slog.Logger
}

func NewRefreshJob(cfg config.Config, db *repository.DB, logger *slog.Logger) *RefreshJob {
	return &RefreshJob{
		cfg:    cfg,
		db:     db,
		cache:  NewProxyCache(cfg, db),
		logger: logger,
	}
}

func (j *RefreshJob) Run(ctx context.Context, _ *queue.ReservedJob) error {
	n, err := j.cache.PurgeExpired(ctx)
	if err != nil {
		return err
	}
	j.logger.Info("mls proxy cache purge", "job", "mls.proxy_cache_purge", "deleted", n)

	closedDays := j.cfg.Comps.ClosedCacheDays
	if closedDays <= 0 {
		closedDays = 30
	}
	nClosed, err := comps.PurgeClosedListingsCache(ctx, j.db, closedDays)
	if err != nil {
		return err
	}
	j.logger.Info("comps closed listings cache purge", "job", "mls.proxy_cache_purge", "deleted", nClosed)
	return nil
}
