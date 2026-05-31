package geocode

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
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

const (
	geocodeFailureInsufficientAddress = "insufficient_address"
	geocodeFailureZeroResults         = "zero_results"
	geocodeFailureRequestError        = "request_error"
)

// EnrichmentService runs Google Geocoding backfill for mirror listings.
type EnrichmentService struct {
	cfg          config.Config
	db           *repository.DB
	queue        *queue.Client
	repo         *geocoderepo.Repository
	client       *Geocoder
	logger       *slog.Logger
	femaKickoff  func(ctx context.Context, datasetSlug string) error
}

// SetFEMAKickoff wires optional FEMA re-enrichment after coordinate recovery.
func (s *EnrichmentService) SetFEMAKickoff(fn func(ctx context.Context, datasetSlug string) error) {
	s.femaKickoff = fn
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
	// Fail fast so the job lands in failed_jobs and operators are alerted
	// rather than silently burning attempt counts with no key configured.
	if !s.client.IsConfigured() {
		return errors.New("GOOGLE_MAPS_GEOCODING_API_KEY is not configured; geocode batch aborted")
	}

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
	var recoveryUpdated bool
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
			if markErr := s.repo.MarkFailedAttempt(ctx, row.ID, geocodeFailureInsufficientAddress, query, true); markErr != nil {
				s.logger.Warn("geocode failure mark failed", "listing_id", row.ID, "error", markErr)
			}
			continue
		}

		lat, lng, geocodeErr := s.client.Geocode(ctx, query)
		if geocodeErr != nil {
			var statusErr ErrGeocodeStatus
			if errors.As(geocodeErr, &statusErr) {
				switch statusErr.Status {
				case "ZERO_RESULTS", "INVALID_REQUEST":
					// Permanent address-level failure — mark bad so it is excluded
					// from future batches.
					s.logger.Debug("geocode bad address",
						"listing_id", row.ID, "status", statusErr.Status, "query", query)
					if markErr := s.repo.MarkFailedAttempt(ctx, row.ID, geocodeFailureZeroResults, query, true); markErr != nil {
						s.logger.Warn("geocode bad address mark failed", "listing_id", row.ID, "error", markErr)
					}
				case "OVER_QUERY_LIMIT", "OVER_DAILY_LIMIT":
					// Quota exhausted — abort the batch so the job is retried
					// after the worker's retry delay rather than burning attempt
					// counts on every remaining row.
					s.logger.Warn("google geocoding quota exceeded; stopping batch",
						"listing_id", row.ID, "status", statusErr.Status)
					return geocodeErr
				default:
					// REQUEST_DENIED, UNKNOWN_ERROR, transient upstream errors.
					s.logger.Warn("geocode request error",
						"listing_id", row.ID, "status", statusErr.Status, "query", query)
					if markErr := s.repo.MarkFailedAttempt(ctx, row.ID, geocodeFailureRequestError, query, false); markErr != nil {
						s.logger.Warn("geocode request error mark failed", "listing_id", row.ID, "error", markErr)
					}
				}
			} else {
				// Network or HTTP-level failure (not a Google status error).
				s.logger.Warn("geocode network error", "listing_id", row.ID, "error", geocodeErr)
				if markErr := s.repo.MarkFailedAttempt(ctx, row.ID, geocodeFailureRequestError, query, false); markErr != nil {
					s.logger.Warn("geocode network error mark failed", "listing_id", row.ID, "error", markErr)
				}
			}
			time.Sleep(interval)
			continue
		}

		if coordErr := s.repo.ApplyCoords(ctx, row.ID, lat, lng, query); coordErr != nil {
			s.logger.Warn("geocode persist failed", "listing_id", row.ID, "error", coordErr)
			// Record the attempt so the listing is not hot-looped back through
			// the pending queue with no state change.
			if markErr := s.repo.MarkFailedAttempt(ctx, row.ID, geocodeFailureRequestError, query, false); markErr != nil {
				s.logger.Warn("geocode persist mark failed", "listing_id", row.ID, "error", markErr)
			}
		} else {
			updated++
			if row.Recovery {
				recoveryUpdated = true
			}
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

	if recoveryUpdated && s.femaKickoff != nil {
		if err := s.femaKickoff(ctx, args.DatasetSlug); err != nil {
			s.logger.Warn("fema re-enrich kickoff failed", "error", err)
		}
	}

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

// CountPendingForAdmin returns listings awaiting geocode for dashboard/admin.
func (s *EnrichmentService) CountPendingForAdmin(ctx context.Context, datasetSlug string) (int64, error) {
	return s.repo.CountPending(ctx, datasetSlug)
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
