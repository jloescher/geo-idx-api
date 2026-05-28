package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/scheduler"
)

// TestSchedulerLockHealthPoolProbeLeak documents why SchedulerLockHealth must not
// call pg_try_advisory_lock on one pool connection and pg_advisory_unlock on another.
func TestSchedulerLockHealthPoolProbeLeak(t *testing.T) {
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

	const lockID int64 = 913374299

	// Broken probe pattern (old monitoring): lock on conn A, unlock on conn B.
	var acquired bool
	if err := pool.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockID).Scan(&acquired); err != nil {
		t.Fatal(err)
	}
	if !acquired {
		t.Fatal("expected probe to acquire lock")
	}
	if _, err := pool.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lockID); err != nil {
		t.Fatal(err)
	}

	// Lock remains on the first pooled connection; scheduler cannot lead.
	leader, ok, err := scheduler.TryAcquireLeader(ctx, dsn, lockID)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		leader.Release(ctx)
		t.Fatal("scheduler should not acquire after broken pool probe left lock on another session")
	}

	// Observe via pg_locks and clear by releasing on the holding connection.
	var holderPID int32
	err = pool.QueryRow(ctx, `
		SELECT pid FROM pg_locks
		WHERE locktype = 'advisory' AND classid = 0 AND objid = $1::bigint AND granted
		LIMIT 1
	`, lockID).Scan(&holderPID)
	if err != nil {
		t.Fatalf("expected leaked lock in pg_locks: %v", err)
	}

	// Release leaked lock by terminating the backend that still holds it on the pooled connection.
	_, _ = pool.Exec(ctx, `SELECT pg_terminate_backend($1)`, holderPID)

	leader2, ok2, err := scheduler.TryAcquireLeader(ctx, dsn, lockID)
	if err != nil || !ok2 {
		t.Fatalf("after cleanup acquire ok=%v err=%v", ok2, err)
	}
	leader2.Release(ctx)
}

func TestSchedulerLockHealthObservesHolder(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	db, err := repository.NewFromDSN(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	const lockID int64 = 913374298
	repo := repository.NewMonitoringRepo(db)

	health, err := repo.SchedulerLockHealth(ctx, lockID)
	if err != nil {
		t.Fatal(err)
	}
	if health.LeaderActive {
		t.Fatal("expected no leader before acquire")
	}

	leader, ok, err := scheduler.TryAcquireLeader(ctx, dsn, lockID)
	if err != nil || !ok {
		t.Fatalf("acquire ok=%v err=%v", ok, err)
	}
	defer leader.Release(ctx)

	health, err = repo.SchedulerLockHealth(ctx, lockID)
	if err != nil {
		t.Fatal(err)
	}
	if !health.LeaderActive {
		t.Fatal("expected leader_active after scheduler acquire")
	}
	if health.HolderPID == nil || *health.HolderPID == 0 {
		t.Fatal("expected holder_pid from pg_locks")
	}
}
