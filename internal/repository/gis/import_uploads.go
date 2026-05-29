package gisrepo

import "context"

// SetImportUploadStatus updates shapefile import lifecycle on gis_import_uploads.
func (r *Repository) SetImportUploadStatus(ctx context.Context, uploadID int64, status, errMsg string) error {
	if uploadID <= 0 {
		return nil
	}
	if len(errMsg) > 2000 {
		errMsg = errMsg[:2000]
	}
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE gis_import_uploads
		SET status = $2,
		    error = NULLIF($3, ''),
		    updated_at = NOW()
		WHERE id = $1
	`, uploadID, status, errMsg)
	return err
}
