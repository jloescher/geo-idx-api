package queue_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
)

func TestEnqueueReserveRoundTrip(t *testing.T) {
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

	client := queue.NewClient(pool, "jobs", "idx_jobs_wakeup_test", 90*time.Second, time.Hour)

	_, _ = pool.Exec(ctx, `DELETE FROM jobs WHERE queue = 'test-roundtrip'`)

	id, err := client.Enqueue(ctx, "test-roundtrip", queue.TypeNoop, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	job, err := client.Reserve(ctx, []string{"test-roundtrip"})
	if err != nil {
		t.Fatal(err)
	}
	if job == nil || job.ID != id {
		t.Fatalf("expected job %d, got %+v", id, job)
	}
	if job.Payload.Type != queue.TypeNoop {
		t.Fatalf("unexpected type %s", job.Payload.Type)
	}

	if err := client.Delete(ctx, job.ID); err != nil {
		t.Fatal(err)
	}
}

func TestConfigLoad(t *testing.T) {
	_, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
}
