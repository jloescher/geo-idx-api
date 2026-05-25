package gis

import (
	"context"
	"log/slog"

	"github.com/quantyralabs/idx-api/internal/queue"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
)

// BootstrapAction names a queue job to enqueue for GIS gap-fill or fresh bootstrap.
type BootstrapAction struct {
	Name    string
	JobType string
}

// PlanBootstrapActions returns queue jobs to run for the current layer counts.
// Fresh DB (all empty) enqueues a single initial sync; partial DB enqueues only missing layers.
func PlanBootstrapActions(counts gisrepo.LayerCounts) []BootstrapAction {
	if counts.AllEmpty() {
		return []BootstrapAction{{Name: "gis-initial-sync", JobType: queue.TypeGISInitialSync}}
	}
	var actions []BootstrapAction
	if counts.Parcels == 0 {
		actions = append(actions, BootstrapAction{Name: "gis-parcel-gap-fill", JobType: queue.TypeGISMonthlyParcelRefresh})
	}
	if counts.Zips == 0 {
		actions = append(actions, BootstrapAction{Name: "gis-zip-gap-fill", JobType: queue.TypeGISZipSync})
	}
	if counts.Cities == 0 || counts.Counties == 0 {
		actions = append(actions, BootstrapAction{Name: "gis-boundary-gap-fill", JobType: queue.TypeGISAnnualBoundariesRefresh})
	}
	return actions
}

// SyncCoordinator orchestrates gap-fill and fresh-database GIS bootstrap.
type SyncCoordinator struct {
	parcels    *ParcelSyncService
	boundaries *BoundarySyncService
	repo       *gisrepo.Repository
	logger     *slog.Logger
}

// NewSyncCoordinator wires parcel and boundary sync services for bootstrap orchestration.
func NewSyncCoordinator(parcels *ParcelSyncService, boundaries *BoundarySyncService, repo *gisrepo.Repository, logger *slog.Logger) *SyncCoordinator {
	if logger == nil {
		logger = slog.Default()
	}
	return &SyncCoordinator{parcels: parcels, boundaries: boundaries, repo: repo, logger: logger}
}

// LayerCounts loads current persistent GIS row totals.
func (c *SyncCoordinator) LayerCounts(ctx context.Context) (gisrepo.LayerCounts, error) {
	return c.repo.LoadLayerCounts(ctx)
}

// RunGapFill enqueues or runs sync for any layer with zero rows (partial DB recovery).
func (c *SyncCoordinator) RunGapFill(ctx context.Context) error {
	counts, err := c.repo.LoadLayerCounts(ctx)
	if err != nil {
		return err
	}
	if counts.Parcels == 0 {
		if err := c.parcels.Kickoff(ctx); err != nil {
			return err
		}
	}
	if err := c.boundaries.RunGapFill(ctx); err != nil {
		return err
	}
	return nil
}

// InitialSyncService runs the first-time GIS bootstrap (boundaries before parcels).
type InitialSyncService struct {
	coordinator *SyncCoordinator
	logger      *slog.Logger
}

// NewInitialSyncService wires coordinator for fresh-database bootstrap.
func NewInitialSyncService(parcels *ParcelSyncService, boundaries *BoundarySyncService, repo *gisrepo.Repository, logger *slog.Logger) *InitialSyncService {
	if logger == nil {
		logger = slog.Default()
	}
	return &InitialSyncService{
		coordinator: NewSyncCoordinator(parcels, boundaries, repo, logger),
		logger:      logger,
	}
}

// Run executes boundaries inline first, then kicks off paginated parcel sync.
func (s *InitialSyncService) Run(ctx context.Context) error {
	before, err := s.coordinator.repo.LoadLayerCounts(ctx)
	if err != nil {
		return err
	}
	if err := s.coordinator.boundaries.RunBootstrapBoundaries(ctx); err != nil {
		return err
	}
	if err := s.coordinator.parcels.RunInitialSync(ctx); err != nil {
		return err
	}
	after, err := s.coordinator.repo.LoadLayerCounts(ctx)
	if err != nil {
		return err
	}
	s.logger.Info("gis bootstrap complete",
		"parcels_before", before.Parcels, "parcels_after", after.Parcels,
		"parcels_kickoff_async", after.Parcels == 0,
		"cities_before", before.Cities, "cities_after", after.Cities,
		"counties_before", before.Counties, "counties_after", after.Counties,
		"zips_before", before.Zips, "zips_after", after.Zips)
	return nil
}
