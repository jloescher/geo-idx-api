package scheduler_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/quantyralabs/idx-api/internal/scheduler"
)

func TestAdvisoryLockSingleHolder(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	key := int64(913374211)
	leader1, ok, err := scheduler.TryAcquireLeader(ctx, dsn, key)
	if err != nil || !ok {
		t.Fatalf("first acquire ok=%v err=%v", ok, err)
	}
	defer leader1.Release(ctx)

	leader2, ok2, err := scheduler.TryAcquireLeader(ctx, dsn, key)
	if err != nil {
		t.Fatal(err)
	}
	if ok2 {
		leader2.Release(ctx)
		t.Fatal("second acquire should fail while first holds lock")
	}
}

func TestAdvisoryLockHandoff(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	key := int64(913374212)
	leader1, ok, err := scheduler.TryAcquireLeader(ctx, dsn, key)
	if err != nil || !ok {
		t.Fatalf("acquire: ok=%v err=%v", ok, err)
	}
	leader1.Release(ctx)

	leader2, ok2, err := scheduler.TryAcquireLeader(ctx, dsn, key)
	if err != nil || !ok2 {
		t.Fatalf("re-acquire after release: ok=%v err=%v", ok2, err)
	}
	leader2.Release(ctx)
}

func TestAdvisoryLockConcurrentAcquire(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	key := int64(913374213)
	var wg sync.WaitGroup
	holders := make(chan bool, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			leader, ok, err := scheduler.TryAcquireLeader(ctx, dsn, key)
			if err != nil {
				holders <- false
				return
			}
			if ok {
				time.Sleep(50 * time.Millisecond)
				leader.Release(ctx)
			}
			holders <- ok
		}()
	}
	wg.Wait()
	close(holders)
	wins := 0
	for ok := range holders {
		if ok {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("expected exactly one lock holder, got %d", wins)
	}
}
