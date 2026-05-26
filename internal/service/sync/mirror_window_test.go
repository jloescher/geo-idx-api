package sync

import (
	"strings"
	"testing"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestBridgeReplicationFilterAllTime(t *testing.T) {
	cfg := config.Config{MLS: config.MLSConfig{LocalMirrorRollingMonths: 0}}
	got := BridgeReplicationFilter(cfg)
	if got != activePendingStatusFilter {
		t.Fatalf("got %q", got)
	}
}

func TestBridgeReplicationFilterRolling(t *testing.T) {
	cfg := config.Config{MLS: config.MLSConfig{LocalMirrorRollingMonths: 3}}
	got := BridgeReplicationFilter(cfg)
	if got != activePendingStatusFilter {
		t.Fatalf("Bridge replication must not add timestamp $filter (Stellar returns 400); got %q", got)
	}
}

func TestBridgeIncrementalFilter(t *testing.T) {
	since := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	got := BridgeIncrementalFilter("stellar", since)
	if !strings.Contains(got, activePendingStatusFilter) {
		t.Fatalf("missing status: %q", got)
	}
	if !strings.Contains(got, "BridgeModificationTimestamp gt 2026-05-20T12:00:00Z") {
		t.Fatalf("missing bridge ts filter: %q", got)
	}
	if strings.Contains(got, "datetime'") {
		t.Fatalf("must use bare ISO timestamp: %q", got)
	}
}

func TestSparkReplicationFilterRolling(t *testing.T) {
	cfg := config.Config{MLS: config.MLSConfig{LocalMirrorRollingMonths: 3}}
	got := SparkReplicationFilter(cfg)
	if !strings.Contains(got, "ModificationTimestamp gt 20") {
		t.Fatalf("missing spark timestamp: %q", got)
	}
}
