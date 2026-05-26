package sync

import (
	"testing"
	"time"
)

func TestClusterRateLimiterMinInterval(t *testing.T) {
	lim := NewClusterRateLimiter(nil, "spark", 4, 1200)
	if got := lim.MinInterval(); got != 250*time.Millisecond {
		t.Fatalf("expected 250ms spacing, got %v", got)
	}
}

func TestClusterRateLimiterDisabled(t *testing.T) {
	lim := NewClusterRateLimiter(nil, "bridge", 0, 0)
	if err := lim.Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestParseRetryAfterSeconds(t *testing.T) {
	if got := parseRetryAfter("30"); got != 30*time.Second {
		t.Fatalf("expected 30s, got %v", got)
	}
}

func TestParseRetryAfterEmpty(t *testing.T) {
	if got := parseRetryAfter(""); got != 0 {
		t.Fatalf("expected 0, got %v", got)
	}
}
