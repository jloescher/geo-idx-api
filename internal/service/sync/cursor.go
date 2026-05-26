package sync

import (
	"context"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/repository"
)

// SyncCursor tracks replication/incremental state per dataset (listing_sync_cursors).
type SyncCursor struct {
	DatasetSlug               string
	LastModificationTimestamp *time.Time
	IncrementalWindowEnd      *time.Time
	ReplicationNextURL        *string
	ReplicationInProgress     bool
	LastSyncFinishedAt        *time.Time
	UpdatedAt                 time.Time
}

// CursorPatch updates cursor fields after a fetch/persist cycle.
type CursorPatch struct {
	ApplyReplicationState bool
	ReplicationNextURL    *string
	ReplicationInProgress *bool
	MaxModificationTs     *time.Time
	IncrementalWindowEnd  *time.Time
	MarkSyncFinished      bool
}

// CursorStore reads/writes listing_sync_cursors.
type CursorStore struct {
	db *repository.DB
}

func NewCursorStore(db *repository.DB) *CursorStore {
	return &CursorStore{db: db}
}

func (s *CursorStore) ForDataset(ctx context.Context, dataset string) (SyncCursor, error) {
	var c SyncCursor
	c.DatasetSlug = dataset
	err := s.db.Pool.QueryRow(ctx, `
		INSERT INTO listing_sync_cursors (dataset_slug, replication_in_progress, created_at, updated_at)
		VALUES ($1, FALSE, NOW(), NOW())
		ON CONFLICT (dataset_slug) DO UPDATE SET updated_at = listing_sync_cursors.updated_at
		RETURNING dataset_slug, last_modification_timestamp, incremental_window_end,
			replication_next_url, replication_in_progress, last_sync_finished_at, updated_at
	`, dataset).Scan(
		&c.DatasetSlug,
		&c.LastModificationTimestamp,
		&c.IncrementalWindowEnd,
		&c.ReplicationNextURL,
		&c.ReplicationInProgress,
		&c.LastSyncFinishedAt,
		&c.UpdatedAt,
	)
	return c, err
}

func (s *CursorStore) MirrorSeeded(ctx context.Context, dataset string) (bool, error) {
	var n int
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM listings WHERE dataset_slug = $1 LIMIT 1
	`, dataset).Scan(&n)
	return n > 0, err
}

// ReplicationChainActive is true while replication paging is driven by persist finalize (not kickoff).
func ReplicationChainActive(c SyncCursor) bool {
	if c.ReplicationInProgress {
		return true
	}
	if c.ReplicationNextURL != nil && strings.TrimSpace(*c.ReplicationNextURL) != "" {
		return true
	}
	return false
}

// ShouldKickoffReplication returns whether kickoff may enqueue the initial replication fetch.
// In-progress chains continue via persist finalize only.
func (s *CursorStore) ShouldKickoffReplication(ctx context.Context, c SyncCursor) (bool, error) {
	if ReplicationChainActive(c) {
		return false, nil
	}
	seeded, err := s.MirrorSeeded(ctx, c.DatasetSlug)
	if err != nil {
		return false, err
	}
	return !seeded, nil
}

func (s *CursorStore) ShouldRunIncremental(c SyncCursor) bool {
	if c.ReplicationInProgress {
		return false
	}
	return c.LastModificationTimestamp != nil
}

// mergeCursorPatch computes the row state after applying a patch (used by ApplyPatch and tests).
func mergeCursorPatch(c SyncCursor, patch CursorPatch, now time.Time) SyncCursor {
	out := c
	if patch.ReplicationInProgress != nil {
		out.ReplicationInProgress = *patch.ReplicationInProgress
	}
	if patch.ApplyReplicationState {
		out.ReplicationNextURL = patch.ReplicationNextURL
	}
	if patch.IncrementalWindowEnd != nil {
		out.IncrementalWindowEnd = patch.IncrementalWindowEnd
	}
	if patch.MaxModificationTs != nil {
		if out.LastModificationTimestamp == nil || patch.MaxModificationTs.After(*out.LastModificationTimestamp) {
			t := *patch.MaxModificationTs
			out.LastModificationTimestamp = &t
		}
	}
	if patch.MarkSyncFinished {
		out.LastSyncFinishedAt = &now
		out.IncrementalWindowEnd = nil
	}
	return out
}

func (s *CursorStore) ApplyPatch(ctx context.Context, dataset string, patch CursorPatch) error {
	c, err := s.ForDataset(ctx, dataset)
	if err != nil {
		return err
	}

	merged := mergeCursorPatch(c, patch, time.Now())

	_, err = s.db.Pool.Exec(ctx, `
		UPDATE listing_sync_cursors SET
			last_modification_timestamp = $2,
			incremental_window_end = $3,
			replication_next_url = $4,
			replication_in_progress = $5,
			last_sync_finished_at = $6,
			updated_at = NOW()
		WHERE dataset_slug = $1
	`, dataset,
		merged.LastModificationTimestamp,
		merged.IncrementalWindowEnd,
		merged.ReplicationNextURL,
		merged.ReplicationInProgress,
		merged.LastSyncFinishedAt,
	)
	return err
}
