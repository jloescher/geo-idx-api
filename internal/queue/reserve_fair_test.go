package queue_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quantyralabs/idx-api/internal/queue"
)

func TestReserveFairRotatesQueues(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	client := queue.NewClient(pool, "jobs", "idx_jobs_wakeup_fair_test", 90*time.Second, time.Hour)

	queues := []string{"fair-test-a", "fair-test-b"}
	for _, q := range queues {
		if _, err := pool.Exec(ctx, `DELETE FROM jobs WHERE queue = $1`, q); err != nil {
			t.Fatal(err)
		}
	}

	idA, err := client.Enqueue(ctx, queues[0], queue.TypeNoop, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	idB, err := client.Enqueue(ctx, queues[1], queue.TypeNoop, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	job, next, err := client.ReserveFair(ctx, queues, 0)
	if err != nil {
		t.Fatal(err)
	}
	if job == nil || job.ID != idA || job.Queue != queues[0] {
		t.Fatalf("expected first queue job %d, got %+v", idA, job)
	}
	if next != 1 {
		t.Fatalf("expected next cursor 1, got %d", next)
	}

	if err := client.Delete(ctx, job.ID); err != nil {
		t.Fatal(err)
	}

	job, _, err = client.ReserveFair(ctx, queues, next)
	if err != nil {
		t.Fatal(err)
	}
	if job == nil || job.ID != idB || job.Queue != queues[1] {
		t.Fatalf("expected second queue job %d, got %+v", idB, job)
	}
	if err := client.Delete(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
}

func TestReserveFairSkipsEmptyQueue(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	client := queue.NewClient(pool, "jobs", "idx_jobs_wakeup_fair_skip", 90*time.Second, time.Hour)
	queues := []string{"fair-empty-a", "fair-has-b"}
	if _, err := pool.Exec(ctx, `DELETE FROM jobs WHERE queue = ANY($1)`, queues); err != nil {
		t.Fatal(err)
	}

	idB, err := client.Enqueue(ctx, queues[1], queue.TypeNoop, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	job, next, err := client.ReserveFair(ctx, queues, 0)
	if err != nil {
		t.Fatal(err)
	}
	if job == nil || job.ID != idB {
		t.Fatalf("expected job on %s, got %+v", queues[1], job)
	}
	if next != 1 {
		t.Fatalf("expected next index 1, got %d", next)
	}
	if err := client.Delete(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
}
