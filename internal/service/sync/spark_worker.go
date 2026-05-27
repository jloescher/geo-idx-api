package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/fema"
	"github.com/quantyralabs/idx-api/internal/service/geocode"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// SparkWorker handles Spark replication queue jobs (BeachesMLS).
type SparkWorker struct {
	cfg        config.Config
	db         *repository.DB
	queue      *queue.Client
	store      *ReplicaPageStore
	mirror     *ListingMirrorWriter
	sync       *SparkSync
	cursors    *CursorStore
	femaEnrich    *fema.EnrichmentService
	geocodeEnrich *geocode.EnrichmentService
	logger        *slog.Logger
}

// SetFEMAEnrichment attaches the FEMA flood enrichment service (optional).
func (w *SparkWorker) SetFEMAEnrichment(s *fema.EnrichmentService) {
	w.femaEnrich = s
}

// SetGeocodeEnrichment attaches the geocode backfill service (optional).
func (w *SparkWorker) SetGeocodeEnrichment(s *geocode.EnrichmentService) {
	w.geocodeEnrich = s
}

func NewSparkWorker(cfg config.Config, db *repository.DB, q *queue.Client, logger *slog.Logger) *SparkWorker {
	upsertChunk := cfg.Spark.SyncUpsertChunk
	if upsertChunk <= 0 {
		upsertChunk = 250
	}
	return &SparkWorker{
		cfg:     cfg,
		db:      db,
		queue:   q,
		store:   NewReplicaPageStore(db, cfg),
		mirror:  NewListingMirrorWriter(db, upsertChunk, cfg.MLS.SyncExpand, cfg.Bridge.SyncExpand),
		sync:    NewSparkSync(cfg, db),
		cursors: NewCursorStore(db),
		logger:  logger,
	}
}

func (w *SparkWorker) FetchPage(ctx context.Context, job *queue.ReservedJob) error {
	var args fetchPageArgs
	if err := json.Unmarshal(job.Payload.Args, &args); err != nil {
		return err
	}
	if args.Dataset == "" {
		return fmt.Errorf("spark fetch: missing dataset")
	}

	maxChain := w.cfg.Spark.SyncMaxChainedFetch
	if maxChain > 0 && args.ChainDepth >= maxChain {
		w.logger.Warn("spark fetch chain cap", "dataset", args.Dataset, "depth", args.ChainDepth)
		return nil
	}

	cursor, err := w.cursors.ForDataset(ctx, args.Dataset)
	if err != nil {
		return err
	}

	if args.Mode == "incremental" && cursor.ReplicationInProgress {
		return nil
	}

	// Pin the upper bound before HTTP so queue retries use the same window (not a sliding "now").
	if args.Mode == "incremental" && cursor.IncrementalWindowEnd == nil {
		windowEnd := time.Now().UTC()
		if err := w.cursors.ApplyPatch(ctx, args.Dataset, CursorPatch{IncrementalWindowEnd: &windowEnd}); err != nil {
			return err
		}
		cursor.IncrementalWindowEnd = &windowEnd
	}

	if skip, err := skipReplicationFetchWhenPageActive(ctx, w.store, w.logger, "spark", args.Dataset, args.Mode); err != nil || skip {
		return err
	}

	var result PageResult
	switch args.Mode {
	case "replication":
		result, err = w.sync.FetchReplicationPage(ctx, cursor)
	default:
		result, err = w.sync.FetchIncrementalPage(ctx, cursor, args.IncrementalSkip)
	}
	if err != nil {
		if healed, healErr := maybeSelfHealReplicationFetch(ctx, w.cfg, w.queue, w.logger,
			"spark", args.Dataset, w.cfg.Spark.SyncFetchQueue, queue.TypeSparkFetchPage, args.Mode, cursor, err); healed {
			return healErr
		}
		return err
	}

	if result.Forbidden {
		inProgress := false
		return w.cursors.ApplyPatch(ctx, args.Dataset, CursorPatch{
			ApplyReplicationState: true,
			ReplicationNextURL:    nil,
			ReplicationInProgress: &inProgress,
		})
	}
	if result.HTTPError {
		w.logger.Error("spark fetch http error",
			"dataset", args.Dataset,
			"status", result.HTTPStatus,
			"mode", args.Mode,
			"odata_error", result.ODataError,
			"url", result.UpstreamURL,
		)
		fetchErr := fetchHTTPFailure("spark", result.HTTPStatus)
		if healed, err := maybeSelfHealReplicationFetch(ctx, w.cfg, w.queue, w.logger,
			"spark", args.Dataset, w.cfg.Spark.SyncFetchQueue, queue.TypeSparkFetchPage, args.Mode, cursor, fetchErr); healed {
			return err
		}
		if healed, err := maybeSelfHealIncrementalBadRequest(ctx, w.queue, w.cursors, w.logger,
			"spark", args.Dataset, w.cfg.Spark.SyncFetchQueue, queue.TypeSparkFetchPage, args.Mode, result.HTTPStatus); healed {
			return err
		}
		return fetchErr
	}

	if len(result.Rows) == 0 && args.Mode == "incremental" && !result.IncrementalHasMore {
		patch := CursorPatch{MarkSyncFinished: true}
		if result.IncrementalWindowEnd != nil {
			patch.IncrementalWindowEnd = result.IncrementalWindowEnd
		}
		if err := w.cursors.ApplyPatch(ctx, args.Dataset, patch); err != nil {
			return err
		}
		if w.femaEnrich != nil {
			if err := w.femaEnrich.EnqueueKickoffIfAbsent(ctx, ""); err != nil {
				w.logger.Warn("fema flood enrich kickoff", "error", err)
			}
		}
		if w.geocodeEnrich != nil {
			if err := w.geocodeEnrich.EnqueueKickoffIfAbsent(ctx, ""); err != nil {
				w.logger.Warn("geocode enrich kickoff", "error", err)
			}
		}
		return nil
	}

	patch, dispatchIncremental, nextFetch := w.continuationPlan(args, result)
	if args.Mode == "replication" && len(result.Rows) > 0 {
		inProgress := true
		if err := w.cursors.ApplyPatch(ctx, args.Dataset, CursorPatch{ReplicationInProgress: &inProgress}); err != nil {
			return err
		}
	}
	return w.dispatchPersistBatch(ctx, args.Dataset, args.Mode, result, patch, dispatchIncremental, nextFetch)
}

func (w *SparkWorker) continuationPlan(args fetchPageArgs, result PageResult) (CursorPatch, bool, *fetchPageArgs) {
	patch := CursorPatch{}
	if result.IncrementalWindowEnd != nil {
		patch.IncrementalWindowEnd = result.IncrementalWindowEnd
	}

	if args.Mode == "replication" {
		inProgress := !result.ReplicationComplete
		patch.ApplyReplicationState = true
		patch.ReplicationNextURL = result.NextReplicationURL
		patch.ReplicationInProgress = &inProgress
		patch.MaxModificationTs = result.MaxModificationTs

		if !result.ReplicationComplete && result.NextReplicationURL != nil && *result.NextReplicationURL != "" {
			mode := "replication"
			return patch, false, &fetchPageArgs{
				Dataset:    args.Dataset,
				Mode:       mode,
				ChainDepth: args.ChainDepth + 1,
			}
		}
		dispatchInc := result.ReplicationComplete && result.MaxModificationTs != nil
		return patch, dispatchInc, nil
	}

	patch.MaxModificationTs = result.MaxModificationTs
	patch.MarkSyncFinished = !result.IncrementalHasMore
	if result.IncrementalWindowEnd != nil {
		patch.IncrementalWindowEnd = result.IncrementalWindowEnd
	}

	if result.IncrementalHasMore {
		top := w.cfg.Spark.SyncIncrementalTop
		if top <= 0 {
			top = 1000
		}
		skip := args.IncrementalSkip + top
		if skip >= 10000 {
			w.logger.Warn("spark incremental skip cap", "dataset", args.Dataset, "skip", skip)
			return patch, false, nil
		}
		mode := "incremental"
		return patch, false, &fetchPageArgs{
			Dataset:         args.Dataset,
			Mode:            mode,
			IncrementalSkip: skip,
			ChainDepth:      args.ChainDepth + 1,
		}
	}
	return patch, false, nil
}

func (w *SparkWorker) dispatchPersistBatch(
	ctx context.Context,
	dataset, mode string,
	result PageResult,
	patch CursorPatch,
	dispatchIncremental bool,
	nextFetch *fetchPageArgs,
) error {
	persistQueue := w.cfg.Spark.SyncPersistQueue
	if len(result.Rows) == 0 {
		finalizeArgs := w.buildFinalizeArgs(dataset, nil, patch, dispatchIncremental, nextFetch)
		_, err := w.queue.Enqueue(ctx, persistQueue, queue.TypeSparkPersistFinalize, finalizeArgs, 0)
		return err
	}

	chunkSize := w.cfg.Spark.SyncPersistChunk
	if w.cfg.MLS.BeachesPersistChunk > 0 {
		chunkSize = w.cfg.MLS.BeachesPersistChunk
	}
	pageID, chunkTotal, err := w.store.StorePage(ctx, "spark", dataset, mode, result.Rows, chunkSize, replicaPageMetaFromResult(result))
	if err != nil {
		return err
	}
	w.logger.Info("stored replica page", "provider", "spark", "dataset", dataset, "page_id", pageID, "rows", len(result.Rows), "chunks", chunkTotal)

	finalizeArgs := w.buildFinalizeArgs(dataset, &pageID, patch, dispatchIncremental, nextFetch)

	if chunkTotal == 0 {
		_, err := w.queue.Enqueue(ctx, persistQueue, queue.TypeSparkPersistFinalize, finalizeArgs, 0)
		return err
	}

	chunkJobs := make([]queue.BatchJob, 0, chunkTotal)
	for i := 1; i <= chunkTotal; i++ {
		chunkJobs = append(chunkJobs, queue.BatchJob{
			Type: queue.TypeSparkPersistChunk,
			Args: persistChunkArgs{
				ReplicaPageID: pageID,
				ChunkIndex:    i,
				ChunkTotal:    chunkTotal,
				Dataset:       dataset,
			},
		})
	}

	batchID, err := w.queue.EnqueueBatch(ctx, queue.BatchSpec{
		Name:  "spark-replica-persist:" + dataset,
		Queue: persistQueue,
		Jobs:  chunkJobs,
		OnComplete: queue.BatchJob{
			Type: queue.TypeSparkPersistFinalize,
			Args: finalizeArgs,
		},
	})
	if err != nil {
		return err
	}
	return w.store.MarkProcessing(ctx, pageID, batchID)
}

func (w *SparkWorker) buildFinalizeArgs(
	dataset string,
	pageID *int64,
	patch CursorPatch,
	dispatchIncremental bool,
	nextFetch *fetchPageArgs,
) persistFinalizeArgs {
	args := persistFinalizeArgs{
		Dataset:                  dataset,
		ReplicaPageID:            pageID,
		DispatchIncrementalAfter: dispatchIncremental,
	}
	if patch.ApplyReplicationState {
		args.ApplyReplicationState = true
		args.ReplicationNextURL = patch.ReplicationNextURL
		args.ReplicationInProgress = patch.ReplicationInProgress
	}
	if patch.MaxModificationTs != nil {
		s := patch.MaxModificationTs.UTC().Format(time.RFC3339)
		args.MaxModificationTs = &s
	}
	if patch.IncrementalWindowEnd != nil {
		s := patch.IncrementalWindowEnd.UTC().Format(time.RFC3339)
		args.IncrementalWindowEnd = &s
	}
	if patch.MarkSyncFinished {
		args.MarkSyncFinished = true
	}
	if nextFetch != nil {
		args.NextFetchMode = &nextFetch.Mode
		args.NextIncrementalSkip = nextFetch.IncrementalSkip
		args.NextChainDepth = nextFetch.ChainDepth
	}
	return args
}

func (w *SparkWorker) PersistChunk(ctx context.Context, job *queue.ReservedJob) error {
	var wrapper struct {
		BatchID string           `json:"batch_id"`
		Job     persistChunkArgs `json:"job"`
	}
	if err := json.Unmarshal(job.Payload.Args, &wrapper); err != nil {
		return err
	}
	args := wrapper.Job
	if args.ReplicaPageID == 0 {
		return nil
	}
	rows, err := w.store.RowsForChunk(ctx, args.ReplicaPageID, args.ChunkIndex, args.ChunkTotal)
	if err != nil {
		return err
	}
	stats, err := w.mirror.HydrateReplicaBatch(ctx, args.Dataset, mls.MirrorProviderSpark, rows)
	if err != nil {
		return err
	}
	w.logger.Info("persisted mirror chunk",
		"provider", "spark",
		"dataset", args.Dataset,
		"page_id", args.ReplicaPageID,
		"chunk", args.ChunkIndex,
		"upserted", stats.Upserted,
		"deleted", stats.Deleted,
		"skipped", stats.Skipped,
	)
	return nil
}

func (w *SparkWorker) PersistFinalize(ctx context.Context, job *queue.ReservedJob) error {
	var args persistFinalizeArgs
	if err := json.Unmarshal(job.Payload.Args, &args); err != nil {
		return err
	}

	if args.ApplyReplicationState || args.MaxModificationTs != nil || args.IncrementalWindowEnd != nil || args.MarkSyncFinished || args.ReplicationInProgress != nil {
		patch := CursorPatch{
			ApplyReplicationState: args.ApplyReplicationState,
			ReplicationNextURL:    args.ReplicationNextURL,
			ReplicationInProgress: args.ReplicationInProgress,
			MarkSyncFinished:      args.MarkSyncFinished,
		}
		if args.MaxModificationTs != nil {
			if t, err := time.Parse(time.RFC3339, *args.MaxModificationTs); err == nil {
				patch.MaxModificationTs = &t
			}
		}
		if args.IncrementalWindowEnd != nil {
			if t, err := time.Parse(time.RFC3339, *args.IncrementalWindowEnd); err == nil {
				patch.IncrementalWindowEnd = &t
			}
		}
		if err := w.cursors.ApplyPatch(ctx, args.Dataset, patch); err != nil {
			return err
		}
	}

	if args.ReplicaPageID != nil {
		_ = w.store.MarkCompleted(ctx, *args.ReplicaPageID)
		_ = w.store.DeletePage(ctx, *args.ReplicaPageID)
	}

	fetchQueue := w.cfg.Spark.SyncFetchQueue
	if args.DispatchIncrementalAfter {
		_, err := w.queue.Enqueue(ctx, fetchQueue, queue.TypeSparkFetchPage, fetchPageArgs{
			Dataset: args.Dataset,
			Mode:    "incremental",
		}, 0)
		return err
	}
	if args.NextFetchMode != nil {
		_, err := w.queue.Enqueue(ctx, fetchQueue, queue.TypeSparkFetchPage, fetchPageArgs{
			Dataset:         args.Dataset,
			Mode:            *args.NextFetchMode,
			IncrementalSkip: args.NextIncrementalSkip,
			ChainDepth:      args.NextChainDepth,
		}, 0)
		return err
	}
	if args.MarkSyncFinished && w.femaEnrich != nil {
		if err := w.femaEnrich.EnqueueKickoffIfAbsent(ctx, ""); err != nil {
			w.logger.Warn("fema flood enrich kickoff", "error", err)
		}
	}
	if args.MarkSyncFinished && w.geocodeEnrich != nil {
		if err := w.geocodeEnrich.EnqueueKickoffIfAbsent(ctx, ""); err != nil {
			w.logger.Warn("geocode enrich kickoff", "error", err)
		}
	}
	return nil
}
