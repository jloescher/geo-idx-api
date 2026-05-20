package sync

import (
	"context"
	"log/slog"

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
	mode, err := k.freshness.Mode(ctx, dataset, provider)
	if err != nil {
		return err
	}
	if mode != ModeCatchUp {
		return nil
	}

	active, err := k.store.HasActivePage(ctx, provider, dataset)
	if err != nil || active {
		return err
	}

	cursor, err := k.cursors.ForDataset(ctx, dataset)
	if err != nil {
		return err
	}

	fetchMode := "incremental"
	if runRep, err := k.cursors.ShouldRunReplication(ctx, cursor); err != nil {
		return err
	} else if runRep {
		fetchMode = "replication"
	} else if !k.cursors.ShouldRunIncremental(cursor) {
		return nil
	}

	id, err := k.queue.Enqueue(ctx, fetchQueue, jobType, fetchPageArgs{
		Dataset: dataset,
		Mode:    fetchMode,
	}, 0)
	if err != nil {
		k.logger.Error("enqueue fetch", "provider", provider, "dataset", dataset, "error", err)
		return err
	}
	k.logger.Info("enqueued fetch", "provider", provider, "dataset", dataset, "mode", fetchMode, "queue", fetchQueue, "job_id", id)
	return nil
}
