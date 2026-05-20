package sync

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// BridgeWorker handles Bridge replication queue jobs.
type BridgeWorker struct {
	cfg    config.Config
	db     *repository.DB
	queue  *queue.Client
	store  *ReplicaPageStore
	mirror *ListingMirrorWriter
	logger *slog.Logger
}

func NewBridgeWorker(cfg config.Config, db *repository.DB, q *queue.Client, logger *slog.Logger) *BridgeWorker {
	return &BridgeWorker{
		cfg:    cfg,
		db:     db,
		queue:  q,
		store:  NewReplicaPageStore(db, cfg),
		mirror: NewListingMirrorWriter(db),
		logger: logger,
	}
}

type fetchPageArgs struct {
	Dataset         string `json:"dataset"`
	Mode            string `json:"mode"`
	IncrementalSkip int    `json:"incremental_skip"`
}

type persistChunkArgs struct {
	ReplicaPageID int64  `json:"replica_page_id"`
	ChunkIndex    int    `json:"chunk_index"`
	ChunkTotal    int    `json:"chunk_total"`
	Dataset       string `json:"dataset"`
}

type persistFinalizeArgs struct {
	ReplicaPageID *int64 `json:"replica_page_id"`
	Dataset       string `json:"dataset"`
}

func (w *BridgeWorker) FetchPage(ctx context.Context, job *queue.ReservedJob) error {
	var args fetchPageArgs
	_ = json.Unmarshal(job.Payload.Args, &args)
	w.logger.Info("bridge fetch page", "dataset", args.Dataset, "mode", args.Mode)
	return nil
}

func (w *BridgeWorker) PersistChunk(ctx context.Context, job *queue.ReservedJob) error {
	var wrapper struct {
		BatchID string           `json:"batch_id"`
		Job     persistChunkArgs `json:"job"`
	}
	_ = json.Unmarshal(job.Payload.Args, &wrapper)
	args := wrapper.Job
	if args.ReplicaPageID == 0 {
		return nil
	}
	rows, err := w.store.RowsForChunk(ctx, args.ReplicaPageID, args.ChunkIndex, args.ChunkTotal)
	if err != nil {
		return err
	}
	return w.mirror.UpsertBatch(ctx, args.Dataset, rows)
}

func (w *BridgeWorker) PersistFinalize(ctx context.Context, job *queue.ReservedJob) error {
	var args persistFinalizeArgs
	_ = json.Unmarshal(job.Payload.Args, &args)
	if args.ReplicaPageID != nil {
		_ = w.store.MarkCompleted(ctx, *args.ReplicaPageID)
		_ = w.store.DeletePage(ctx, *args.ReplicaPageID)
	}
	return nil
}
