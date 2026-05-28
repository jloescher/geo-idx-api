package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
	"github.com/quantyralabs/idx-api/internal/service/gis"
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
	lockKey := s.cfg.Scheduler.LeaderLockKey
	if lockKey == 0 {
		lockKey = DefaultLeaderLockKey
	}
	poll := s.cfg.Scheduler.StandbyPollInterval

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		leader, ok, err := TryAcquireLeader(ctx, s.db.Pool, lockKey)
		if err != nil {
			s.logger.Warn("scheduler leader acquire failed, retrying", "lock_key", lockKey, "error", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(poll):
				continue
			}
		}
		if !ok {
			s.logger.Info("scheduler standby, waiting for leader lock", "lock_key", lockKey)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(poll):
				continue
			}
		}

		s.logger.Info("scheduler leader acquired", "lock_key", lockKey)
		runErr := s.runAsLeader(ctx)
		leader.Release(ctx)

		if ctx.Err() != nil {
			return ctx.Err()
		}
		if runErr != nil {
			return runErr
		}
		s.logger.Warn("scheduler leader session ended, re-electing", "lock_key", lockKey)
	}
}

func (s *Scheduler) runAsLeader(ctx context.Context) error {
	s.addJob(ctx, "coingecko", "0 */10 * * * *", s.cfg.Coingecko.Queue, queue.TypeCryptoRefreshPricing)
	s.addJob(ctx, "mls-proxy-cache-purge", "0 */15 * * * *", "default", queue.TypeMLSProxyCachePurge)
	s.addJob(ctx, "mls-kickoff", "0 * * * * *", s.cfg.MLS.SyncKickoffQueue, queue.TypeMLSReplicationKickoff)
	s.addJob(ctx, "mls-replication-resume", s.cfg.MLS.ReplicationResumeCron, s.cfg.MLS.SyncKickoffQueue, queue.TypeMLSReplicationResume)
	s.addJob(ctx, "purge-replica", "0 15 4 * * *", "default", queue.TypeMLSPurgeReplicaPages)
	s.addJob(ctx, "purge-closed", "0 5 3 * * *", "default", queue.TypeMLSPurgeClosed)
	s.addJob(ctx, "mirror-key-reconcile", "0 0 4 * * *", s.cfg.MLS.SyncKickoffQueue, queue.TypeMLSMirrorKeyReconcile)
	s.addJob(ctx, "fema-flood-enrich", "0 30 4 * * *", s.cfg.FEMA.EnrichQueue, queue.TypeFEMAFloodEnrichKickoff)
	s.addJob(ctx, "mls-geocode-listings", "0 15 5 * * *", s.cfg.Geocode.EnrichQueue, queue.TypeMLSGeocodeListingsKickoff)
	s.addJob(ctx, "gis-probe", "0 30 6 * * 1", s.cfg.GIS.Queue, queue.TypeGISProbeSources)
	s.addJob(ctx, "gis-monthly-parcel-refresh", "0 0 2 1 * *", s.cfg.GIS.SyncQueue, queue.TypeGISMonthlyParcelRefresh)
	s.addJob(ctx, "gis-annual-boundaries-refresh", "0 0 3 1 1 *", s.cfg.GIS.SyncQueue, queue.TypeGISAnnualBoundariesRefresh)
	if _, err := s.cron.AddFunc("0 15 */6 * * *", s.withoutOverlap("gis-bootstrap-recheck", func() {
		s.enqueueGISSyncBootstrap(ctx)
	})); err != nil {
		s.logger.Error("cron register failed", "name", "gis-bootstrap-recheck", "error", err)
	}

	s.logger.Info("cron schedules registered",
		"mls_kickoff", "every minute at :00",
		"mls_replication_resume", s.cfg.MLS.ReplicationResumeCron,
		"mirror_key_reconcile", "daily 04:00 UTC",
		"mls_proxy_cache_purge", "every 15 min",
		"coingecko", "every 10 min",
		"gis_monthly_parcel_refresh", "1st of month 02:00",
		"gis_annual_boundaries_refresh", "Jan 1 03:00",
		"gis_bootstrap_recheck", "every 6 hours at :15",
	)

	// First tick is up to ~60s away; enqueue kickoff once so dev workers show activity immediately.
	s.enqueue(ctx, "mls-kickoff-startup", s.cfg.MLS.SyncKickoffQueue, queue.TypeMLSReplicationKickoff, nil)
	s.enqueueGISSyncBootstrap(ctx)

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

func (s *Scheduler) enqueueGISSyncBootstrap(ctx context.Context) {
	repo := gisrepo.New(s.db)
	counts, err := repo.LoadLayerCounts(ctx)
	if err != nil {
		s.logger.Warn("gis bootstrap check failed", "error", err)
		return
	}
	if !counts.NeedsBootstrap() {
		return
	}
	queueName := s.cfg.GIS.SyncQueue
	if queueName == "" {
		queueName = "default"
	}
	for _, action := range gis.PlanBootstrapActions(counts) {
		s.enqueue(ctx, action.Name, queueName, action.JobType, nil)
	}
}
