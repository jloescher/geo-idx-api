package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/robfig/cron/v3"
)

// Scheduler dispatches recurring jobs (Laravel routes/console.php parity).
type Scheduler struct {
	cfg    config.Config
	queue  *queue.Client
	db     *repository.DB
	logger *slog.Logger
	cron   *cron.Cron
	locks  sync.Map
}

func New(cfg config.Config, q *queue.Client, db *repository.DB, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		cfg:    cfg,
		queue:  q,
		db:     db,
		logger: logger,
		cron:   cron.New(cron.WithSeconds()),
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	_, _ = s.cron.AddFunc("0 */10 * * * *", s.withoutOverlap("coingecko", func() {
		_, _ = s.queue.Enqueue(ctx, s.cfg.Coingecko.Queue, queue.TypeCryptoRefreshPricing, nil, 0)
	}))
	_, _ = s.cron.AddFunc("0 */15 * * * *", s.withoutOverlap("mls-cache", func() {
		_, _ = s.queue.Enqueue(ctx, "default", queue.TypeMLSListingsCacheRefresh, nil, 0)
	}))
	_, _ = s.cron.AddFunc("0 * * * * *", s.withoutOverlap("mls-kickoff", func() {
		_, _ = s.queue.Enqueue(ctx, "default", queue.TypeMLSReplicationKickoff, nil, 0)
	}))
	_, _ = s.cron.AddFunc("0 15 4 * * *", s.withoutOverlap("purge-replica", func() {
		_, _ = s.queue.Enqueue(ctx, "default", queue.TypeMLSPurgeReplicaPages, nil, 0)
	}))
	_, _ = s.cron.AddFunc("0 5 3 * * *", s.withoutOverlap("purge-closed", func() {
		_, _ = s.queue.Enqueue(ctx, "default", queue.TypeMLSPurgeClosed, nil, 0)
	}))
	_, _ = s.cron.AddFunc("0 30 6 * * 1", s.withoutOverlap("gis-probe", func() {
		_, _ = s.queue.Enqueue(ctx, s.cfg.GIS.Queue, queue.TypeGISProbeSources, nil, 0)
	}))

	s.cron.Start()
	<-ctx.Done()
	stopCtx := s.cron.Stop()
	select {
	case <-stopCtx.Done():
	case <-time.After(10 * time.Second):
	}
	return ctx.Err()
}

func (s *Scheduler) withoutOverlap(name string, fn func()) func() {
	return func() {
		if _, loaded := s.locks.LoadOrStore(name, true); loaded {
			return
		}
		defer s.locks.Delete(name)
		fn()
	}
}
