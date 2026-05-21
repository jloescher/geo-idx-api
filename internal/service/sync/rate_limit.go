package sync

import (
	"context"
	"sync"
	"time"
)

// syncRateLimiter spaces outbound MLS sync HTTP GETs (Bridge fetch jobs).
type syncRateLimiter struct {
	minInterval time.Duration
	mu          sync.Mutex
	last        time.Time
}

func newSyncRateLimiter(maxPerSecond int) *syncRateLimiter {
	if maxPerSecond <= 0 {
		return nil
	}
	return &syncRateLimiter{
		minInterval: time.Second / time.Duration(maxPerSecond),
	}
}

func (r *syncRateLimiter) wait(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.last.IsZero() {
		wait := r.minInterval - time.Since(r.last)
		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	r.last = time.Now()
	return nil
}
