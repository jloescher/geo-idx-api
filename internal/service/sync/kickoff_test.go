package sync

import (
	"context"
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

func TestTryIncrementalKickoffSkipsWhenReplicationChainActive(t *testing.T) {
	k := &Kickoff{cfg: config.Config{MLS: config.MLSConfig{ReplicationFreshnessMinutes: 15}}}
	cursor := SyncCursor{ReplicationInProgress: true}
	if err := k.tryIncrementalKickoff(context.Background(), "bridge", "stellar", "bridge-sync-fetch", "bridge.fetch_page", cursor); err != nil {
		t.Fatal(err)
	}
}

func TestTryIncrementalKickoffPollsWhenStaleAfterReplication(t *testing.T) {
	k := &Kickoff{cfg: config.Config{MLS: config.MLSConfig{ReplicationFreshnessMinutes: 15}}}
	finished := time.Now().Add(-9 * time.Hour)
	cursor := SyncCursor{
		LastModificationTimestamp: &finished,
		LastSyncFinishedAt:        &finished,
	}
	if !k.shouldPollIncremental(cursor) {
		t.Fatal("stale seeded mirror should poll incremental")
	}
	if ReplicationChainActive(cursor) {
		t.Fatal("replication chain should be inactive after bulk load")
	}
}
