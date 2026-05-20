package sync

import (
	"strings"
	"testing"

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
	if !strings.Contains(got, activePendingStatusFilter) {
		t.Fatalf("missing status filter: %q", got)
	}
	if !strings.Contains(got, "ModificationTimestamp gt datetime'") {
		t.Fatalf("missing rolling timestamp: %q", got)
	}
}

func TestSparkReplicationFilterRolling(t *testing.T) {
	cfg := config.Config{MLS: config.MLSConfig{LocalMirrorRollingMonths: 3}}
	got := SparkReplicationFilter(cfg)
	if !strings.Contains(got, "ModificationTimestamp gt 20") {
		t.Fatalf("missing spark timestamp: %q", got)
	}
}
