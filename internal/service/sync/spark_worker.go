package sync

import (
	"context"
	"log/slog"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// SparkWorker handles Spark replication jobs (sequential persist chain for beaches).
type SparkWorker struct {
	bridge *BridgeWorker
}

func NewSparkWorker(cfg config.Config, db *repository.DB, q *queue.Client, logger *slog.Logger) *SparkWorker {
	return &SparkWorker{bridge: NewBridgeWorker(cfg, db, q, logger)}
}

func (w *SparkWorker) FetchPage(ctx context.Context, job *queue.ReservedJob) error {
	return w.bridge.FetchPage(ctx, job)
}

func (w *SparkWorker) PersistChunk(ctx context.Context, job *queue.ReservedJob) error {
	return w.bridge.PersistChunk(ctx, job)
}

func (w *SparkWorker) PersistFinalize(ctx context.Context, job *queue.ReservedJob) error {
	return w.bridge.PersistFinalize(ctx, job)
}
