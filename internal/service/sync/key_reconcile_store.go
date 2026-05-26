package sync

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	tag, err := s.db.Pool.Exec(ctx, `
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
	var n int64
	err := s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM reconcile_listing_keys
		WHERE run_id = $1 AND dataset_slug = $2
	`, runID, dataset).Scan(&n)
	return n, err
}
