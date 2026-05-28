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

// BridgeWorker handles Bridge replication queue jobs.
type BridgeWorker struct {
	cfg        config.Config
	db         *repository.DB
	queue      *queue.Client
	store      *ReplicaPageStore
	mirror     *ListingMirrorWriter
	sync       *BridgeSync
	cursors    *CursorStore
	femaEnrich    *fema.EnrichmentService
	geocodeEnrich *geocode.EnrichmentService
	logger        *slog.Logger
}

// SetFEMAEnrichment attaches the FEMA flood enrichment service (optional).
func (w *BridgeWorker) SetFEMAEnrichment(s *fema.EnrichmentService) {
	w.femaEnrich = s
}

// SetGeocodeEnrichment attaches the geocode backfill service (optional).
func (w *BridgeWorker) SetGeocodeEnrichment(s *geocode.EnrichmentService) {
	w.geocodeEnrich = s
}

func NewBridgeWorker(cfg config.Config, db *repository.DB, q *queue.Client, logger *slog.Logger) *BridgeWorker {
	return &BridgeWorker{
		cfg:     cfg,
		db:      db,
		queue:   q,
		store:   NewReplicaPageStore(db, cfg),
		mirror:  NewListingMirrorWriter(db, cfg.Bridge.SyncUpsertChunk, cfg.MLS.SyncExpand, cfg.Bridge.SyncExpand),
		sync:    NewBridgeSync(cfg, db),
		cursors: NewCursorStore(db),
		logger:  logger,
	}
}

type fetchPageArgs struct {
	Dataset         string `json:"dataset"`
	Mode            string `json:"mode"`
	IncrementalSkip int    `json:"incremental_skip"`
	ChainDepth      int    `json:"chain_depth"`
}

type persistChunkArgs struct {
	ReplicaPageID int64  `json:"replica_page_id"`
	ChunkIndex    int    `json:"chunk_index"`
	ChunkTotal    int    `json:"chunk_total"`
	Dataset       string `json:"dataset"`
}

type persistFinalizeArgs struct {
	ReplicaPageID            *int64  `json:"replica_page_id,omitempty"`
	Dataset                  string  `json:"dataset"`
	ApplyReplicationState    bool    `json:"apply_replication_state,omitempty"`
	ReplicationNextURL       *string `json:"replication_next_url,omitempty"`
	ReplicationInProgress    *bool   `json:"replication_in_progress,omitempty"`
	MaxModificationTs        *string `json:"max_modification_ts,omitempty"`
	IncrementalWindowEnd     *string `json:"incremental_window_end,omitempty"`
	MarkSyncFinished         bool    `json:"mark_sync_finished,omitempty"`
	DispatchIncrementalAfter bool    `json:"dispatch_incremental_after,omitempty"`
	NextFetchMode            *string `json:"next_fetch_mode,omitempty"`
	NextIncrementalSkip      int     `json:"next_incremental_skip,omitempty"`
	NextChainDepth           int     `json:"next_chain_depth,omitempty"`
}

func (w *BridgeWorker) FetchPage(ctx context.Context, job *queue.ReservedJob) error {
	var args fetchPageArgs
	if err := json.Unmarshal(job.Payload.Args, &args); err != nil {
		return err
	}
	if args.Dataset == "" {
		return fmt.Errorf("bridge fetch: missing dataset")
	}

	maxChain := w.cfg.Bridge.SyncMaxChainedFetch
	if maxChain > 0 && args.ChainDepth >= maxChain {
		w.logger.Warn("bridge fetch chain cap", "dataset", args.Dataset, "depth", args.ChainDepth)
		return nil
	}

	cursor, err := w.cursors.ForDataset(ctx, args.Dataset)
	if err != nil {
		return err
	}

	if args.Mode == "incremental" && cursor.ReplicationInProgress {
		return nil
	}

	if skip, err := skipReplicationFetchWhenPageActive(ctx, w.store, w.logger, "bridge", args.Dataset, args.Mode); err != nil || skip {
		return err
	}

	var result PageResult
	switch args.Mode {
	case "replication":
		result, err = w.sync.FetchReplicationPage(ctx, args.Dataset, cursor)
	case "nav_hydrate":
		result, err = w.sync.FetchNavHydratePage(ctx, args.Dataset, args.IncrementalSkip)
	default:
		result, err = w.sync.FetchIncrementalPage(ctx, args.Dataset, cursor, args.IncrementalSkip)
	}
	if err != nil {
		if healed, healErr := maybeSelfHealReplicationFetch(ctx, w.cfg, w.queue, w.logger,
			"bridge", args.Dataset, w.cfg.Bridge.SyncFetchQueue, queue.TypeBridgeFetchPage, args.Mode, cursor, err); healed {
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
		w.logger.Error("bridge fetch http error", "dataset", args.Dataset, "status", result.HTTPStatus, "url", result.UpstreamURL)
		fetchErr := fetchHTTPFailure("bridge", result.HTTPStatus)
		if healed, err := maybeSelfHealReplicationFetch(ctx, w.cfg, w.queue, w.logger,
			"bridge", args.Dataset, w.cfg.Bridge.SyncFetchQueue, queue.TypeBridgeFetchPage, args.Mode, cursor, fetchErr); healed {
			return err
		}
		if healed, err := maybeSelfHealIncrementalBadRequest(ctx, w.queue, w.cursors, w.logger,
			"bridge", args.Dataset, w.cfg.Bridge.SyncFetchQueue, queue.TypeBridgeFetchPage, args.Mode, result.HTTPStatus); healed {
			return err
		}
		return fetchErr
	}

	if len(result.Rows) == 0 && (args.Mode == "incremental" || args.Mode == "nav_hydrate") && !result.IncrementalHasMore {
		if err := w.cursors.ApplyPatch(ctx, args.Dataset, CursorPatch{MarkSyncFinished: true}); err != nil {
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

func (w *BridgeWorker) continuationPlan(args fetchPageArgs, result PageResult) (CursorPatch, bool, *fetchPageArgs) {
	if args.Mode == "replication" {
		inProgress := !result.ReplicationComplete
		patch := CursorPatch{
			ApplyReplicationState: true,
			ReplicationNextURL:    result.NextReplicationURL,
			ReplicationInProgress: &inProgress,
			MaxModificationTs:     result.MaxModificationTs,
		}
		if !result.ReplicationComplete && result.NextReplicationURL != nil && *result.NextReplicationURL != "" {
			mode := "replication"
			return patch, false, &fetchPageArgs{
				Dataset:    args.Dataset,
				Mode:       mode,
				ChainDepth: args.ChainDepth + 1,
			}
		}
		if result.ReplicationComplete && w.cfg.Bridge.SyncNavHydrateAfterReplication {
			mode := "nav_hydrate"
			return patch, false, &fetchPageArgs{
				Dataset:    args.Dataset,
				Mode:       mode,
				ChainDepth: args.ChainDepth + 1,
			}
		}
		dispatchInc := result.ReplicationComplete && result.MaxModificationTs != nil
		return patch, dispatchInc, nil
	}

	if args.Mode == "nav_hydrate" {
		patch := CursorPatch{
			MaxModificationTs: result.MaxModificationTs,
		}
		if result.IncrementalHasMore {
			top := w.cfg.Bridge.SyncIncrementalTop
			if top <= 0 {
				top = 200
			}
			skip := args.IncrementalSkip + top
			if skip >= 10000 {
				w.logger.Warn("bridge nav hydrate skip cap", "dataset", args.Dataset, "skip", skip)
				return patch, result.MaxModificationTs != nil, nil
			}
			return patch, false, &fetchPageArgs{
				Dataset:         args.Dataset,
				Mode:            "nav_hydrate",
				IncrementalSkip: skip,
				ChainDepth:      args.ChainDepth + 1,
			}
		}
		return patch, result.MaxModificationTs != nil, nil
	}

	patch := CursorPatch{
		MaxModificationTs: result.MaxModificationTs,
		MarkSyncFinished:  !result.IncrementalHasMore,
	}
	if result.IncrementalHasMore {
		top := w.cfg.Bridge.SyncIncrementalTop
		if top <= 0 {
			top = 200
		}
		skip := args.IncrementalSkip + top
		if skip >= 10000 {
			w.logger.Warn("bridge incremental skip cap", "dataset", args.Dataset, "skip", skip)
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

func (w *BridgeWorker) dispatchPersistBatch(
	ctx context.Context,
	dataset, mode string,
	result PageResult,
	patch CursorPatch,
	dispatchIncremental bool,
	nextFetch *fetchPageArgs,
) error {
	persistQueue := w.cfg.Bridge.SyncPersistQueue
	if len(result.Rows) == 0 {
		finalizeArgs := w.buildFinalizeArgs(dataset, nil, patch, dispatchIncremental, nextFetch)
		_, err := w.queue.Enqueue(ctx, persistQueue, queue.TypeBridgePersistFinalize, finalizeArgs, 0)
		return err
	}

	chunkSize := w.cfg.Bridge.SyncPersistChunk
	pageID, chunkTotal, err := w.store.StorePage(ctx, "bridge", dataset, mode, result.Rows, chunkSize, replicaPageMetaFromResult(result))
	if err != nil {
		return err
	}
	w.logger.Info("stored replica page", "dataset", dataset, "page_id", pageID, "rows", len(result.Rows), "chunks", chunkTotal)

	finalizeArgs := w.buildFinalizeArgs(dataset, &pageID, patch, dispatchIncremental, nextFetch)

	if chunkTotal == 0 {
		_, err := w.queue.Enqueue(ctx, persistQueue, queue.TypeBridgePersistFinalize, finalizeArgs, 0)
		return err
	}

	chunkJobs := make([]queue.BatchJob, 0, chunkTotal)
	for i := 1; i <= chunkTotal; i++ {
		chunkJobs = append(chunkJobs, queue.BatchJob{
			Type: queue.TypeBridgePersistChunk,
			Args: persistChunkArgs{
				ReplicaPageID: pageID,
				ChunkIndex:    i,
				ChunkTotal:    chunkTotal,
				Dataset:       dataset,
			},
		})
	}

	batchID, err := w.queue.EnqueueBatch(ctx, queue.BatchSpec{
		Name:  "bridge-replica-persist:" + dataset,
		Queue: persistQueue,
		Jobs:  chunkJobs,
		OnComplete: queue.BatchJob{
			Type: queue.TypeBridgePersistFinalize,
			Args: finalizeArgs,
		},
	})
	if err != nil {
		return err
	}
	return w.store.MarkProcessing(ctx, pageID, batchID)
}

func replicaPageMetaFromResult(result PageResult) ReplicaPageMeta {
	return ReplicaPageMeta{
		FetchURL:    result.FetchURL,
		UpstreamURL: result.UpstreamURL,
		ODataQuery:  result.ODataQuery,
	}
}

func (w *BridgeWorker) buildFinalizeArgs(
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
	if patch.MarkSyncFinished {
		args.MarkSyncFinished = true
	}
	if patch.IncrementalWindowEnd != nil {
		s := patch.IncrementalWindowEnd.UTC().Format(time.RFC3339)
		args.IncrementalWindowEnd = &s
	}
	if nextFetch != nil {
		args.NextFetchMode = &nextFetch.Mode
		args.NextIncrementalSkip = nextFetch.IncrementalSkip
		args.NextChainDepth = nextFetch.ChainDepth
	}
	return args
}

func (w *BridgeWorker) PersistChunk(ctx context.Context, job *queue.ReservedJob) error {
	timeout := w.cfg.MLS.PersistChunkTimeout
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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
	stats, err := w.mirror.HydrateReplicaBatch(ctx, args.Dataset, mls.MirrorProviderBridge, rows)
	if err != nil {
		return err
	}
	w.logger.Info("persisted mirror chunk",
		"dataset", args.Dataset,
		"page_id", args.ReplicaPageID,
		"chunk", args.ChunkIndex,
		"upserted", stats.Upserted,
		"deleted", stats.Deleted,
		"skipped", stats.Skipped,
	)
	return nil
}

func (w *BridgeWorker) PersistFinalize(ctx context.Context, job *queue.ReservedJob) error {
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

	fetchQueue := w.cfg.Bridge.SyncFetchQueue
	if args.DispatchIncrementalAfter {
		_, err := w.queue.Enqueue(ctx, fetchQueue, queue.TypeBridgeFetchPage, fetchPageArgs{
			Dataset: args.Dataset,
			Mode:    "incremental",
		}, 0)
		return err
	}
	if args.NextFetchMode != nil {
		_, err := w.queue.Enqueue(ctx, fetchQueue, queue.TypeBridgeFetchPage, fetchPageArgs{
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
