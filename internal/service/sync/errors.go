package sync

import (
	"time"

	"github.com/quantyralabs/idx-api/internal/mlsupstream"
)

type (
	ErrUpstreamRateLimited = mlsupstream.ErrRateLimited
	ErrUpstreamTimeout     = mlsupstream.ErrTimeout
)

func IsUpstreamRateLimited(err error) bool { return mlsupstream.IsRateLimited(err) }

func IsUpstreamTimeout(err error) bool { return mlsupstream.IsTimeout(err) }

func UpstreamRetryDelay(err error, rateLimitDelay, timeoutDelay time.Duration) time.Duration {
	return mlsupstream.RetryDelay(err, rateLimitDelay, timeoutDelay)
}
