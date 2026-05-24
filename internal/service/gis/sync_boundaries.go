package gis

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/quantyralabs/idx-api/internal/config"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
)

// BoundarySyncService syncs FDOT admin boundaries into gis_cities/counties/zips.
// Persistent GIS tables + monthly parcel refresh deliver instant Leaflet map performance →
// higher visitor engagement before the 3-listing hard gate → more OTP registrations and
// qualified leads while keeping marginal cost at zero.
type BoundarySyncService struct {
	cfg    config.Config
	db     *gisrepo.Repository
	client *ArcGISClient
	logger *slog.Logger
}

// NewBoundarySyncService creates a boundary sync service.
func NewBoundarySyncService(cfg config.Config, repo *gisrepo.Repository, logger *slog.Logger) *BoundarySyncService {
	return &BoundarySyncService{
		cfg:    cfg,
		db:     repo,
		client: NewArcGISClient(cfg.GIS),
		logger: logger,
	}
}

// RunAnnualRefresh syncs all FDOT boundary layers.
func (s *BoundarySyncService) RunAnnualRefresh(ctx context.Context) error {
	gen64, err := s.db.BumpSourceGeneration(ctx, FDOTAdminBoundariesKey)
	if err != nil {
		return fmt.Errorf("bump fdot generation: %w", err)
	}
	gen := int(gen64)
	fp := "fdot_annual_sync"

	if err := s.syncLayer(ctx, FDOTCountiesURL, gen, &fp, s.syncCounties); err != nil {
		return err
	}
	if err := s.syncLayer(ctx, FDOTCitiesURL, gen, &fp, s.syncCities); err != nil {
		return err
	}
	if err := s.syncLayer(ctx, FDOTZipsURL, gen, &fp, s.syncZips); err != nil {
		return err
	}

	if _, err := s.db.DeleteStaleCities(ctx, FDOTAdminBoundariesKey, gen); err != nil {
		return err
	}
	if _, err := s.db.DeleteStaleCounties(ctx, FDOTAdminBoundariesKey, gen); err != nil {
		return err
	}
	if _, err := s.db.DeleteStaleZips(ctx, FDOTAdminBoundariesKey, gen); err != nil {
		return err
	}
	s.logger.Info("gis boundary sync complete", "generation", gen)
	return nil
}

// RunInitialSync syncs boundaries when tables are empty.
func (s *BoundarySyncService) RunInitialSync(ctx context.Context) error {
	cities, counties, zips, err := s.db.CountBoundaries(ctx)
	if err != nil {
		return err
	}
	if cities > 0 || counties > 0 || zips > 0 {
		s.logger.Info("gis initial boundary sync skipped", "cities", cities, "counties", counties, "zips", zips)
		return nil
	}
	return s.RunAnnualRefresh(ctx)
}

type boundaryUpserter func(ctx context.Context, feats []ArcGISFeature, gen int, fp *string) error

func (s *BoundarySyncService) syncLayer(ctx context.Context, endpoint string, gen int, fp *string, upsert boundaryUpserter) error {
	offset := 0
	pageSize := s.cfg.GIS.SyncPageSize
	if pageSize <= 0 {
		pageSize = 2000
	}
	for {
		body, err := s.client.FetchLayerPage(endpoint, "1=1", offset, pageSize)
		if err != nil {
			return err
		}
		page, err := ParseFeatureCollection(body)
		if err != nil {
			return err
		}
		if err := upsert(ctx, page.Features, gen, fp); err != nil {
			return err
		}
		s.logger.Info("gis boundary page synced", "endpoint", endpoint, "offset", offset, "features", len(page.Features))
		if !page.ExceededTransferLimit && len(page.Features) < pageSize {
			break
		}
		offset += pageSize
	}
	return nil
}

func (s *BoundarySyncService) syncCounties(ctx context.Context, feats []ArcGISFeature, gen int, fp *string) error {
	chunk := s.cfg.GIS.SyncUpsertChunk
	if chunk <= 0 {
		chunk = 500
	}
	var batch []gisrepo.CountyRow
	for _, feat := range feats {
		row, err := ExtractCountyRow(feat, gen, fp)
		if err != nil {
			continue
		}
		batch = append(batch, row)
		if len(batch) >= chunk {
			if err := s.db.BulkUpsertCounties(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		return s.db.BulkUpsertCounties(ctx, batch)
	}
	return nil
}

func (s *BoundarySyncService) syncCities(ctx context.Context, feats []ArcGISFeature, gen int, fp *string) error {
	chunk := s.cfg.GIS.SyncUpsertChunk
	if chunk <= 0 {
		chunk = 500
	}
	var batch []gisrepo.CityRow
	for _, feat := range feats {
		row, err := ExtractCityRow(feat, gen, fp)
		if err != nil {
			continue
		}
		batch = append(batch, row)
		if len(batch) >= chunk {
			if err := s.db.BulkUpsertCities(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		return s.db.BulkUpsertCities(ctx, batch)
	}
	return nil
}

func (s *BoundarySyncService) syncZips(ctx context.Context, feats []ArcGISFeature, gen int, fp *string) error {
	chunk := s.cfg.GIS.SyncUpsertChunk
	if chunk <= 0 {
		chunk = 500
	}
	var batch []gisrepo.ZipRow
	for _, feat := range feats {
		row, err := ExtractZipRow(feat, gen, fp)
		if err != nil {
			continue
		}
		batch = append(batch, row)
		if len(batch) >= chunk {
			if err := s.db.BulkUpsertZips(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		return s.db.BulkUpsertZips(ctx, batch)
	}
	return nil
}

// InitialSyncService coordinates first-run parcel + boundary sync.
type InitialSyncService struct {
	parcels    *ParcelSyncService
	boundaries *BoundarySyncService
	logger     *slog.Logger
}

// NewInitialSyncService creates the combined initial sync coordinator.
func NewInitialSyncService(parcels *ParcelSyncService, boundaries *BoundarySyncService, logger *slog.Logger) *InitialSyncService {
	return &InitialSyncService{parcels: parcels, boundaries: boundaries, logger: logger}
}

// Run executes initial sync for empty persistent tables.
func (s *InitialSyncService) Run(ctx context.Context) error {
	if err := s.parcels.RunInitialSync(ctx); err != nil {
		return err
	}
	return s.boundaries.RunInitialSync(ctx)
}
