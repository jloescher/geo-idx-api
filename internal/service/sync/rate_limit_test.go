package sync

import (
	"context"
	"testing"
	"time"
)

func TestSyncRateLimiterSpacesRequests(t *testing.T) {
	lim := newSyncRateLimiter(10)
	ctx := context.Background()
	start := time.Now()
	if err := lim.wait(ctx); err != nil {
		t.Fatal(err)
	}
	if err := lim.wait(ctx); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Fatalf("expected spacing between requests, got %v", elapsed)
	}
}

func TestSyncRateLimiterDisabled(t *testing.T) {
	if lim := newSyncRateLimiter(0); lim != nil {
		t.Fatal("expected nil limiter when max per second is 0")
	}
}
