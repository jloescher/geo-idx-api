package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/quantyralabs/idx-api/internal/repository"
)

const reconcileKeyInsertChunk = 500

// ReconcileKeyStore persists upstream listing keys for a reconcile run.
type ReconcileKeyStore struct {
	db *repository.DB
}

func NewReconcileKeyStore(db *repository.DB) *ReconcileKeyStore {
	return &ReconcileKeyStore{db: db}
}

func (s *ReconcileKeyStore) InsertKeys(ctx context.Context, runID uuid.UUID, dataset string, keys []string) error {
	for i := 0; i < len(keys); i += reconcileKeyInsertChunk {
		end := i + reconcileKeyInsertChunk
		if end > len(keys) {
			end = len(keys)
		}
		if err := s.insertChunk(ctx, runID, dataset, keys[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func (s *ReconcileKeyStore) insertChunk(ctx context.Context, runID uuid.UUID, dataset string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, key := range keys {
		if key == "" {
			continue
		}
		batch.Queue(`
			INSERT INTO reconcile_listing_keys (run_id, dataset_slug, listing_key)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING
		`, runID, dataset, key)
	}
	if batch.Len() == 0 {
		return nil
	}
	br := s.db.Pool.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert reconcile key: %w", err)
		}
	}
	return nil
}

// DeleteStaleMirrorRows removes mirror rows not present in the reconcile key set.
func (s *ReconcileKeyStore) DeleteStaleMirrorRows(ctx context.Context, runID uuid.UUID, dataset string) (int64, error) {
	return s.deleteStaleMirrorRows(ctx, s.db.Pool, runID, dataset)
}

func (s *ReconcileKeyStore) DeleteStaleMirrorRowsTx(ctx context.Context, tx pgx.Tx, runID uuid.UUID, dataset string) (int64, error) {
	return s.deleteStaleMirrorRows(ctx, tx, runID, dataset)
}

type execer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (s *ReconcileKeyStore) deleteStaleMirrorRows(ctx context.Context, db execer, runID uuid.UUID, dataset string) (int64, error) {
	tag, err := db.Exec(ctx, `
		DELETE FROM listings l
		WHERE l.dataset_slug = $1
		  AND NOT EXISTS (
		    SELECT 1 FROM reconcile_listing_keys r
		    WHERE r.run_id = $2
		      AND r.dataset_slug = l.dataset_slug
		      AND r.listing_key = l.listing_key
		  )
	`, dataset, runID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *ReconcileKeyStore) PurgeRun(ctx context.Context, runID uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM reconcile_listing_keys WHERE run_id = $1`, runID)
	return err
}

func (s *ReconcileKeyStore) CountKeys(ctx context.Context, runID uuid.UUID, dataset string) (int64, error) {
	return s.countKeys(ctx, s.db.Pool, runID, dataset)
}

func (s *ReconcileKeyStore) CountKeysTx(ctx context.Context, tx pgx.Tx, runID uuid.UUID, dataset string) (int64, error) {
	return s.countKeys(ctx, tx, runID, dataset)
}

type rowScanner interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *ReconcileKeyStore) countKeys(ctx context.Context, q rowScanner, runID uuid.UUID, dataset string) (int64, error) {
	var n int64
	err := q.QueryRow(ctx, `
		SELECT COUNT(*) FROM reconcile_listing_keys
		WHERE run_id = $1 AND dataset_slug = $2
	`, runID, dataset).Scan(&n)
	return n, err
}

// CountMirrorListings returns all mirror rows for a dataset (reconcile deletes any status).
func (s *ReconcileKeyStore) CountMirrorListings(ctx context.Context, dataset string) (int64, error) {
	var n int64
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM listings WHERE dataset_slug = $1
	`, dataset).Scan(&n)
	return n, err
}

func (s *ReconcileKeyStore) RecordRunStart(ctx context.Context, runID uuid.UUID, dataset, provider string, mirrorCount int64) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO reconcile_runs (run_id, dataset_slug, provider, status, mirror_count_before, started_at)
		VALUES ($1, $2, $3, 'running', $4, NOW())
		ON CONFLICT (run_id) DO NOTHING
	`, runID, dataset, provider, mirrorCount)
	return err
}

func (s *ReconcileKeyStore) RecordRunComplete(ctx context.Context, runID uuid.UUID, keysSeen, rowsDeleted int64) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE reconcile_runs
		SET status = 'completed',
		    keys_seen = $2,
		    rows_deleted = $3,
		    finished_at = NOW()
		WHERE run_id = $1
	`, runID, keysSeen, rowsDeleted)
	return err
}

func (s *ReconcileKeyStore) RecordRunFailed(ctx context.Context, runID uuid.UUID, errMsg string) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE reconcile_runs
		SET status = 'failed',
		    error_message = $2,
		    finished_at = NOW()
		WHERE run_id = $1
	`, runID, errMsg)
	return err
}

// PurgeStaleRuns removes reconcile_listing_keys rows older than the retention window.
func (s *ReconcileKeyStore) PurgeStaleStaging(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	tag, err := s.db.Pool.Exec(ctx, `
		DELETE FROM reconcile_listing_keys WHERE created_at < $1
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
