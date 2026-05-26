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
	if err := k.kickoffBridge(ctx); err != nil {
		return err
	}
	return k.kickoffSpark(ctx)
}

func (k *KeyReconcile) kickoffBridge(ctx context.Context) error {
	if !k.cfg.MLS.StellarEnabled {
		return nil
	}
	for _, ds := range k.cfg.Bridge.Datasets {
		if err := k.enqueueDataset(ctx, "bridge", ds, k.cfg.Bridge.SyncFetchQueue, queue.TypeBridgeReconcileKeys); err != nil {
			return err
		}
	}
	return nil
}

func (k *KeyReconcile) kickoffSpark(ctx context.Context) error {
	if !k.cfg.MLS.BeachesEnabled || k.cfg.Spark.AccessToken == "" {
		return nil
	}
	for _, ds := range k.cfg.Spark.Datasets {
		if err := k.enqueueDataset(ctx, "spark", ds, k.cfg.Spark.SyncFetchQueue, queue.TypeSparkReconcileKeys); err != nil {
			return err
		}
	}
	return nil
}

func (k *KeyReconcile) enqueueDataset(ctx context.Context, provider, dataset, fetchQueue, jobType string) error {
	active, err := k.replica.HasActivePage(ctx, provider, dataset)
	if err != nil {
		return err
	}
	if active {
		k.logger.Info("mirror key reconcile deferred: active replica page",
			"provider", provider, "dataset", dataset)
		return nil
	}

	cursor, err := k.cursors.ForDataset(ctx, dataset)
	if err != nil {
		return err
	}
	if ReplicationChainActive(cursor) {
		k.logger.Info("mirror key reconcile deferred: replication chain active",
			"provider", provider, "dataset", dataset)
		return nil
	}

	pending, err := k.queue.HasPendingFetch(ctx, fetchQueue, jobType, dataset)
	if err != nil {
		return err
	}
	if pending {
		k.logger.Debug("mirror key reconcile skipped: job already queued",
			"provider", provider, "dataset", dataset)
		return nil
	}

	runID := uuid.New()
	id, err := k.queue.Enqueue(ctx, fetchQueue, jobType, reconcileKeysArgs{
		RunID:   runID.String(),
		Dataset: dataset,
	}, 0)
	if err != nil {
		return err
	}
	k.logger.Info("enqueued mirror key reconcile",
		"provider", provider, "dataset", dataset, "run_id", runID, "job_id", id)
	return nil
}

func (k *KeyReconcile) RunBridgePage(ctx context.Context, job *queue.ReservedJob) error {
	var args reconcileKeysArgs
	if err := json.Unmarshal(job.Payload.Args, &args); err != nil {
		return err
	}
	return k.runPage(ctx, "bridge", args, func(ctx context.Context) (KeyPageResult, error) {
		return k.bridgeSync.FetchReconcileKeysPage(ctx, args.Dataset, args.NextURL, args.UseFallback, args.CollectionSkip)
	})
}

func (k *KeyReconcile) RunSparkPage(ctx context.Context, job *queue.ReservedJob) error {
	var args reconcileKeysArgs
	if err := json.Unmarshal(job.Payload.Args, &args); err != nil {
		return err
	}
	return k.runPage(ctx, "spark", args, func(ctx context.Context) (KeyPageResult, error) {
		return k.sparkSync.FetchReconcileKeysPage(ctx, args.Dataset, args.NextURL)
	})
}

func (k *KeyReconcile) runPage(ctx context.Context, provider string, args reconcileKeysArgs, fetch func(context.Context) (KeyPageResult, error)) error {
	if args.Dataset == "" || args.RunID == "" {
		return fmt.Errorf("reconcile keys: missing dataset or run_id")
	}
	runID, err := uuid.Parse(args.RunID)
	if err != nil {
		return fmt.Errorf("reconcile keys: invalid run_id: %w", err)
	}

	start := time.Now()
	page, err := fetch(ctx)
	if err != nil {
		return err
	}

	if page.HTTPError {
		if provider == "bridge" && !args.UseFallback {
			k.logger.Warn("bridge reconcile replication failed; retrying on /Property collection",
				"dataset", args.Dataset, "status", page.HTTPStatus, "error", page.ODataError)
			return k.enqueueNextPage(ctx, provider, args, reconcileKeysArgs{
				RunID:       args.RunID,
				Dataset:     args.Dataset,
				UseFallback: true,
			})
		}
		if page.HTTPStatus == 429 || page.HTTPStatus == 503 {
			return mlsupstream.ErrRateLimited{Provider: provider, Status: page.HTTPStatus}
		}
		return fmt.Errorf("reconcile keys upstream HTTP %d: %s", page.HTTPStatus, page.ODataError)
	}

	if err := k.keyStore.InsertKeys(ctx, runID, args.Dataset, page.Keys); err != nil {
		return err
	}

	if !page.Complete {
		next := reconcileKeysArgs{
			RunID:          args.RunID,
			Dataset:        args.Dataset,
			NextURL:        page.NextURL,
			UseFallback:    page.UseFallback,
			CollectionSkip: page.CollectionSkip,
		}
		return k.enqueueNextPage(ctx, provider, args, next)
	}

	deleted, err := k.keyStore.DeleteStaleMirrorRows(ctx, runID, args.Dataset)
	if err != nil {
		return err
	}
	keysSeen, err := k.keyStore.CountKeys(ctx, runID, args.Dataset)
	if err != nil {
		return err
	}
	if err := k.keyStore.PurgeRun(ctx, runID); err != nil {
		return err
	}

	k.logger.Info("mirror key reconcile complete",
		"provider", provider,
		"dataset", args.Dataset,
		"run_id", args.RunID,
		"keys_seen", keysSeen,
		"rows_deleted", deleted,
		"duration_ms", time.Since(start).Milliseconds(),
	)
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
