package fema

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
	femarepo "github.com/quantyralabs/idx-api/internal/repository/fema"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

// FloodEnrichKickoffArgs is the kickoff job payload (empty or optional dataset filter).
type FloodEnrichKickoffArgs struct {
	DatasetSlug string `json:"dataset_slug,omitempty"`
}

// FloodEnrichBatchArgs is one batch of stale listings.
type FloodEnrichBatchArgs struct {
	CursorID    int64  `json:"cursor_id"`
	Limit       int    `json:"limit"`
	DatasetSlug string `json:"dataset_slug,omitempty"`
}

// EnrichmentService runs NFHL point enrichment jobs.
type EnrichmentService struct {
	cfg    config.Config
	db     *repository.DB
	queue  *queue.Client
	repo   *femarepo.Repository
	client *Client
	logger *slog.Logger
}

// NewEnrichmentService wires FEMA enrichment dependencies.
func NewEnrichmentService(cfg config.Config, db *repository.DB, q *queue.Client, logger *slog.Logger) *EnrichmentService {
	if logger == nil {
		logger = slog.Default()
	}
	return &EnrichmentService{
		cfg:    cfg,
		db:     db,
		queue:  q,
		repo:   femarepo.NewRepository(db.Pool, cfg.FEMA.FloodStaleDays),
		client: NewClient(cfg.FEMA),
		logger: logger,
	}
}

// EnqueueKickoffIfAbsent enqueues fema.flood_enrich_kickoff unless one is already pending.
func (s *EnrichmentService) EnqueueKickoffIfAbsent(ctx context.Context, datasetSlug string) error {
	active, err := femarepo.HasActiveFloodEnrichJob(ctx, s.db.Pool, s.cfg.Queue.Table)
	if err != nil {
		return err
	}
	if active {
		s.logger.Debug("fema flood enrich kickoff skipped, job already queued")
		return nil
	}
	_, err = s.queue.Enqueue(ctx, s.cfg.FEMA.EnrichQueue, queue.TypeFEMAFloodEnrichKickoff, FloodEnrichKickoffArgs{
		DatasetSlug: datasetSlug,
	}, 0)
	if err != nil {
		return err
	}
	s.logger.Info("fema flood enrich kickoff enqueued", "queue", s.cfg.FEMA.EnrichQueue)
	return nil
}

// Kickoff counts stale rows and enqueues the first batch.
func (s *EnrichmentService) Kickoff(ctx context.Context, args FloodEnrichKickoffArgs) error {
	stale, err := s.repo.CountStale(ctx, args.DatasetSlug)
	if err != nil {
		return err
	}
	s.logger.Info("fema flood enrich kickoff", "stale", stale, "dataset", args.DatasetSlug)
	if stale == 0 {
		return nil
	}
	limit := s.cfg.FEMA.EnrichBatchSize
	if limit <= 0 {
		limit = 2000
	}
	_, err = s.queue.Enqueue(ctx, s.cfg.FEMA.EnrichQueue, queue.TypeFEMAFloodEnrichBatch, FloodEnrichBatchArgs{
		CursorID:    0,
		Limit:       limit,
		DatasetSlug: args.DatasetSlug,
	}, 0)
	return err
}

// RunBatch processes one batch and chains the next when full.
func (s *EnrichmentService) RunBatch(ctx context.Context, args FloodEnrichBatchArgs) error {
	start := time.Now()
	defer func() {
		enrichBatchDurationSeconds.Observe(time.Since(start).Seconds())
	}()

	limit := args.Limit
	if limit <= 0 {
		limit = s.cfg.FEMA.EnrichBatchSize
	}
	if limit <= 0 {
		limit = 2000
	}

	rows, err := s.repo.SelectStaleForEnrichment(ctx, args.CursorID, limit, args.DatasetSlug)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		s.logger.Info("fema flood enrich batch complete", "cursor", args.CursorID, "processed", 0)
		return nil
	}

	now := time.Now().UTC()
	updates := make([]femarepo.FEMAUpdate, 0, len(rows))
	var mu sync.Mutex
	var wg sync.WaitGroup
	workers := 8
	sem := make(chan struct{}, workers)

	for _, row := range rows {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(r femarepo.ListingCoord) {
			defer wg.Done()
			defer func() { <-sem }()

			attrs, err := s.client.QueryPoint(ctx, r.Latitude, r.Longitude)
			if err != nil {
				s.logger.Warn("fema point query failed", "listing_id", r.ID, "error", err)
				return
			}

			u := buildListingFEMAUpdate(r.ID, now, attrs)

			mu.Lock()
			updates = append(updates, u)
			mu.Unlock()
		}(row)
	}
	wg.Wait()

	if err := s.repo.BatchUpdateFEMA(ctx, updates); err != nil {
		return err
	}
	enrichListingsUpdatedTotal.Add(float64(len(updates)))

	maxID := rows[len(rows)-1].ID
	s.logger.Info("fema flood enrich batch",
		"cursor", args.CursorID,
		"processed", len(rows),
		"updated", len(updates),
		"max_id", maxID,
	)

	if len(rows) == limit {
		_, err = s.queue.Enqueue(ctx, s.cfg.FEMA.EnrichQueue, queue.TypeFEMAFloodEnrichBatch, FloodEnrichBatchArgs{
			CursorID:    maxID,
			Limit:       limit,
			DatasetSlug: args.DatasetSlug,
		}, 0)
		return err
	}
	return nil
}

// buildListingFEMAUpdate maps a successful NFHL point query to a persistable row update.
func buildListingFEMAUpdate(id int64, now time.Time, attrs *PointAttributes) femarepo.FEMAUpdate {
	u := femarepo.FEMAUpdate{
		ID:                 id,
		FloodZoneUpdatedAt: now,
		LowRiskFloodZoneYN: false,
	}
	if attrs != nil {
		u.FEMAFloodZoneCode = attrs.FLDZone
		u.FloodZoneSFHA_TF = attrs.SFHA_TF
		if len(attrs.Raw) > 0 {
			u.FloodZoneRaw = attrs.Raw
		}
		u.LowRiskFloodZoneYN = mls.ComputeLowRiskFloodZoneYN(attrs.FLDZone)
	}
	return u
}

// CountStaleForAdmin returns listings needing FEMA refresh.
func (s *EnrichmentService) CountStaleForAdmin(ctx context.Context, datasetSlug string) (int64, error) {
	return s.repo.CountStale(ctx, datasetSlug)
}

// AdminKickoff enqueues kickoff and returns stale estimate.
func (s *EnrichmentService) AdminKickoff(ctx context.Context, datasetSlug string) (stale int64, enqueued bool, err error) {
	stale, err = s.repo.CountStale(ctx, datasetSlug)
	if err != nil {
		return 0, false, err
	}
	if err := s.EnqueueKickoffIfAbsent(ctx, datasetSlug); err != nil {
		return stale, false, err
	}
	return stale, true, nil
}

// UnmarshalBatchArgs parses batch job args.
func UnmarshalBatchArgs(raw json.RawMessage) (FloodEnrichBatchArgs, error) {
	var args FloodEnrichBatchArgs
	if len(raw) == 0 {
		return args, nil
	}
	err := json.Unmarshal(raw, &args)
	return args, err
}

// UnmarshalKickoffArgs parses kickoff job args.
func UnmarshalKickoffArgs(raw json.RawMessage) (FloodEnrichKickoffArgs, error) {
	var args FloodEnrichKickoffArgs
	if len(raw) == 0 {
		return args, nil
	}
	err := json.Unmarshal(raw, &args)
	return args, err
}
