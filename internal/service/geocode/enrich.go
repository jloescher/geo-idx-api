package geocode

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	geocoderepo "github.com/quantyralabs/idx-api/internal/repository/geocode"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// GeocodeKickoffArgs is the kickoff job payload.
type GeocodeKickoffArgs struct {
	DatasetSlug string `json:"dataset_slug,omitempty"`
}

// GeocodeBatchArgs is one batch of listings to geocode.
type GeocodeBatchArgs struct {
	CursorID    int64  `json:"cursor_id"`
	Limit       int    `json:"limit"`
	DatasetSlug string `json:"dataset_slug,omitempty"`
}

// EnrichmentService runs Google Geocoding backfill for mirror listings.
type EnrichmentService struct {
	cfg    config.Config
	db     *repository.DB
	queue  *queue.Client
	repo   *geocoderepo.Repository
	client *Geocoder
	logger *slog.Logger
}

// NewEnrichmentService wires geocode enrichment dependencies.
func NewEnrichmentService(cfg config.Config, db *repository.DB, q *queue.Client, logger *slog.Logger) *EnrichmentService {
	if logger == nil {
		logger = slog.Default()
	}
	return &EnrichmentService{
		cfg:    cfg,
		db:     db,
		queue:  q,
		repo:   geocoderepo.NewRepository(db.Pool),
		client: NewGeocoder(cfg.Geocode.GoogleMapsAPIKey, cfg.Geocode.HTTPTimeout),
		logger: logger,
	}
}

// EnqueueKickoffIfAbsent enqueues mls.geocode_listings_kickoff unless one is already pending.
func (s *EnrichmentService) EnqueueKickoffIfAbsent(ctx context.Context, datasetSlug string) error {
	active, err := geocoderepo.HasActiveGeocodeJob(ctx, s.db.Pool, s.cfg.Queue.Table)
	if err != nil {
		return err
	}
	if active {
		s.logger.Debug("geocode enrich kickoff skipped, job already queued")
		return nil
	}
	_, err = s.queue.Enqueue(ctx, s.cfg.Geocode.EnrichQueue, queue.TypeMLSGeocodeListingsKickoff, GeocodeKickoffArgs{
		DatasetSlug: datasetSlug,
	}, 0)
	if err != nil {
		return err
	}
	s.logger.Info("geocode enrich kickoff enqueued", "queue", s.cfg.Geocode.EnrichQueue)
	return nil
}

// Kickoff counts pending rows and enqueues the first batch.
func (s *EnrichmentService) Kickoff(ctx context.Context, args GeocodeKickoffArgs) error {
	pending, err := s.repo.CountPending(ctx, args.DatasetSlug)
	if err != nil {
		return err
	}
	s.logger.Info("geocode enrich kickoff", "pending", pending, "dataset", args.DatasetSlug)
	if pending == 0 {
		return nil
	}
	limit := s.cfg.Geocode.EnrichBatchSize
	if limit <= 0 {
		limit = 200
	}
	_, err = s.queue.Enqueue(ctx, s.cfg.Geocode.EnrichQueue, queue.TypeMLSGeocodeListingsBatch, GeocodeBatchArgs{
		CursorID:    0,
		Limit:       limit,
		DatasetSlug: args.DatasetSlug,
	}, 0)
	return err
}

// RunBatch geocodes one batch and chains the next when full.
func (s *EnrichmentService) RunBatch(ctx context.Context, args GeocodeBatchArgs) error {
	limit := args.Limit
	if limit <= 0 {
		limit = s.cfg.Geocode.EnrichBatchSize
	}
	if limit <= 0 {
		limit = 200
	}

	rows, err := s.repo.SelectPending(ctx, args.CursorID, limit, args.DatasetSlug)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		s.logger.Info("geocode enrich batch complete", "cursor", args.CursorID, "processed", 0)
		return nil
	}

	rps := s.cfg.Geocode.MaxRequestsPerSecond
	if rps <= 0 {
		rps = 5
	}
	interval := time.Second / time.Duration(rps)

	var updated int
	for _, row := range rows {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		query, ok := BuildGeocodeQuery(
			row.DatasetSlug,
			strVal(row.Unparsed),
			row.City, row.State, row.Postal,
			row.StreetNumber, row.StreetName,
		)
		if !ok {
			s.logger.Debug("geocode skip insufficient_address",
				"listing_id", row.ID, "dataset", row.DatasetSlug)
			continue
		}

		lat, lng, err := s.client.Geocode(ctx, query)
		if err != nil {
			if strings.Contains(err.Error(), "ZERO_RESULTS") {
				s.logger.Debug("geocode zero results", "listing_id", row.ID, "query", query)
			} else {
				s.logger.Warn("geocode failed", "listing_id", row.ID, "error", err)
			}
			time.Sleep(interval)
			continue
		}

		if err := s.repo.ApplyCoords(ctx, row.ID, lat, lng, query); err != nil {
			s.logger.Warn("geocode persist failed", "listing_id", row.ID, "error", err)
		} else {
			updated++
		}
		time.Sleep(interval)
	}

	maxID := rows[len(rows)-1].ID
	s.logger.Info("geocode enrich batch",
		"cursor", args.CursorID,
		"processed", len(rows),
		"updated", updated,
		"max_id", maxID,
	)

	if len(rows) == limit {
		_, err = s.queue.Enqueue(ctx, s.cfg.Geocode.EnrichQueue, queue.TypeMLSGeocodeListingsBatch, GeocodeBatchArgs{
			CursorID:    maxID,
			Limit:       limit,
			DatasetSlug: args.DatasetSlug,
		}, 0)
		return err
	}
	return nil
}

func strVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// UnmarshalGeocodeKickoffArgs parses kickoff job args.
func UnmarshalGeocodeKickoffArgs(raw json.RawMessage) (GeocodeKickoffArgs, error) {
	var args GeocodeKickoffArgs
	if len(raw) == 0 {
		return args, nil
	}
	err := json.Unmarshal(raw, &args)
	return args, err
}

// UnmarshalGeocodeBatchArgs parses batch job args.
func UnmarshalGeocodeBatchArgs(raw json.RawMessage) (GeocodeBatchArgs, error) {
	var args GeocodeBatchArgs
	if len(raw) == 0 {
		return args, nil
	}
	err := json.Unmarshal(raw, &args)
	return args, err
}
