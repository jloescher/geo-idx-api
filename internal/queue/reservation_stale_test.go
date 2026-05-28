package queue

import (
	"testing"
	"time"
)

func TestReservationStaleBefore(t *testing.T) {
	c := NewClient(nil, "jobs", "idx_jobs_wakeup", time.Minute, time.Hour)
	now := int64(1_000_000)
	got := c.reservationStaleBefore(now)
	want := now - 1800
	if got != want {
		t.Fatalf("got %d want %d (half of 3600s)", got, want)
	}
	short := NewClient(nil, "jobs", "idx_jobs_wakeup", time.Minute, 10*time.Minute)
	got = short.reservationStaleBefore(now)
	want = now - 600
	if got != want {
		t.Fatalf("short timeout: got %d want %d (10m floor)", got, want)
	}
}
