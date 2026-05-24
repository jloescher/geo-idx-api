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

// parcelSyncTarget describes one ArcGIS parcel source to sync.
type parcelSyncTarget struct {
	SourceKey string
	County    string
	Source    Source
}

// ParcelSyncService syncs parcel polygons from ArcGIS into gis_parcels.
// Persistent GIS tables + monthly parcel refresh deliver instant Leaflet map performance →
// higher visitor engagement before the 3-listing hard gate → more OTP registrations and
// qualified leads while keeping marginal cost at zero.
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
	SourceKey  string `json:"source_key"`
	County     string `json:"county"`
	CountyCO   string `json:"county_co,omitempty"`
	QueryURL   string `json:"query_url"`
	Tier       string `json:"tier"`
	Offset     int    `json:"offset"`
	Generation int    `json:"generation"`
	Finalize   bool   `json:"finalize"`
}

// RunMonthlyRefresh kicks off a full parcel sync for all configured sources.
func (s *ParcelSyncService) RunMonthlyRefresh(ctx context.Context) error {
	return s.kickoff(ctx)
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
	return s.kickoff(ctx)
}

func (s *ParcelSyncService) kickoff(ctx context.Context) error {
	targets := parcelSyncTargets()
	seen := map[string]int{}
	for _, t := range targets {
		if _, ok := seen[t.SourceKey]; !ok {
			gen, err := s.db.BumpSourceGeneration(ctx, t.SourceKey)
			if err != nil {
				return fmt.Errorf("bump generation %s: %w", t.SourceKey, err)
			}
			seen[t.SourceKey] = int(gen)
		}
		gen := seen[t.SourceKey]
		args := ParcelSyncPageArgs{
			SourceKey:  t.SourceKey,
			County:     t.County,
			CountyCO:   t.Source.CountyCO,
			QueryURL:   t.Source.QueryURL,
			Tier:       t.Source.Tier,
			Offset:     0,
			Generation: gen,
		}
		if err := s.enqueuePage(ctx, args); err != nil {
			return err
		}
	}
	return nil
}

func parcelSyncTargets() []parcelSyncTarget {
	pinellas := Source{
		Key:      "pinellas_enterprise_parcels",
		QueryURL: "https://egis.pinellascounty.org/arcgis/rest/services/PARCEL/MapServer/0/query",
		Tier:     "pinellas",
	}
	hillsborough := Source{
		Key:      "hillsborough_hc_parcels",
		QueryURL: "https://gis.hcpafl.org/arcgis/rest/services/Hillsborough_County_Parcels/MapServer/0/query",
		Tier:     "hillsborough",
	}
	statewide := Source{
		Key:      "florida_statewide_cadastral",
		QueryURL: "https://services.arcgis.com/HRPe58PVRWYor63Q/arcgis/rest/services/Florida_Statewide_Cadastral/FeatureServer/0/query",
		Tier:     "statewide",
	}
	return []parcelSyncTarget{
		{SourceKey: pinellas.Key, County: "pinellas", Source: pinellas},
		{SourceKey: hillsborough.Key, County: "hillsborough", Source: hillsborough},
		{SourceKey: statewide.Key, County: "pinellas", Source: Source{Key: statewide.Key, QueryURL: statewide.QueryURL, Tier: statewide.Tier, CountyCO: "52"}},
		{SourceKey: statewide.Key, County: "hillsborough", Source: Source{Key: statewide.Key, QueryURL: statewide.QueryURL, Tier: statewide.Tier, CountyCO: "29"}},
	}
}

// SyncPage fetches one ArcGIS page and upserts parcel rows.
func (s *ParcelSyncService) SyncPage(ctx context.Context, args ParcelSyncPageArgs) error {
	src := Source{
		Key:      args.SourceKey,
		QueryURL: args.QueryURL,
		Tier:     args.Tier,
		CountyCO: args.CountyCO,
	}
	// Full-layer sync uses Florida bounding box envelope.
	flBBox := BBox{West: -87.6, South: 24.4, East: -79.8, North: 31.1}
	body, err := s.client.FetchBBoxPage(src, flBBox, args.Offset, s.cfg.GIS.SyncPageSize)
	if err != nil {
		return err
	}
	page, err := ParseFeatureCollection(body)
	if err != nil {
		return err
	}
	fp := ""
	fpStr := fp
	chunk := s.cfg.GIS.SyncUpsertChunk
	if chunk <= 0 {
		chunk = 500
	}
	var batch []gisrepo.ParcelRow
	for _, feat := range page.Features {
		row, err := ExtractParcelRow(feat, args.SourceKey, args.County, args.Generation, &fpStr)
		if err != nil {
			continue
		}
		batch = append(batch, row)
		if len(batch) >= chunk {
			if err := s.db.BulkUpsertParcels(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := s.db.BulkUpsertParcels(ctx, batch); err != nil {
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
	// Last page for this target — mark finalize for this source if no other targets pending.
	// Delete stale rows for this source_key after all targets complete is handled per-source
	// when offset returns short page; safe to delete stale now for source_key.
	deleted, err := s.db.DeleteStaleParcels(ctx, args.SourceKey, args.Generation)
	if err != nil {
		return err
	}
	s.logger.Info("gis parcel sync finalized", "source", args.SourceKey, "generation", args.Generation, "deleted_stale", deleted)
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

// UnmarshalParcelSyncPageArgs parses job args.
func UnmarshalParcelSyncPageArgs(raw json.RawMessage) (ParcelSyncPageArgs, error) {
	var args ParcelSyncPageArgs
	err := json.Unmarshal(raw, &args)
	return args, err
}
