package sync

import (
	"context"
	"log/slog"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// Kickoff dispatches replication fetch when datasets need catch-up.
type Kickoff struct {
	cfg       config.Config
	db        *repository.DB
	queue     *queue.Client
	store     *ReplicaPageStore
	cursors *CursorStore
	logger  *slog.Logger
}

func NewKickoff(cfg config.Config, db *repository.DB, q *queue.Client, logger *slog.Logger) *Kickoff {
	return &Kickoff{
		cfg:     cfg,
		db:      db,
		queue:   q,
		store:   NewReplicaPageStore(db, cfg),
		cursors: NewCursorStore(db),
		logger:  logger,
	}
}

func (k *Kickoff) Run(ctx context.Context) error {
	if err := k.dispatchBridge(ctx); err != nil {
		return err
	}
	return k.dispatchSpark(ctx)
}

// ResumeStalledReplication re-enqueues replication fetch when a chain is active but idle.
func (k *Kickoff) ResumeStalledReplication(ctx context.Context) error {
	if err := k.resumeProvider(ctx, "bridge"); err != nil {
		return err
	}
	return k.resumeProvider(ctx, "spark")
}

func (k *Kickoff) resumeProvider(ctx context.Context, provider string) error {
	switch provider {
	case "bridge":
		if !k.cfg.MLS.StellarEnabled {
			return nil
		}
		for _, ds := range k.cfg.Bridge.Datasets {
			if err := k.resumeDataset(ctx, provider, ds, k.cfg.Bridge.SyncFetchQueue, queue.TypeBridgeFetchPage); err != nil {
				return err
			}
		}
	case "spark":
		if !k.cfg.MLS.BeachesEnabled || k.cfg.Spark.AccessToken == "" {
			return nil
		}
		for _, ds := range k.cfg.Spark.Datasets {
			if err := k.resumeDataset(ctx, provider, ds, k.cfg.Spark.SyncFetchQueue, queue.TypeSparkFetchPage); err != nil {
				return err
			}
		}
	}
	return nil
}

func (k *Kickoff) resumeDataset(ctx context.Context, provider, dataset, fetchQueue, jobType string) error {
	cursor, err := k.cursors.ForDataset(ctx, dataset)
	if err != nil {
		return err
	}
	if !ReplicationChainActive(cursor) {
		return nil
	}

	stall := time.Duration(k.cfg.MLS.ReplicationResumeStallMinutes) * time.Minute
	if stall > 0 && time.Since(cursor.UpdatedAt) < stall {
		return nil
	}

	active, err := k.store.HasActivePage(ctx, provider, dataset)
	if err != nil || active {
		return err
	}

	pending, err := k.queue.HasPendingFetch(ctx, fetchQueue, jobType, dataset)
	if err != nil || pending {
		return err
	}

	id, err := k.queue.Enqueue(ctx, fetchQueue, jobType, fetchPageArgs{
		Dataset: dataset,
		Mode:    "replication",
	}, 0)
	if err != nil {
		return err
	}
	k.logger.Info("replication resume enqueued",
		"provider", provider, "dataset", dataset, "job_id", id)
	return nil
}

func (k *Kickoff) dispatchBridge(ctx context.Context) error {
	if !k.cfg.MLS.StellarEnabled {
		return nil
	}
	for _, ds := range k.cfg.Bridge.Datasets {
		if err := k.dispatchDataset(ctx, "bridge", ds, k.cfg.Bridge.SyncFetchQueue, queue.TypeBridgeFetchPage); err != nil {
			return err
		}
	}
	return nil
}

func (k *Kickoff) dispatchSpark(ctx context.Context) error {
	if !k.cfg.MLS.BeachesEnabled {
		return nil
	}
	if k.cfg.Spark.AccessToken == "" {
		k.logger.WarnContext(ctx, "spark replication kickoff skipped: SPARK_ACCESS_TOKEN not set")
		return nil
	}
	for _, ds := range k.cfg.Spark.Datasets {
		if err := k.dispatchDataset(ctx, "spark", ds, k.cfg.Spark.SyncFetchQueue, queue.TypeSparkFetchPage); err != nil {
			return err
		}
	}
	return nil
}

func (k *Kickoff) dispatchDataset(ctx context.Context, provider, dataset, fetchQueue, jobType string) error {
	active, err := k.store.HasActivePage(ctx, provider, dataset)
	if err != nil {
		return err
	}
	if active {
		k.logger.Debug("kickoff skipped: active replica page",
			"provider", provider, "dataset", dataset)
		return nil
	}

	cursor, err := k.cursors.ForDataset(ctx, dataset)
	if err != nil {
		return err
	}

	if ReplicationChainActive(cursor) {
		k.logger.Debug("kickoff skipped: replication chain active",
			"provider", provider, "dataset", dataset)
		return nil
	}

	runRep, err := k.cursors.ShouldKickoffReplication(ctx, cursor)
	if err != nil {
		return err
	}
	if runRep {
		return k.enqueueFetch(ctx, provider, dataset, fetchQueue, jobType, "replication")
	}

	return k.tryIncrementalKickoff(ctx, provider, dataset, fetchQueue, jobType, cursor)
}

func (k *Kickoff) tryIncrementalKickoff(ctx context.Context, provider, dataset, fetchQueue, jobType string, cursor SyncCursor) error {
	if ReplicationChainActive(cursor) {
		return nil
	}

	if !k.cursors.ShouldRunIncremental(cursor) {
		return nil
	}
	if !k.shouldPollIncremental(cursor) {
		return nil
	}

	pending, err := k.queue.HasPendingFetch(ctx, fetchQueue, jobType, dataset)
	if err != nil {
		return err
	}
	if pending {
		k.logger.Debug("kickoff skipped: fetch already queued",
			"provider", provider, "dataset", dataset)
		return nil
	}

	return k.enqueueFetch(ctx, provider, dataset, fetchQueue, jobType, "incremental")
}

// shouldPollIncremental gates steady-state Bridge/Spark updates after the mirror is seeded.
func (k *Kickoff) shouldPollIncremental(cursor SyncCursor) bool {
	if cursor.ReplicationInProgress {
		return false
	}
	if cursor.ReplicationNextURL != nil && *cursor.ReplicationNextURL != "" {
		return false
	}

	threshold := k.cfg.MLS.ReplicationFreshnessMinutes
	if threshold < 1 {
		threshold = 15
	}
	interval := time.Duration(threshold) * time.Minute

	if cursor.LastSyncFinishedAt == nil {
		return true
	}
	return time.Since(*cursor.LastSyncFinishedAt) >= interval
}

func (k *Kickoff) enqueueFetch(ctx context.Context, provider, dataset, fetchQueue, jobType, mode string) error {
	id, err := k.queue.Enqueue(ctx, fetchQueue, jobType, fetchPageArgs{
		Dataset: dataset,
		Mode:    mode,
	}, 0)
	if err != nil {
		k.logger.Error("enqueue fetch", "provider", provider, "dataset", dataset, "error", err)
		return err
	}
	k.logger.Info("enqueued fetch", "provider", provider, "dataset", dataset, "mode", mode, "queue", fetchQueue, "job_id", id)
	return nil
}
