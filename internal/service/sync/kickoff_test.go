package sync

import (
	"testing"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestShouldPollIncrementalAfterFreshSync(t *testing.T) {
	k := &Kickoff{cfg: config.Config{MLS: config.MLSConfig{ReplicationFreshnessMinutes: 15}}}
	finished := time.Now().Add(-5 * time.Minute)
	cursor := SyncCursor{LastSyncFinishedAt: &finished}
	if k.shouldPollIncremental(cursor) {
		t.Fatal("expected no poll within freshness window")
	}
}

func TestShouldPollIncrementalWhenStale(t *testing.T) {
	k := &Kickoff{cfg: config.Config{MLS: config.MLSConfig{ReplicationFreshnessMinutes: 15}}}
	finished := time.Now().Add(-20 * time.Minute)
	cursor := SyncCursor{LastSyncFinishedAt: &finished}
	if !k.shouldPollIncremental(cursor) {
		t.Fatal("expected poll after freshness window")
	}
}

func TestShouldPollIncrementalNeverSynced(t *testing.T) {
	k := &Kickoff{cfg: config.Config{MLS: config.MLSConfig{ReplicationFreshnessMinutes: 15}}}
	if !k.shouldPollIncremental(SyncCursor{}) {
		t.Fatal("expected poll when last_sync_finished_at is nil")
	}
}

func TestShouldPollIncrementalSkipsDuringReplication(t *testing.T) {
	k := &Kickoff{cfg: config.Config{MLS: config.MLSConfig{ReplicationFreshnessMinutes: 15}}}
	next := "https://example/next"
	cursor := SyncCursor{ReplicationNextURL: &next}
	if k.shouldPollIncremental(cursor) {
		t.Fatal("expected no incremental during replication paging")
	}
}
