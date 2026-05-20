package sync

import (
	"testing"
	"time"
)

func TestMergeCursorPatch_replicationInProgressWithoutNextURL(t *testing.T) {
	next := "https://example/Property?$skiptoken=1"
	c := SyncCursor{
		DatasetSlug:           "beaches",
		ReplicationNextURL:    &next,
		ReplicationInProgress: false,
	}
	inProgress := true
	merged := mergeCursorPatch(c, CursorPatch{ReplicationInProgress: &inProgress}, time.Now())
	if !merged.ReplicationInProgress {
		t.Fatal("expected replication_in_progress true")
	}
	if merged.ReplicationNextURL == nil || *merged.ReplicationNextURL != next {
		t.Fatal("expected replication_next_url unchanged until finalize")
	}
}

func TestMergeCursorPatch_applyReplicationStateClearsNextURL(t *testing.T) {
	prev := "https://example/old"
	c := SyncCursor{
		DatasetSlug:           "stellar",
		ReplicationNextURL:    &prev,
		ReplicationInProgress: true,
	}
	inProgress := false
	merged := mergeCursorPatch(c, CursorPatch{
		ApplyReplicationState: true,
		ReplicationNextURL:    nil,
		ReplicationInProgress: &inProgress,
	}, time.Now())
	if merged.ReplicationInProgress {
		t.Fatal("expected replication_in_progress false")
	}
	if merged.ReplicationNextURL != nil {
		t.Fatal("expected replication_next_url cleared")
	}
}

func TestMergeCursorPatch_maxBridgeTsMonotonic(t *testing.T) {
	older := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	c := SyncCursor{LastBridgeModificationTimestamp: &newer}
	merged := mergeCursorPatch(c, CursorPatch{MaxBridgeTs: &older}, time.Now())
	if !merged.LastBridgeModificationTimestamp.Equal(newer) {
		t.Fatal("expected max timestamp to remain newer value")
	}
	merged = mergeCursorPatch(c, CursorPatch{MaxBridgeTs: &newer}, time.Now())
	if merged.LastBridgeModificationTimestamp == nil || !merged.LastBridgeModificationTimestamp.Equal(newer) {
		t.Fatal("expected max timestamp updated to newer")
	}
}

func TestMergeCursorPatch_markSyncFinishedClearsWindow(t *testing.T) {
	window := time.Now().UTC()
	c := SyncCursor{IncrementalWindowEnd: &window}
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	merged := mergeCursorPatch(c, CursorPatch{MarkSyncFinished: true}, now)
	if merged.IncrementalWindowEnd != nil {
		t.Fatal("expected incremental_window_end cleared")
	}
	if merged.LastSyncFinishedAt == nil || !merged.LastSyncFinishedAt.Equal(now) {
		t.Fatal("expected last_sync_finished_at set")
	}
}
