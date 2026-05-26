package mlsupstream

import (
	"errors"
	"testing"
	"time"
)

func TestIsRateLimited(t *testing.T) {
	err := ErrRateLimited{Provider: "spark", Status: 429}
	if !IsRateLimited(err) {
		t.Fatal("expected rate limit detection")
	}
	if IsRateLimited(errors.New("other")) {
		t.Fatal("unexpected match")
	}
}

func TestRetryDelay(t *testing.T) {
	rl := 300 * time.Second
	to := 60 * time.Second
	if got := RetryDelay(ErrRateLimited{}, rl, to); got != rl {
		t.Fatalf("expected rate limit delay %v, got %v", rl, got)
	}
	if got := RetryDelay(ErrTimeout{}, rl, to); got != to {
		t.Fatalf("expected timeout delay %v, got %v", to, got)
	}
}
