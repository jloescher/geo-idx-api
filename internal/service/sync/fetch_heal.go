package sync

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/mlsupstream"
	"github.com/quantyralabs/idx-api/internal/queue"
)

// fetchHTTPFailure converts a non-2xx page result into a typed sync error.
func fetchHTTPFailure(provider string, status int) error {
	switch status {
	case http.StatusTooManyRequests:
		return mlsupstream.ErrRateLimited{Provider: provider, Status: status}
	case http.StatusServiceUnavailable:
		return mlsupstream.ErrTimeout{Provider: provider, Status: status}
	default:
		return errors.New(provider + " fetch http " + http.StatusText(status))
	}
}

// maybeSelfHealReplicationFetch re-enqueues replication fetch when the chain is still active.
// Returns nil when the current job should complete without failing (self-heal enqueued).
func maybeSelfHealReplicationFetch(
	ctx context.Context,
	cfg config.Config,
	q *queue.Client,
	logger *slog.Logger,
	provider, dataset, fetchQueue, jobType, mode string,
	cursor SyncCursor,
	fetchErr error,
) (bool, error) {
	if mode != "replication" || !ReplicationChainActive(cursor) {
		return false, fetchErr
	}
	if fetchErr == nil || (!mlsupstream.IsRateLimited(fetchErr) && !mlsupstream.IsTimeout(fetchErr)) {
		return false, fetchErr
	}

	delay := UpstreamRetryDelay(fetchErr,
		time.Duration(cfg.MLS.RateLimitRetrySeconds)*time.Second,
		time.Duration(cfg.MLS.TimeoutRetrySeconds)*time.Second,
	)

	pending, err := q.HasPendingFetch(ctx, fetchQueue, jobType, dataset)
	if err != nil {
		return false, err
	}
	if pending {
		logger.Info("self-heal skipped: fetch already pending",
			"provider", provider, "dataset", dataset)
		return true, nil
	}

	id, err := q.Enqueue(ctx, fetchQueue, jobType, fetchPageArgs{
		Dataset: dataset,
		Mode:    "replication",
	}, delay)
	if err != nil {
		return false, fetchErr
	}
	logger.Info("self-heal re-enqueued replication fetch",
		"provider", provider, "dataset", dataset, "job_id", id, "delay", delay)
	return true, nil
}

const incrementalBadRequestHealDelay = 30 * time.Second

// maybeSelfHealIncrementalBadRequest clears a pinned incremental_window_end and re-enqueues
// incremental fetch after Spark/Bridge reject the OData query with HTTP 400.
// Returns true when the current job should complete without failing.
func maybeSelfHealIncrementalBadRequest(
	ctx context.Context,
	q *queue.Client,
	cursors *CursorStore,
	logger *slog.Logger,
	provider, dataset, fetchQueue, jobType, mode string,
	httpStatus int,
) (bool, error) {
	if mode != "incremental" || httpStatus != http.StatusBadRequest {
		return false, nil
	}

	pending, err := q.HasPendingFetch(ctx, fetchQueue, jobType, dataset)
	if err != nil {
		return false, err
	}
	if pending {
		logger.Info("incremental 400 self-heal skipped: fetch already pending",
			"provider", provider, "dataset", dataset)
		return true, nil
	}

	if err := cursors.ApplyPatch(ctx, dataset, CursorPatch{ClearIncrementalWindowEnd: true}); err != nil {
		return false, err
	}

	id, err := q.Enqueue(ctx, fetchQueue, jobType, fetchPageArgs{
		Dataset: dataset,
		Mode:    "incremental",
	}, incrementalBadRequestHealDelay)
	if err != nil {
		return false, err
	}
	logger.Info("incremental 400 self-heal: cleared window and re-enqueued fetch",
		"provider", provider, "dataset", dataset, "job_id", id, "delay", incrementalBadRequestHealDelay)
	return true, nil
}
