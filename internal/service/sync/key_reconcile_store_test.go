package sync

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/quantyralabs/idx-api/internal/repository"
)

func TestReconcileKeyStoreDeleteStaleMirrorRows(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	db, err := repository.NewFromDSN(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	runID := uuid.New()
	dataset := "reconcile_test_" + runID.String()[:8]
	store := NewReconcileKeyStore(db)

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO listings (dataset_slug, listing_key, mls_listing_id, standard_status, list_price, modification_timestamp, raw_data)
		VALUES
		  ($1, 'keep-me', 'L1', 'active', 100000, NOW(), '{}'::jsonb),
		  ($1, 'delete-me', 'L2', 'pending', 200000, NOW(), '{}'::jsonb)
	`, dataset)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM listings WHERE dataset_slug = $1`, dataset)
		_, _ = db.Pool.Exec(context.Background(), `DELETE FROM reconcile_listing_keys WHERE dataset_slug = $1`, dataset)
	})

	if err := store.InsertKeys(ctx, runID, dataset, []string{"keep-me", "keep-me"}); err != nil {
		t.Fatal(err)
	}

	deleted, err := store.DeleteStaleMirrorRows(ctx, runID, dataset)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d want 1", deleted)
	}

	var remaining int
	if err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM listings WHERE dataset_slug = $1
	`, dataset).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 1 {
		t.Fatalf("remaining listings = %d want 1", remaining)
	}

	if err := store.PurgeRun(ctx, runID); err != nil {
		t.Fatal(err)
	}
}
