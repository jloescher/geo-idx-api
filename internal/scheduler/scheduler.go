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
	s.addJob(ctx, "coingecko", "0 */10 * * * *", s.cfg.Coingecko.Queue, queue.TypeCryptoRefreshPricing)
	s.addJob(ctx, "mls-cache", "0 */15 * * * *", "default", queue.TypeMLSListingsCacheRefresh)
	s.addJob(ctx, "mls-kickoff", "0 * * * * *", "default", queue.TypeMLSReplicationKickoff)
	s.addJob(ctx, "purge-replica", "0 15 4 * * *", "default", queue.TypeMLSPurgeReplicaPages)
	s.addJob(ctx, "purge-closed", "0 5 3 * * *", "default", queue.TypeMLSPurgeClosed)
	s.addJob(ctx, "gis-probe", "0 30 6 * * 1", s.cfg.GIS.Queue, queue.TypeGISProbeSources)

	s.logger.Info("cron schedules registered",
		"mls_kickoff", "every minute at :00",
		"mls_search_cache_purge", "every 15 min",
		"coingecko", "every 10 min",
	)

	// First tick is up to ~60s away; enqueue kickoff once so dev workers show activity immediately.
	s.enqueue(ctx, "mls-kickoff-startup", "default", queue.TypeMLSReplicationKickoff, nil)

	s.cron.Start()
	<-ctx.Done()
	stopCtx := s.cron.Stop()
	select {
	case <-stopCtx.Done():
	case <-time.After(10 * time.Second):
	}
	return ctx.Err()
}

func (s *Scheduler) addJob(ctx context.Context, name, spec, queueName, jobType string) {
	_, err := s.cron.AddFunc(spec, s.withoutOverlap(name, func() {
		s.enqueue(ctx, name, queueName, jobType, nil)
	}))
	if err != nil {
		s.logger.Error("cron register failed", "name", name, "spec", spec, "error", err)
	}
}

func (s *Scheduler) enqueue(ctx context.Context, name, queueName, jobType string, args any) {
	id, err := s.queue.Enqueue(ctx, queueName, jobType, args, 0)
	if err != nil {
		s.logger.Error("enqueue failed", "task", name, "type", jobType, "queue", queueName, "error", err)
		return
	}
	s.logger.Info("enqueued scheduled job", "task", name, "type", jobType, "queue", queueName, "job_id", id)
}

func (s *Scheduler) withoutOverlap(name string, fn func()) func() {
	return func() {
		if _, loaded := s.locks.LoadOrStore(name, true); loaded {
			s.logger.Debug("skipped overlapping run", "task", name)
			return
		}
		defer s.locks.Delete(name)
		fn()
	}
}
