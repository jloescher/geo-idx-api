package queue

import (
	"errors"
	"time"

	"github.com/quantyralabs/idx-api/internal/mlsupstream"
)

// ErrReconcileBusy indicates another worker holds the reconcile run advisory lock.
type ErrReconcileBusy struct {
	RunID string
}

func (e ErrReconcileBusy) Error() string {
	return "mirror key reconcile run busy: " + e.RunID
}

// IsReconcileBusy reports whether err is a reconcile single-flight lock failure.
func IsReconcileBusy(err error) bool {
	var busy ErrReconcileBusy
	return errors.As(err, &busy)
}

type (
	// ErrRateLimited is returned when MLS upstream responds 429 after in-request retries.
	ErrRateLimited = mlsupstream.ErrRateLimited
	// ErrTimeout is returned for context deadlines or upstream 503 timeouts.
	ErrTimeout = mlsupstream.ErrTimeout
)

// IsRateLimited reports whether err is a rate-limit failure from sync fetch.
func IsRateLimited(err error) bool { return mlsupstream.IsRateLimited(err) }

// IsTimeout reports whether err is a timeout-class failure from sync fetch.
func IsTimeout(err error) bool { return mlsupstream.IsTimeout(err) }

// RetryDelay picks queue/self-heal delay for a sync fetch failure.
func RetryDelay(err error, rateLimitDelay, timeoutDelay time.Duration) time.Duration {
	return mlsupstream.RetryDelay(err, rateLimitDelay, timeoutDelay)
}
