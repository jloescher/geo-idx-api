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
	cfg    config.Config
	db     *repository.DB
	queue  *queue.Client
	store  *ReplicaPageStore
	logger *slog.Logger
}

func NewKickoff(cfg config.Config, db *repository.DB, q *queue.Client, logger *slog.Logger) *Kickoff {
	return &Kickoff{cfg: cfg, db: db, queue: q, store: NewReplicaPageStore(db, cfg), logger: logger}
}

func (k *Kickoff) Run(ctx context.Context) error {
	for _, ds := range k.cfg.Bridge.Datasets {
		active, err := k.store.HasActivePage(ctx, "bridge", ds)
		if err != nil || active {
			continue
		}
		_, _ = k.queue.Enqueue(ctx, k.cfg.Bridge.SyncFetchQueue, queue.TypeBridgeFetchPage, fetchPageArgs{
			Dataset: ds,
			Mode:    "incremental",
		}, 0)
	}
	for _, ds := range k.cfg.Spark.Datasets {
		active, err := k.store.HasActivePage(ctx, "spark", ds)
		if err != nil || active {
			continue
		}
		_, _ = k.queue.Enqueue(ctx, k.cfg.Spark.SyncFetchQueue, queue.TypeSparkFetchPage, fetchPageArgs{
			Dataset: ds,
			Mode:    "incremental",
		}, 0)
	}
	return nil
}
