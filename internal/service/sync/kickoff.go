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
	cursors   *CursorStore
	freshness *Freshness
	logger    *slog.Logger
}

func NewKickoff(cfg config.Config, db *repository.DB, q *queue.Client, logger *slog.Logger) *Kickoff {
	return &Kickoff{
		cfg:       cfg,
		db:        db,
		queue:     q,
		store:     NewReplicaPageStore(db, cfg),
		cursors:   NewCursorStore(db),
		freshness: NewFreshness(cfg, db),
		logger:    logger,
	}
}

func (k *Kickoff) Run(ctx context.Context) error {
	if err := k.dispatchBridge(ctx); err != nil {
		return err
	}
	return k.dispatchSpark(ctx)
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
	if err != nil || active {
		return err
	}

	cursor, err := k.cursors.ForDataset(ctx, dataset)
	if err != nil {
		return err
	}

	if runRep, err := k.cursors.ShouldRunReplication(ctx, cursor); err != nil {
		return err
	} else if runRep {
		return k.enqueueFetch(ctx, provider, dataset, fetchQueue, jobType, "replication")
	}

	if !k.cursors.ShouldRunIncremental(cursor) {
		return nil
	}
	if !k.shouldPollIncremental(cursor) {
		return nil
	}

	return k.enqueueFetch(ctx, provider, dataset, fetchQueue, jobType, "incremental")
}

// shouldPollIncremental gates steady-state Bridge/Spark updates after the mirror is seeded.
// Bridge docs: poll Property with BridgeModificationTimestamp gt cursor on an interval.
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
