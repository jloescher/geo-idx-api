package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mlsupstream"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
)

const (
	reconcileMinKeysRatio     = 0.5
	reconcileMinKeysFloor     = 50
	reconcileStagingRetention = 7 * 24 * time.Hour
)

// reconcileKeysArgs is the payload for bridge.reconcile_keys / spark.reconcile_keys jobs.
type reconcileKeysArgs struct {
	RunID          string  `json:"run_id"`
	Dataset        string  `json:"dataset"`
	NextURL        *string `json:"next_url,omitempty"`
	UseFallback    bool    `json:"use_fallback,omitempty"`
	CollectionSkip int     `json:"collection_skip,omitempty"`
}

// KeyReconcile orchestrates nightly listing-key reconciliation against upstream AP catalogs.
type KeyReconcile struct {
	cfg        config.Config
	db         *repository.DB
	queue      *queue.Client
	cursors    *CursorStore
	replica    *ReplicaPageStore
	keyStore   *ReconcileKeyStore
	bridgeSync *BridgeSync
	sparkSync  *SparkSync
	logger     *slog.Logger
}

func NewKeyReconcile(cfg config.Config, db *repository.DB, q *queue.Client, logger *slog.Logger) *KeyReconcile {
	return &KeyReconcile{
		cfg:        cfg,
		db:         db,
		queue:      q,
		cursors:    NewCursorStore(db),
		replica:    NewReplicaPageStore(db, cfg),
		keyStore:   NewReconcileKeyStore(db),
		bridgeSync: NewBridgeSync(cfg, db),
		sparkSync:  NewSparkSync(cfg, db),
		logger:     logger,
	}
}

// Kickoff enqueues reconcile key sweeps for enabled datasets (skips when replication is active).
func (k *KeyReconcile) Kickoff(ctx context.Context) error {
	deferred := false
	if err := k.kickoffBridge(ctx, &deferred); err != nil {
		return err
	}
	if err := k.kickoffSpark(ctx, &deferred); err != nil {
		return err
	}
	if deferred {
		return k.scheduleDeferredKickoffRetry(ctx)
	}
	n, err := k.keyStore.PurgeStaleStaging(ctx, reconcileStagingRetention)
	if err != nil {
		return err
	}
	if n > 0 {
		k.logger.Info("purged stale reconcile staging keys", "rows", n)
	}
	return nil
}

func (k *KeyReconcile) kickoffBridge(ctx context.Context, deferred *bool) error {
	if !k.cfg.MLS.StellarEnabled {
		return nil
	}
	for _, ds := range k.cfg.Bridge.Datasets {
		if err := k.enqueueDataset(ctx, "bridge", ds, k.cfg.Bridge.SyncFetchQueue, queue.TypeBridgeReconcileKeys, queue.TypeBridgeFetchPage, deferred); err != nil {
			return err
		}
	}
	return nil
}

func (k *KeyReconcile) kickoffSpark(ctx context.Context, deferred *bool) error {
	if !k.cfg.MLS.BeachesEnabled || k.cfg.Spark.AccessToken == "" {
		return nil
	}
	for _, ds := range k.cfg.Spark.Datasets {
		if err := k.enqueueDataset(ctx, "spark", ds, k.cfg.Spark.SyncFetchQueue, queue.TypeSparkReconcileKeys, queue.TypeSparkFetchPage, deferred); err != nil {
			return err
		}
	}
	return nil
}

func (k *KeyReconcile) enqueueDataset(ctx context.Context, provider, dataset, fetchQueue, reconcileJobType, syncFetchJobType string, deferred *bool) error {
	if ok, err := k.syncFetchBlocksReconcile(ctx, provider, dataset, fetchQueue, syncFetchJobType); err != nil {
		return err
	} else if ok {
		k.logger.Info("mirror key reconcile deferred: sync fetch active",
			"provider", provider, "dataset", dataset)
		*deferred = true
		return nil
	}

	active, err := k.replica.HasActivePage(ctx, provider, dataset)
	if err != nil {
		return err
	}
	if active {
		k.logger.Info("mirror key reconcile deferred: active replica page",
			"provider", provider, "dataset", dataset)
		*deferred = true
		return nil
	}

	cursor, err := k.cursors.ForDataset(ctx, dataset)
	if err != nil {
		return err
	}
	if ReplicationChainActive(cursor) {
		k.logger.Info("mirror key reconcile deferred: replication chain active",
			"provider", provider, "dataset", dataset)
		*deferred = true
		return nil
	}

	pending, err := k.queue.HasPendingFetch(ctx, fetchQueue, reconcileJobType, dataset)
	if err != nil {
		return err
	}
	if pending {
		k.logger.Debug("mirror key reconcile skipped: job already queued",
			"provider", provider, "dataset", dataset)
		return nil
	}

	runID := uuid.New()
	mirrorCount, err := k.keyStore.CountMirrorListings(ctx, dataset)
	if err != nil {
		return err
	}
	if err := k.keyStore.RecordRunStart(ctx, runID, dataset, provider, mirrorCount); err != nil {
		return err
	}

	id, err := k.queue.Enqueue(ctx, fetchQueue, reconcileJobType, reconcileKeysArgs{
		RunID:   runID.String(),
		Dataset: dataset,
	}, 0)
	if err != nil {
		return err
	}
	k.logger.Info("enqueued mirror key reconcile",
		"provider", provider, "dataset", dataset, "run_id", runID, "job_id", id, "mirror_count", mirrorCount)
	return nil
}

func (k *KeyReconcile) scheduleDeferredKickoffRetry(ctx context.Context) error {
	kickoffQueue := k.cfg.MLS.SyncKickoffQueue
	if kickoffQueue == "" {
		kickoffQueue = "sync-kickoff"
	}
	pending, err := k.queue.HasPendingJobType(ctx, kickoffQueue, queue.TypeMLSMirrorKeyReconcile)
	if err != nil {
		return err
	}
	if pending {
		return nil
	}
	retryMin := k.cfg.MLS.MirrorKeyReconcileRetryMinutes
	if retryMin < 1 {
		retryMin = 30
	}
	delay := time.Duration(retryMin) * time.Minute
	id, err := k.queue.Enqueue(ctx, kickoffQueue, queue.TypeMLSMirrorKeyReconcile, struct{}{}, delay)
	if err != nil {
		return err
	}
	k.logger.Info("scheduled deferred mirror key reconcile kickoff",
		"queue", kickoffQueue, "delay_minutes", retryMin, "job_id", id)
	return nil
}

func (k *KeyReconcile) syncFetchBlocksReconcile(ctx context.Context, provider, dataset, fetchQueue, syncFetchJobType string) (bool, error) {
	pending, err := k.queue.HasPendingFetch(ctx, fetchQueue, syncFetchJobType, dataset)
	if err != nil {
		return false, err
	}
	if pending {
		return true, nil
	}
	active, err := k.replica.HasActivePage(ctx, provider, dataset)
	if err != nil {
		return false, err
	}
	return active, nil
}

func (k *KeyReconcile) RunBridgePage(ctx context.Context, job *queue.ReservedJob) error {
	var args reconcileKeysArgs
	if err := json.Unmarshal(job.Payload.Args, &args); err != nil {
		return err
	}
	return k.runPage(ctx, "bridge", args, k.cfg.Bridge.SyncFetchQueue, queue.TypeBridgeFetchPage, func(ctx context.Context) (KeyPageResult, error) {
		return k.bridgeSync.FetchReconcileKeysPage(ctx, args.Dataset, args.NextURL, args.UseFallback, args.CollectionSkip)
	})
}

func (k *KeyReconcile) RunSparkPage(ctx context.Context, job *queue.ReservedJob) error {
	var args reconcileKeysArgs
	if err := json.Unmarshal(job.Payload.Args, &args); err != nil {
		return err
	}
	return k.runPage(ctx, "spark", args, k.cfg.Spark.SyncFetchQueue, queue.TypeSparkFetchPage, func(ctx context.Context) (KeyPageResult, error) {
		return k.sparkSync.FetchReconcileKeysPage(ctx, args.Dataset, args.NextURL)
	})
}

func (k *KeyReconcile) runPage(ctx context.Context, provider string, args reconcileKeysArgs, fetchQueue, syncFetchJobType string, fetch func(context.Context) (KeyPageResult, error)) error {
	if args.Dataset == "" || args.RunID == "" {
		return fmt.Errorf("reconcile keys: missing dataset or run_id")
	}
	runID, err := uuid.Parse(args.RunID)
	if err != nil {
		return fmt.Errorf("reconcile keys: invalid run_id: %w", err)
	}

	tx, err := k.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	ok, err := TryAcquireReconcileRunLock(ctx, tx, runID)
	if err != nil {
		return err
	}
	if !ok {
		return queue.ErrReconcileBusy{RunID: args.RunID}
	}

	start := time.Now()
	page, err := fetch(ctx)
	if err != nil {
		return err
	}

	if page.Forbidden {
		msg := fmt.Sprintf("upstream forbidden (HTTP %d)", page.HTTPStatus)
		_ = k.keyStore.RecordRunFailed(ctx, runID, msg)
		return fmt.Errorf("reconcile keys: %s", msg)
	}

	if page.HTTPError {
		if provider == "bridge" && !args.UseFallback {
			k.logger.Warn("bridge reconcile replication failed; retrying on /Property collection",
				"dataset", args.Dataset, "status", page.HTTPStatus, "error", page.ODataError)
			if err := tx.Commit(ctx); err != nil {
				return err
			}
			return k.enqueueNextPage(ctx, provider, args, reconcileKeysArgs{
				RunID:       args.RunID,
				Dataset:     args.Dataset,
				UseFallback: true,
			})
		}
		if page.HTTPStatus == 429 || page.HTTPStatus == 503 {
			return mlsupstream.ErrRateLimited{Provider: provider, Status: page.HTTPStatus}
		}
		msg := fmt.Sprintf("upstream HTTP %d: %s", page.HTTPStatus, page.ODataError)
		_ = k.keyStore.RecordRunFailed(ctx, runID, msg)
		return fmt.Errorf("reconcile keys: %s", msg)
	}

	if err := k.keyStore.InsertKeys(ctx, runID, args.Dataset, page.Keys); err != nil {
		return err
	}

	if !page.Complete {
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		next := reconcileKeysArgs{
			RunID:          args.RunID,
			Dataset:        args.Dataset,
			NextURL:        page.NextURL,
			UseFallback:    page.UseFallback,
			CollectionSkip: page.CollectionSkip,
		}
		return k.enqueueNextPage(ctx, provider, args, next)
	}

	if blocked, err := k.syncFetchBlocksReconcile(ctx, provider, args.Dataset, fetchQueue, syncFetchJobType); err != nil {
		return err
	} else if blocked {
		return fmt.Errorf("reconcile keys: sync fetch active for %s, refusing delete", args.Dataset)
	}

	cursor, err := k.cursors.ForDataset(ctx, args.Dataset)
	if err != nil {
		return err
	}
	if ReplicationChainActive(cursor) {
		return fmt.Errorf("reconcile keys: replication chain active for %s, refusing delete", args.Dataset)
	}

	keysSeen, err := k.keyStore.CountKeysTx(ctx, tx, runID, args.Dataset)
	if err != nil {
		return err
	}
	mirrorCount, err := k.keyStore.CountMirrorListings(ctx, args.Dataset)
	if err != nil {
		return err
	}
	if err := validateReconcileDelete(keysSeen, mirrorCount); err != nil {
		_ = k.keyStore.RecordRunFailed(ctx, runID, err.Error())
		return err
	}

	deleted, err := k.keyStore.DeleteStaleMirrorRowsTx(ctx, tx, runID, args.Dataset)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if err := k.keyStore.RecordRunComplete(ctx, runID, keysSeen, deleted); err != nil {
		return err
	}
	if err := k.keyStore.PurgeRun(ctx, runID); err != nil {
		return err
	}

	k.logger.Info("mirror key reconcile complete",
		"provider", provider,
		"dataset", args.Dataset,
		"run_id", runID,
		"keys_seen", keysSeen,
		"mirror_count", mirrorCount,
		"rows_deleted", deleted,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

func validateReconcileDelete(keysSeen, mirrorCount int64) error {
	if mirrorCount == 0 {
		return nil
	}
	if keysSeen == 0 {
		return fmt.Errorf("refuse delete: 0 upstream keys with %d mirror rows", mirrorCount)
	}
	if mirrorCount > reconcileMinKeysFloor {
		minKeys := int64(float64(mirrorCount) * reconcileMinKeysRatio)
		if keysSeen < minKeys {
			return fmt.Errorf("refuse delete: keys_seen %d < 50%% of mirror rows %d", keysSeen, mirrorCount)
		}
	}
	return nil
}

func (k *KeyReconcile) enqueueNextPage(ctx context.Context, provider string, current reconcileKeysArgs, next reconcileKeysArgs) error {
	fetchQueue := k.cfg.Bridge.SyncFetchQueue
	jobType := queue.TypeBridgeReconcileKeys
	if provider == "spark" {
		fetchQueue = k.cfg.Spark.SyncFetchQueue
		jobType = queue.TypeSparkReconcileKeys
	}
	id, err := k.queue.Enqueue(ctx, fetchQueue, jobType, next, 0)
	if err != nil {
		return err
	}
	k.logger.Debug("enqueued reconcile keys page",
		"provider", provider, "dataset", current.Dataset, "run_id", current.RunID, "job_id", id)
	return nil
}
