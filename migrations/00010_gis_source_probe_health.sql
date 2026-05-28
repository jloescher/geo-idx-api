-- +goose Up
ALTER TABLE gis_source_states
    ADD COLUMN IF NOT EXISTS last_probe_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_probe_ok BOOLEAN,
    ADD COLUMN IF NOT EXISTS last_probe_http_status INT,
    ADD COLUMN IF NOT EXISTS last_probe_error TEXT;

CREATE TABLE IF NOT EXISTS gis_import_uploads (
    id BIGSERIAL PRIMARY KEY,
    source_key TEXT NOT NULL REFERENCES gis_parcel_sources (source_key) ON DELETE CASCADE,
    original_filename TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    error TEXT,
    uploaded_by_user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT gis_import_uploads_status CHECK (status IN ('pending', 'processing', 'done', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_gis_import_uploads_source ON gis_import_uploads (source_key);
CREATE INDEX IF NOT EXISTS idx_gis_import_uploads_status ON gis_import_uploads (status) WHERE status IN ('pending', 'processing');

-- +goose Down
DROP TABLE IF EXISTS gis_import_uploads;
ALTER TABLE gis_source_states
    DROP COLUMN IF EXISTS last_probe_error,
    DROP COLUMN IF EXISTS last_probe_http_status,
    DROP COLUMN IF EXISTS last_probe_ok,
    DROP COLUMN IF EXISTS last_probe_at;
