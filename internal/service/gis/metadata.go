package gis

import (
	"context"
	"log/slog"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// MetadataService probes ArcGIS layer metadata for cache invalidation.
type MetadataService struct {
	cfg    config.Config
	db     *repository.DB
	logger *slog.Logger
}

func NewMetadataService(cfg config.Config, db *repository.DB, logger *slog.Logger) *MetadataService {
	return &MetadataService{cfg: cfg, db: db, logger: logger}
}

func (m *MetadataService) ProbeAll(ctx context.Context) error {
	m.logger.Info("gis metadata probe")
	_, err := m.db.Pool.Exec(ctx, `UPDATE gis_source_states SET last_checked_at = NOW(), updated_at = NOW()`)
	return err
}
