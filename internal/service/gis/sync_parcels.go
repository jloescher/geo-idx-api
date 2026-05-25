package gis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
)

// ParcelSyncService syncs parcel polygons from ArcGIS into gis_parcels.
type ParcelSyncService struct {
	cfg    config.Config
	db     *gisrepo.Repository
	client *ArcGISClient
	queue  *queue.Client
	logger *slog.Logger
}

// NewParcelSyncService creates a parcel sync worker service.
func NewParcelSyncService(cfg config.Config, repo *gisrepo.Repository, q *queue.Client, logger *slog.Logger) *ParcelSyncService {
	return &ParcelSyncService{
		cfg:    cfg,
		db:     repo,
		client: NewArcGISClient(cfg.GIS),
		queue:  q,
		logger: logger,
	}
}

// ParcelSyncPageArgs is the payload for gis.parcel_sync_page jobs.
type ParcelSyncPageArgs struct {
	SourceKey      string   `json:"source_key"`
	County         string   `json:"county"`
	QueryURL       string   `json:"query_url"`
	SyncMode       string   `json:"sync_mode"`
	ArcGISWhere    string   `json:"arcgis_where,omitempty"`
	ResponseFormat string   `json:"response_format,omitempty"`
	BBoxWest       float64  `json:"bbox_west"`
	BBoxSouth      float64  `json:"bbox_south"`
	BBoxEast       float64  `json:"bbox_east"`
	BBoxNorth      float64  `json:"bbox_north"`
	ParcelIDFields []string `json:"parcel_id_fields,omitempty"`
	Offset         int      `json:"offset"`
	Generation     int      `json:"generation"`
}

// RunMonthlyRefresh kicks off a full parcel sync for all configured sources.
func (s *ParcelSyncService) RunMonthlyRefresh(ctx context.Context) error {
	return s.kickoff(ctx, "")
}

// RunInitialSync runs parcel sync when tables are empty (same as monthly).
func (s *ParcelSyncService) RunInitialSync(ctx context.Context) error {
	empty, err := s.db.IsPersistentEmpty(ctx)
	if err != nil {
		return err
	}
	if !empty {
		parcels, err := s.db.CountParcels(ctx, "")
		if err != nil {
			return err
		}
		if parcels > 0 {
			s.logger.Info("gis initial parcel sync skipped, data present", "parcels", parcels)
			return nil
		}
	}
	return s.kickoff(ctx, "")
}

// Kickoff enqueues paginated parcel sync jobs for all configured sources.
func (s *ParcelSyncService) Kickoff(ctx context.Context) error {
	return s.kickoff(ctx, "")
}

// KickoffCounty enqueues parcel sync jobs for a single county slug.
func (s *ParcelSyncService) KickoffCounty(ctx context.Context, countySlug string) error {
	if countySlug == "" {
		return fmt.Errorf("county slug required")
	}
	found := false
	for _, spec := range s.parcelSyncTargets(countySlug) {
		found = true
		_ = spec
	}
	if !found {
		return fmt.Errorf("unknown or disabled county: %s", countySlug)
	}
	return s.kickoff(ctx, countySlug)
}

func (s *ParcelSyncService) kickoff(ctx context.Context, countyFilter string) error {
	targets := s.parcelSyncTargets(countyFilter)
	if len(targets) == 0 {
		return fmt.Errorf("no parcel sync targets for county=%q", countyFilter)
	}
	if err := s.db.EnsureParcelSourceCatalog(ctx, parcelSourceRowsFromSpecs(ParcelSourceCatalog())); err != nil {
		return fmt.Errorf("ensure parcel source catalog: %w", err)
	}
	seen := map[string]int{}
	for _, spec := range targets {
		if err := s.db.EnsureSourceState(ctx, spec.SourceKey); err != nil {
			return fmt.Errorf("ensure source state %s: %w", spec.SourceKey, err)
		}
		if _, ok := seen[spec.SourceKey]; !ok {
			gen, err := s.db.BumpSourceGeneration(ctx, spec.SourceKey)
			if err != nil {
				return fmt.Errorf("bump generation %s: %w", spec.SourceKey, err)
			}
			seen[spec.SourceKey] = int(gen)
		}
		gen := seen[spec.SourceKey]
		args := argsFromSpec(spec, gen, 0)
		if err := s.enqueuePage(ctx, args); err != nil {
			return err
		}
	}
	return nil
}

func (s *ParcelSyncService) parcelSyncTargets(countyFilter string) []ParcelSourceSpec {
	var out []ParcelSourceSpec
	for _, spec := range EnabledParcelSourcesForConfig(s.cfg.GIS) {
		if countyFilter != "" && spec.CountySlug != countyFilter {
			continue
		}
		out = append(out, spec)
	}
	return out
}

func argsFromSpec(spec ParcelSourceSpec, gen, offset int) ParcelSyncPageArgs {
	return ParcelSyncPageArgs{
		SourceKey:      spec.SourceKey,
		County:         spec.CountySlug,
		QueryURL:       spec.QueryURL,
		SyncMode:       spec.SyncMode,
		ArcGISWhere:    spec.ArcGISWhere,
		ResponseFormat: spec.ResponseFormat,
		BBoxWest:       spec.SyncBBox.West,
		BBoxSouth:      spec.SyncBBox.South,
		BBoxEast:       spec.SyncBBox.East,
		BBoxNorth:      spec.SyncBBox.North,
		ParcelIDFields: spec.ParcelIDFields,
		Offset:         offset,
		Generation:     gen,
	}
}

// SyncPage fetches one ArcGIS page and upserts parcel rows.
func (s *ParcelSyncService) SyncPage(ctx context.Context, args ParcelSyncPageArgs) error {
	spec := ParcelSourceSpec{
		SourceKey:      args.SourceKey,
		CountySlug:     args.County,
		QueryURL:       args.QueryURL,
		SyncMode:       args.SyncMode,
		ArcGISWhere:    args.ArcGISWhere,
		ResponseFormat: args.ResponseFormat,
		ParcelIDFields: args.ParcelIDFields,
		SyncBBox: BBox{
			West: args.BBoxWest, South: args.BBoxSouth,
			East: args.BBoxEast, North: args.BBoxNorth,
		},
	}
	if spec.SyncMode == "" {
		spec.SyncMode = SyncModeBBox
	}
	if spec.ResponseFormat == "" {
		spec.ResponseFormat = FormatGeoJSON
	}
	bbox := spec.SyncBBox
	if bbox.West == 0 && bbox.South == 0 && bbox.East == 0 && bbox.North == 0 {
		bbox = countyBBox(args.County)
	}

	body, err := s.client.FetchParcelPage(spec, bbox, args.Offset, s.cfg.GIS.SyncPageSize)
	if err != nil {
		return err
	}
	page, err := ParseFeatureCollection(body)
	if err != nil {
		return err
	}
	if len(page.Features) == 0 && args.Offset == 0 {
		return fmt.Errorf("gis parcel sync empty first page source=%s county=%s", args.SourceKey, args.County)
	}
	fp := ""
	fpStr := fp
	chunk := s.cfg.GIS.SyncUpsertChunk
	if chunk <= 0 {
		chunk = 500
	}
	var batch []gisrepo.ParcelRow
	statewide := IsStatewideCadastralSource(args.SourceKey)
	for _, feat := range page.Features {
		county := args.County
		if statewide {
			coNo, ok := CONOFromProperties(feat.Properties)
			if !ok {
				continue
			}
			slug, ok := CountySlugFromCONO(coNo)
			if !ok || !IsMLSPilotCounty(slug) {
				continue
			}
			county = slug
		}
		row, err := ExtractParcelRow(feat, args.SourceKey, county, args.Generation, &fpStr, args.ParcelIDFields)
		if err != nil {
			continue
		}
		batch = append(batch, row)
		if len(batch) >= chunk {
			if err := s.db.BulkUpsertParcels(ctx, batch, chunk); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := s.db.BulkUpsertParcels(ctx, batch, chunk); err != nil {
			return err
		}
	}
	s.logger.Info("gis parcel page synced",
		"source", args.SourceKey, "county", args.County, "offset", args.Offset, "features", len(page.Features))

	pageSize := s.cfg.GIS.SyncPageSize
	if pageSize <= 0 {
		pageSize = 2000
	}
	if page.ExceededTransferLimit || len(page.Features) >= pageSize {
		next := args
		next.Offset = args.Offset + pageSize
		return s.enqueuePage(ctx, next)
	}
	staleCounty := args.County
	if statewide {
		staleCounty = ""
	}
	deleted, err := s.db.DeleteStaleParcels(ctx, args.SourceKey, staleCounty, args.Generation)
	if err != nil {
		return err
	}
	if err := s.db.TouchParcelSourceSynced(ctx, args.SourceKey); err != nil {
		s.logger.Warn("touch parcel source synced", "source", args.SourceKey, "error", err)
	}
	s.logger.Info("gis parcel sync finalized", "source", args.SourceKey, "county", args.County, "generation", args.Generation, "deleted_stale", deleted)
	return nil
}

func (s *ParcelSyncService) enqueuePage(ctx context.Context, args ParcelSyncPageArgs) error {
	if s.queue == nil {
		return s.SyncPage(ctx, args)
	}
	queueName := s.cfg.GIS.SyncQueue
	if queueName == "" {
		queueName = "default"
	}
	_, err := s.queue.Enqueue(ctx, queueName, queue.TypeGISParcelSyncPage, args, 0)
	return err
}

func parcelSourceRowsFromSpecs(specs []ParcelSourceSpec) []gisrepo.ParcelSourceRow {
	out := make([]gisrepo.ParcelSourceRow, 0, len(specs))
	for _, s := range specs {
		row := gisrepo.ParcelSourceRow{
			SourceKey:  s.SourceKey,
			CountySlug: s.CountySlug,
			QueryURL:   s.QueryURL,
			SyncMode:   s.SyncMode,
			MLSFeed:    s.MLSFeed,
			Enabled:    s.Enabled,
			Priority:   s.Priority,
		}
		if s.ArcGISWhere != "" {
			row.ArcGISWhere = &s.ArcGISWhere
		}
		row.BBoxWest = &s.SyncBBox.West
		row.BBoxSouth = &s.SyncBBox.South
		row.BBoxEast = &s.SyncBBox.East
		row.BBoxNorth = &s.SyncBBox.North
		if s.HTTPTimeoutSec > 0 {
			row.HTTPTimeoutSec = &s.HTTPTimeoutSec
		}
		if s.Notes != "" {
			row.Notes = &s.Notes
		}
		out = append(out, row)
	}
	return out
}

// UnmarshalParcelSyncPageArgs parses job args.
func UnmarshalParcelSyncPageArgs(raw json.RawMessage) (ParcelSyncPageArgs, error) {
	var args ParcelSyncPageArgs
	err := json.Unmarshal(raw, &args)
	return args, err
}
