package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// HandlerFunc processes a reserved job.
type HandlerFunc func(ctx context.Context, job *ReservedJob) error

// Worker polls and processes jobs from PostgreSQL.
type Worker struct {
	client            *Client
	queues            []string
	handlers          map[string]HandlerFunc
	maxAttempts       int
	rateLimitMax      int
	rateLimitRetry    time.Duration
	timeoutRetry      time.Duration
	pollEvery         time.Duration
	logger            *slog.Logger
	fairQueueCursor   int
	fairFetchCursor   int
	fairPersistCursor int
	fairOtherCursor   int
	fairPoolCursor    int
}

// WorkerRetryPolicy configures soft retries for MLS upstream failures.
type WorkerRetryPolicy struct {
	MaxAttempts        int
	RateLimitMax       int
	RateLimitRetry     time.Duration
	TimeoutRetry       time.Duration
}

func NewWorker(client *Client, queues []string, pollEvery time.Duration, logger *slog.Logger) *Worker {
	return NewWorkerWithRetry(client, queues, pollEvery, logger, WorkerRetryPolicy{
		MaxAttempts:    3,
		RateLimitMax:   50,
		RateLimitRetry: 300 * time.Second,
		TimeoutRetry:   60 * time.Second,
	})
}

func NewWorkerWithRetry(client *Client, queues []string, pollEvery time.Duration, logger *slog.Logger, policy WorkerRetryPolicy) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	if policy.MaxAttempts < 1 {
		policy.MaxAttempts = 3
	}
	if policy.RateLimitMax < 1 {
		policy.RateLimitMax = 50
	}
	if policy.RateLimitRetry <= 0 {
		policy.RateLimitRetry = 300 * time.Second
	}
	if policy.TimeoutRetry <= 0 {
		policy.TimeoutRetry = 60 * time.Second
	}
	return &Worker{
		client:         client,
		queues:         queues,
		handlers:       make(map[string]HandlerFunc),
		maxAttempts:    policy.MaxAttempts,
		rateLimitMax:   policy.RateLimitMax,
		rateLimitRetry: policy.RateLimitRetry,
		timeoutRetry:   policy.TimeoutRetry,
		pollEvery:      pollEvery,
		logger:         logger,
	}
}

func (w *Worker) Register(typ string, fn HandlerFunc) {
	w.handlers[typ] = fn
}

func (w *Worker) reserveNext(ctx context.Context) (*ReservedJob, error) {
	if len(w.queues) <= 1 {
		return w.client.Reserve(ctx, w.queues)
	}

	fetchQ, persistQ, otherQ := partitionWorkerQueues(w.queues)
	if len(fetchQ) > 0 && len(persistQ) > 0 {
		return w.reserveWeighted(ctx, fetchQ, persistQ, otherQ)
	}

	job, next, err := w.client.ReserveFair(ctx, w.queues, w.fairQueueCursor)
	if err != nil {
		return nil, err
	}
	w.fairQueueCursor = next
	return job, nil
}

// reserveWeighted alternates fetch vs persist pools so persist backlog cannot starve fetch (and vice versa).
func (w *Worker) reserveWeighted(ctx context.Context, fetchQ, persistQ, otherQ []string) (*ReservedJob, error) {
	pools := []struct {
		queues *[]string
		cursor *int
	}{
		{&fetchQ, &w.fairFetchCursor},
		{&persistQ, &w.fairPersistCursor},
	}
	if len(otherQ) > 0 {
		pools = append(pools, struct {
			queues *[]string
			cursor *int
		}{&otherQ, &w.fairOtherCursor})
	}

	n := len(pools)
	for i := 0; i < n; i++ {
		idx := (w.fairPoolCursor + i) % n
		q := *pools[idx].queues
		if len(q) == 0 {
			continue
		}
		job, next, err := w.client.ReserveFair(ctx, q, *pools[idx].cursor)
		if err != nil {
			return nil, err
		}
		*pools[idx].cursor = next
		if job != nil {
			w.fairPoolCursor = (idx + 1) % n
			return job, nil
		}
	}
	return nil, nil
}

func (w *Worker) Run(ctx context.Context) error {
	notifyCh, err := w.client.Listen(ctx)
	if err != nil {
		w.logger.Warn("queue listen failed, polling only", "error", err)
	}

	ticker := time.NewTicker(w.pollEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		job, err := w.reserveNext(ctx)
		if err != nil {
			w.logger.Error("reserve job", "error", err)
			w.sleep(ctx, w.pollEvery)
			continue
		}
		if job == nil {
			w.wait(ctx, ticker, notifyCh)
			continue
		}

		if err := w.process(ctx, job); err != nil {
			w.logger.Error("job failed", "id", job.ID, "type", job.Payload.Type, "error", err)
			_ = w.releaseFailed(ctx, job, err)
		} else {
			_ = w.client.Delete(ctx, job.ID)
			w.handleBatchComplete(ctx, job)
		}
	}
}

func (w *Worker) process(ctx context.Context, job *ReservedJob) error {
	if job.Payload.Type == "" && IsLegacyLaravelPayload(job.Raw) {
		w.logger.Info("discarded legacy Laravel queue job (purge with: DELETE FROM jobs WHERE payload LIKE '%CallQueuedHandler%')",
			"id", job.ID,
			"queue", job.Queue,
			"laravel_job", LegacyLaravelJobName(job.Raw),
		)
		return nil
	}

	fn, ok := w.handlers[job.Payload.Type]
	if !ok {
		w.logger.Warn("unknown job type", "type", job.Payload.Type, "id", job.ID, "queue", job.Queue)
		return fmt.Errorf("unknown job type %q", job.Payload.Type)
	}
	return fn(ctx, job)
}

func (w *Worker) handleBatchComplete(ctx context.Context, job *ReservedJob) {
	var wrapper struct {
		BatchID string          `json:"batch_id"`
		Job     json.RawMessage `json:"job"`
	}
	if len(job.Payload.Args) == 0 {
		return
	}
	if err := json.Unmarshal(job.Payload.Args, &wrapper); err != nil || wrapper.BatchID == "" {
		return
	}
	_ = w.client.CompleteBatchJob(ctx, wrapper.BatchID, job.Queue, "", nil)
}

func (w *Worker) wait(ctx context.Context, ticker *time.Ticker, notify <-chan struct{}) {
	select {
	case <-ctx.Done():
	case <-ticker.C:
	case <-notify:
	}
}

func (w *Worker) releaseFailed(ctx context.Context, job *ReservedJob, jobErr error) error {
	if IsRateLimited(jobErr) {
		if job.Attempts() >= w.rateLimitMax {
			return w.client.Fail(ctx, job, jobErr)
		}
		return w.client.ReleaseAt(ctx, job, w.rateLimitRetry, true)
	}
	if IsTimeout(jobErr) {
		if job.Attempts() >= w.maxAttempts {
			return w.client.Fail(ctx, job, jobErr)
		}
		return w.client.ReleaseAt(ctx, job, w.timeoutRetry, false)
	}
	return w.client.Release(ctx, job, w.maxAttempts, jobErr)
}

func (w *Worker) sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
