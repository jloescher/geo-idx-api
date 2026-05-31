package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Client is a PostgreSQL-backed job queue.
// The jobs table schema maintains compatibility with the prior Laravel implementation
// for safe cutover and legacy job purging during migration.
// Revenue impact: durable async work without Redis keeps ops cost low and survives restarts.
type Client struct {
	pool               *pgxpool.Pool
	table              string
	notifyChannel      string
	retryAfter         time.Duration
	reservationTimeout time.Duration
}

func NewClient(pool *pgxpool.Pool, table, notifyChannel string, retryAfter, reservationTimeout time.Duration) *Client {
	if table == "" {
		table = "jobs"
	}
	if notifyChannel == "" {
		notifyChannel = "idx_jobs_wakeup"
	}
	if reservationTimeout <= 0 {
		reservationTimeout = time.Hour
	}
	return &Client{
		pool:               pool,
		table:              table,
		notifyChannel:      notifyChannel,
		retryAfter:         retryAfter,
		reservationTimeout: reservationTimeout,
	}
}

// Enqueue inserts a job and NOTIFYs workers.
func (c *Client) Enqueue(ctx context.Context, queueName string, typ string, args any, delay time.Duration) (int64, error) {
	payload, err := MarshalPayload(typ, args)
	if err != nil {
		return 0, err
	}
	now := time.Now().Unix()
	available := now
	if delay > 0 {
		available = now + int64(delay.Seconds())
	}

	var id int64
	err = c.pool.QueryRow(ctx, fmt.Sprintf(`
		INSERT INTO %s (queue, payload, attempts, reserved_at, available_at, created_at)
		VALUES ($1, $2, 0, NULL, $3, $4)
		RETURNING id
	`, c.table), queueName, string(payload), available, now).Scan(&id)
	if err != nil {
		return 0, err
	}

	_, _ = c.pool.Exec(ctx, "SELECT pg_notify($1, $2)", c.notifyChannel, queueName)

	return id, nil
}

// batchOnComplete is stored in job_batches.options for finalize dispatch.
type batchOnComplete struct {
	Queue string `json:"queue,omitempty"`
	Type  string `json:"type"`
	Args  any    `json:"args,omitempty"`
}

// batchOptionsJSON is stored in job_batches.options.
type batchOptionsJSON struct {
	OnComplete *batchOnComplete `json:"on_complete,omitempty"`
}

// BatchSpec defines a batch of jobs with a completion callback.
type BatchSpec struct {
	Name       string
	Queue      string
	Jobs       []BatchJob
	OnComplete BatchJob // enqueued when all jobs finish
}

// BatchJob is one unit of work in a batch.
type BatchJob struct {
	Type string
	Args any
}

// EnqueueBatch creates job_batches row and child jobs linked by batch_id in payload.
// Revenue impact: parallel persist chunks maximize throughput while finalize gates cursor advance.
func (c *Client) EnqueueBatch(ctx context.Context, spec BatchSpec) (batchID string, err error) {
	if len(spec.Jobs) == 0 {
		return "", fmt.Errorf("batch requires at least one job")
	}
	batchID = uuid.NewString()
	now := time.Now().Unix()

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var options *string
	if spec.OnComplete.Type != "" {
		b, err := json.Marshal(batchOptionsJSON{OnComplete: &batchOnComplete{
			Queue: spec.Queue,
			Type:  spec.OnComplete.Type,
			Args:  spec.OnComplete.Args,
		}})
		if err != nil {
			return "", err
		}
		s := string(b)
		options = &s
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO job_batches (id, name, total_jobs, pending_jobs, failed_jobs, failed_job_ids, options, cancelled_at, created_at, finished_at)
		VALUES ($1, $2, $3, $3, 0, '[]', $4, NULL, $5, NULL)
	`, batchID, spec.Name, len(spec.Jobs), options, now)
	if err != nil {
		return "", err
	}

	for _, job := range spec.Jobs {
		argsWithBatch, _ := json.Marshal(map[string]any{
			"batch_id": batchID,
			"job":      job.Args,
		})
		payload, err := MarshalPayload(job.Type, json.RawMessage(argsWithBatch))
		if err != nil {
			return "", err
		}
		_, err = tx.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s (queue, payload, attempts, reserved_at, available_at, created_at)
			VALUES ($1, $2, 0, NULL, $3, $4)
		`, c.table), spec.Queue, string(payload), now, now)
		if err != nil {
			return "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	_, _ = c.pool.Exec(ctx, "SELECT pg_notify($1, $2)", c.notifyChannel, spec.Queue)

	return batchID, nil
}

// ReservedJob is a claimed job ready for processing.
type ReservedJob struct {
	ID       int64
	Queue    string
	Payload  Payload
	Raw      []byte
	attempts int // post-reserve count (incremented in Reserve)
}

// Reserve claims the next available job on one of the given queues (SKIP LOCKED), lowest id first.
func (c *Client) Reserve(ctx context.Context, queues []string) (*ReservedJob, error) {
	return c.reserveFromQueues(ctx, queues)
}

// ReserveFair rotates across queues so one feed cannot monopolize lowest job ids.
func (c *Client) ReserveFair(ctx context.Context, queues []string, startIndex int) (*ReservedJob, int, error) {
	if len(queues) == 0 {
		return nil, 0, nil
	}
	n := len(queues)
	for i := 0; i < n; i++ {
		idx := (startIndex + i) % n
		job, err := c.reserveFromQueues(ctx, []string{queues[idx]})
		if err != nil {
			return nil, startIndex, err
		}
		if job != nil {
			return job, (idx + 1) % n, nil
		}
	}
	return nil, startIndex, nil
}

func (c *Client) reserveFromQueues(ctx context.Context, queues []string) (*ReservedJob, error) {
	if len(queues) == 0 {
		return nil, nil
	}
	now := time.Now().Unix()
	staleBefore := c.reservationStaleBefore(now)

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, fmt.Sprintf(`
		UPDATE %s
		SET reserved_at = NULL, available_at = $1
		WHERE reserved_at IS NOT NULL AND reserved_at < $2
	`, c.table), now, staleBefore)
	if err != nil {
		return nil, err
	}

	var id int64
	var queue string
	var payloadStr string
	var attempts int

	err = tx.QueryRow(ctx, fmt.Sprintf(`
		SELECT id, queue, payload, attempts FROM %s
		WHERE queue = ANY($1)
		  AND reserved_at IS NULL
		  AND available_at <= $2
		ORDER BY id ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, c.table), queues, now).Scan(&id, &queue, &payloadStr, &attempts)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(`
		UPDATE %s SET reserved_at = $1, attempts = attempts + 1 WHERE id = $2
	`, c.table), now, id)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	p, err := UnmarshalPayload([]byte(payloadStr))
	if err != nil {
		return nil, err
	}

	return &ReservedJob{
		ID:       id,
		Queue:    queue,
		Payload:  p,
		Raw:      []byte(payloadStr),
		attempts: attempts + 1,
	}, nil
}

// reservationStaleBefore matches monitoring stale_reserved (half of reservation timeout, min 10m).
func (c *Client) reservationStaleBefore(now int64) int64 {
	sec := int64(c.reservationTimeout.Seconds()) / 2
	if sec < 600 {
		sec = 600
	}
	return now - sec
}

// Delete removes a successfully processed job.
func (c *Client) Delete(ctx context.Context, id int64) error {
	_, err := c.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, c.table), id)
	return err
}

// Release returns a failed job to the queue or moves to failed_jobs after max attempts.
func (c *Client) Release(ctx context.Context, job *ReservedJob, maxAttempts int, jobErr error) error {
	if job.Attempts() >= maxAttempts {
		return c.Fail(ctx, job, jobErr)
	}
	delay := c.retryAfter
	available := time.Now().Add(delay).Unix()
	_, err := c.pool.Exec(ctx, fmt.Sprintf(`
		UPDATE %s SET reserved_at = NULL, available_at = $1 WHERE id = $2
	`, c.table), available, job.ID)
	return err
}

// ReleaseAt re-queues a failed job for a custom delay. When decrementAttempts is true,
// the reserve-time attempt increment is rolled back (soft retry for rate limits).
func (c *Client) ReleaseAt(ctx context.Context, job *ReservedJob, delay time.Duration, decrementAttempts bool) error {
	available := time.Now().Add(delay).Unix()
	if decrementAttempts {
		_, err := c.pool.Exec(ctx, fmt.Sprintf(`
			UPDATE %s
			SET reserved_at = NULL, available_at = $1, attempts = GREATEST(attempts - 1, 0)
			WHERE id = $2
		`, c.table), available, job.ID)
		return err
	}
	_, err := c.pool.Exec(ctx, fmt.Sprintf(`
		UPDATE %s SET reserved_at = NULL, available_at = $1 WHERE id = $2
	`, c.table), available, job.ID)
	return err
}

// HasPendingJobType reports whether any job of the given type is queued or in-flight on the queue.
func (c *Client) HasPendingJobType(ctx context.Context, queueName, jobType string) (bool, error) {
	var exists bool
	err := c.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %s
			WHERE queue = $1
			  AND (
			    (reserved_at IS NULL AND available_at <= extract(epoch from now())::bigint)
			    OR reserved_at IS NOT NULL
			  )
			  AND payload::jsonb->>'type' = $2
		)
	`, c.table), queueName, jobType).Scan(&exists)
	return exists, err
}

// HasPendingFetch reports whether a fetch_page job for the dataset is queued or in-flight.
func (c *Client) HasPendingFetch(ctx context.Context, queueName, jobType, dataset string) (bool, error) {
	var exists bool
	err := c.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT EXISTS (
			SELECT 1 FROM %s
			WHERE queue = $1
			  AND (
			    (reserved_at IS NULL AND available_at <= extract(epoch from now())::bigint)
			    OR reserved_at IS NOT NULL
			  )
			  AND payload::jsonb->>'type' = $2
			  AND payload::jsonb->'args'->>'dataset' = $3
		)
	`, c.table), queueName, jobType, dataset).Scan(&exists)
	return exists, err
}

func (j *ReservedJob) Attempts() int {
	return j.attempts
}

// Fail records job in failed_jobs and deletes from jobs.
func (c *Client) Fail(ctx context.Context, job *ReservedJob, jobErr error) error {
	uid := uuid.NewString()
	now := time.Now()
	_, err := c.pool.Exec(ctx, `
		INSERT INTO failed_jobs (uuid, connection, queue, payload, exception, failed_at)
		VALUES ($1, 'database', $2, $3, $4, $5)
	`, uid, job.Queue, string(job.Raw), jobErr.Error(), now)
	if err != nil {
		return err
	}
	return c.Delete(ctx, job.ID)
}

// CompleteBatchJob decrements pending_jobs; enqueues on_complete from job_batches.options when zero.
func (c *Client) CompleteBatchJob(ctx context.Context, batchID, completeQueue, completeType string, completeArgs any) error {
	var pending int
	var options *string
	err := c.pool.QueryRow(ctx, `
		UPDATE job_batches
		SET pending_jobs = pending_jobs - 1
		WHERE id = $1 AND pending_jobs > 0
		RETURNING pending_jobs, options
	`, batchID).Scan(&pending, &options)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if pending == 0 {
		now := time.Now().Unix()
		_, _ = c.pool.Exec(ctx, `UPDATE job_batches SET finished_at = $1 WHERE id = $2`, now, batchID)

		typ, queue, args := completeType, completeQueue, completeArgs
		if options != nil && *options != "" {
			var opts batchOptionsJSON
			if json.Unmarshal([]byte(*options), &opts) == nil && opts.OnComplete != nil && opts.OnComplete.Type != "" {
				typ = opts.OnComplete.Type
				args = opts.OnComplete.Args
				if opts.OnComplete.Queue != "" {
					queue = opts.OnComplete.Queue
				}
			}
		}
		if typ != "" && queue != "" {
			_, err = c.Enqueue(ctx, queue, typ, args, 0)
			return err
		}
	}
	return nil
}

// Listen returns a channel signaled on NOTIFY (caller should also poll).
func (c *Client) Listen(ctx context.Context) (<-chan struct{}, error) {
	ch := make(chan struct{}, 1)
	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		defer conn.Release()
		defer close(ch)
		_, err := conn.Exec(ctx, "LISTEN "+quoteIdent(c.notifyChannel))
		if err != nil {
			return
		}
		for {
			notification, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				return
			}
			if notification != nil {
				select {
				case ch <- struct{}{}:
				default:
				}
			}
		}
	}()

	return ch, nil
}
