package gis

import (
	"context"
	"log/slog"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
)

// PersistentGISService orchestrates ArcGIS → PostGIS sync for parcels and boundaries.
// Persistent statewide parcel sync (including Osceola) = instant complete Leaflet maps for all
// pilot counties → maximum engagement before hard gate → more OTP leads.
type PersistentGISService struct {
	cfg        config.Config
	parcels    *ParcelSyncService
	boundaries *BoundarySyncService
	initial    *InitialSyncService
}

// NewPersistentGISService wires parcel and boundary sync for queue handlers.
func NewPersistentGISService(cfg config.Config, repo *gisrepo.Repository, q *queue.Client, logger *slog.Logger) *PersistentGISService {
	if logger == nil {
		logger = slog.Default()
	}
	parcels := NewParcelSyncService(cfg, repo, q, logger)
	boundaries := NewBoundarySyncService(cfg, repo, logger)
	return &PersistentGISService{
		cfg:        cfg,
		parcels:    parcels,
		boundaries: boundaries,
		initial:    NewInitialSyncService(parcels, boundaries, repo, logger),
	}
}

// RunMonthlyParcelRefresh kicks off statewide paginate sync plus Pinellas/Hillsborough failovers.
func (s *PersistentGISService) RunMonthlyParcelRefresh(ctx context.Context) error {
	return s.parcels.RunMonthlyRefresh(ctx)
}

// RunInitialSync runs boundary gap-fill then parcel kickoff on a fresh database.
func (s *PersistentGISService) RunInitialSync(ctx context.Context) error {
	return s.initial.Run(ctx)
}

// RunAnnualBoundariesRefresh syncs FDOT cities, counties, and zips.
func (s *PersistentGISService) RunAnnualBoundariesRefresh(ctx context.Context) error {
	return s.boundaries.RunAnnualRefresh(ctx)
}

// RunZipSync refreshes FDOT zip boundaries only (layer 8).
func (s *PersistentGISService) RunZipSync(ctx context.Context) error {
	return s.boundaries.RunZipSync(ctx)
}

// SyncParcelPage fetches one ArcGIS page and upserts parcel rows.
func (s *PersistentGISService) SyncParcelPage(ctx context.Context, args ParcelSyncPageArgs) error {
	return s.parcels.SyncPage(ctx, args)
}

// KickoffCounty enqueues parcel sync for a single county failover source.
func (s *PersistentGISService) KickoffCounty(ctx context.Context, countySlug string) error {
	return s.parcels.KickoffCounty(ctx, countySlug)
}
