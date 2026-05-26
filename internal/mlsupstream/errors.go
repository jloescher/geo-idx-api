// Package mlsupstream defines typed errors for MLS HTTP fetch failures (shared by queue and sync).
package mlsupstream

import (
	"errors"
	"fmt"
	"time"
)

// ErrRateLimited is returned when MLS upstream responds 429 after in-request retries.
type ErrRateLimited struct {
	Provider string
	Status   int
}

func (e ErrRateLimited) Error() string {
	return fmt.Sprintf("%s upstream rate limited (HTTP %d)", e.Provider, e.Status)
}

// ErrTimeout is returned for context deadlines or upstream 503 timeouts.
type ErrTimeout struct {
	Provider string
	Status   int
	Cause    error
}

func (e ErrTimeout) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s upstream timeout: %v", e.Provider, e.Cause)
	}
	if e.Status > 0 {
		return fmt.Sprintf("%s upstream timeout (HTTP %d)", e.Provider, e.Status)
	}
	return fmt.Sprintf("%s upstream timeout", e.Provider)
}

func (e ErrTimeout) Unwrap() error { return e.Cause }

// IsRateLimited reports whether err is a rate-limit failure from sync fetch.
func IsRateLimited(err error) bool {
	var rl ErrRateLimited
	return errors.As(err, &rl)
}

// IsTimeout reports whether err is a timeout-class failure from sync fetch.
func IsTimeout(err error) bool {
	var to ErrTimeout
	return errors.As(err, &to)
}

// RetryDelay picks queue/self-heal delay for a sync fetch failure.
func RetryDelay(err error, rateLimitDelay, timeoutDelay time.Duration) time.Duration {
	if IsRateLimited(err) {
		return rateLimitDelay
	}
	if IsTimeout(err) {
		return timeoutDelay
	}
	return rateLimitDelay
}
