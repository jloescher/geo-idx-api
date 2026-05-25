package gisrepo

import "context"

// HasActiveParcelSyncJob reports whether the queue already has a pending or in-flight
// gis.parcel_sync_page job for the given source (avoids duplicate kickoffs).
func (r *Repository) HasActiveParcelSyncJob(ctx context.Context, queueName, sourceKey string) (bool, error) {
	if queueName == "" || sourceKey == "" {
		return false, nil
	}
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM jobs
			WHERE queue = $1
			  AND payload::jsonb->>'type' = 'gis.parcel_sync_page'
			  AND payload::jsonb->'args'->>'source_key' = $2
			  AND (
			    reserved_at IS NULL
			    OR reserved_at > EXTRACT(EPOCH FROM NOW())::bigint - 7200
			  )
		)`, queueName, sourceKey).Scan(&exists)
	return exists, err
}
