package sync

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quantyralabs/idx-api/internal/config"
)

const sparkBudgetWindow = 5 * time.Minute

// ClusterRateLimiter coordinates MLS HTTP spacing across processes via PostgreSQL.
type ClusterRateLimiter struct {
	pool        *pgxpool.Pool
	provider    string
	minInterval time.Duration
	maxPer5Min  int
}

// NewClusterRateLimiter builds a limiter for provider bridge|spark.
// maxPer5Min <= 0 disables the rolling 5-minute window (Bridge).
func NewClusterRateLimiter(pool *pgxpool.Pool, provider string, maxPerSecond, maxPer5Min int) *ClusterRateLimiter {
	var minInterval time.Duration
	if maxPerSecond > 0 {
		minInterval = time.Second / time.Duration(maxPerSecond)
	}
	return &ClusterRateLimiter{
		pool:        pool,
		provider:    provider,
		minInterval: minInterval,
		maxPer5Min:  maxPer5Min,
	}
}

// NewSparkClusterRateLimiter uses Spark sync env caps (replication + live).
func NewSparkClusterRateLimiter(pool *pgxpool.Pool, cfg config.Config) *ClusterRateLimiter {
	return NewClusterRateLimiter(pool, "spark", cfg.Spark.SyncMaxRequestsPerSecond, cfg.Spark.SyncMaxRequestsPer5Min)
}

// NewBridgeClusterRateLimiter uses Bridge sync env caps (spacing only).
func NewBridgeClusterRateLimiter(pool *pgxpool.Pool, cfg config.Config) *ClusterRateLimiter {
	return NewClusterRateLimiter(pool, "bridge", cfg.Bridge.SyncMaxRequestsPerSecond, 0)
}

// Wait blocks until this process may issue the next outbound HTTP attempt for the provider.
func (l *ClusterRateLimiter) Wait(ctx context.Context) error {
	if l == nil || l.pool == nil {
		return nil
	}
	if l.minInterval <= 0 && l.maxPer5Min <= 0 {
		return nil
	}
	for {
		wait, err := l.tryAcquire(ctx)
		if err != nil {
			return err
		}
		if wait <= 0 {
			return nil
		}
		if wait > 60*time.Second {
			wait = 60 * time.Second
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (l *ClusterRateLimiter) tryAcquire(ctx context.Context) (time.Duration, error) {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var nextAllowedAt, windowStart time.Time
	var windowCount int
	err = tx.QueryRow(ctx, `
		SELECT next_allowed_at, window_start, window_count
		FROM sync_rate_budget
		WHERE provider = $1
		FOR UPDATE
	`, l.provider).Scan(&nextAllowedAt, &windowStart, &windowCount)
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()

	if l.maxPer5Min > 0 && now.Sub(windowStart) >= sparkBudgetWindow {
		windowStart = now
		windowCount = 0
	}

	if l.maxPer5Min > 0 && windowCount >= l.maxPer5Min {
		resetAt := windowStart.Add(sparkBudgetWindow)
		if now.Before(resetAt) {
			return resetAt.Sub(now), nil
		}
		windowStart = now
		windowCount = 0
	}

	if l.minInterval > 0 && now.Before(nextAllowedAt) {
		return nextAllowedAt.Sub(now), nil
	}

	newNext := now
	if l.minInterval > 0 {
		base := nextAllowedAt
		if now.After(base) {
			base = now
		}
		newNext = base.Add(l.minInterval)
	}
	windowCount++

	_, err = tx.Exec(ctx, `
		UPDATE sync_rate_budget
		SET next_allowed_at = $2, window_start = $3, window_count = $4
		WHERE provider = $1
	`, l.provider, newNext, windowStart, windowCount)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return 0, nil
}

// MinInterval exposes spacing for tests.
func (l *ClusterRateLimiter) MinInterval() time.Duration {
	if l == nil {
		return 0
	}
	return l.minInterval
}
