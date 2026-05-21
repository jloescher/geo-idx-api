package sync

import (
	"testing"
)

func TestReplicationChainActive_inProgress(t *testing.T) {
	if !ReplicationChainActive(SyncCursor{ReplicationInProgress: true}) {
		t.Fatal("expected active when replication_in_progress")
	}
}

func TestReplicationChainActive_nextURL(t *testing.T) {
	next := "https://example/next"
	if !ReplicationChainActive(SyncCursor{ReplicationNextURL: &next}) {
		t.Fatal("expected active when replication_next_url set")
	}
}

func TestReplicationChainActive_idle(t *testing.T) {
	if ReplicationChainActive(SyncCursor{}) {
		t.Fatal("expected inactive when replication finished")
	}
}

func TestReplicationChainActive_emptyNextURL(t *testing.T) {
	empty := "  "
	if ReplicationChainActive(SyncCursor{ReplicationNextURL: &empty}) {
		t.Fatal("expected inactive for blank replication_next_url")
	}
}
