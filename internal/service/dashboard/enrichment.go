package dashboard

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quantyralabs/idx-api/internal/config"
	femarepo "github.com/quantyralabs/idx-api/internal/repository/fema"
	geocoderepo "github.com/quantyralabs/idx-api/internal/repository/geocode"
)

// OutcomeCount is a reason → count pair for enrichment dashboards.
type OutcomeCount struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

// DatasetEnrichmentBreakdown groups outcome counts per dataset.
type DatasetEnrichmentBreakdown struct {
	DatasetSlug string         `json:"dataset_slug"`
	ByReason    []OutcomeCount `json:"by_reason"`
}

// FEMANullSample is a drill-down row for listings missing FEMA codes.
type FEMANullSample struct {
	ListingKey         string  `json:"listing_key"`
	DatasetSlug        string  `json:"dataset_slug"`
	FailureReason      string  `json:"failure_reason"`
	AttemptCount       int     `json:"attempt_count"`
	FloodZoneUpdatedAt *string `json:"flood_zone_updated_at,omitempty"`
}

// GeocodeFailureSample is a drill-down row for geocode failures.
type GeocodeFailureSample struct {
	ListingKey    string  `json:"listing_key"`
	DatasetSlug   string  `json:"dataset_slug"`
	FailureReason string  `json:"failure_reason"`
	AttemptCount  int     `json:"attempt_count"`
	QuerySnippet  string  `json:"query_snippet,omitempty"`
	FailedAt      *string `json:"failed_at,omitempty"`
}

// FEMAMetric summarizes NFHL enrichment health.
type FEMAMetric struct {
	Stale          int64                        `json:"stale"`
	NullWithCoords int64                        `json:"null_with_coords"`
	ByReason       []OutcomeCount               `json:"by_reason"`
	ByDataset      []DatasetEnrichmentBreakdown `json:"by_dataset"`
	ActiveJob      bool                         `json:"active_job"`
	NullSamples    []FEMANullSample               `json:"null_samples"`
	Status         string                       `json:"status"`
}

// GeocodeMetric summarizes coordinate backfill health.
type GeocodeMetric struct {
	Pending           int64                        `json:"pending"`
	MissingCoords     int64                        `json:"missing_coords"`
	BadAddress        int64                        `json:"bad_address"`
	HighAttemptErrors int64                        `json:"high_attempt_errors"`
	ByReason          []OutcomeCount               `json:"by_reason"`
	ByDataset         []DatasetEnrichmentBreakdown `json:"by_dataset"`
	ActiveJob         bool                         `json:"active_job"`
	FailureSamples    []GeocodeFailureSample       `json:"failure_samples"`
	Status            string                       `json:"status"`
}

// EnrichmentMetric groups FEMA and geocode listing enrichment signals.
type EnrichmentMetric struct {
	FEMA    FEMAMetric    `json:"fema"`
	Geocode GeocodeMetric `json:"geocode"`
	Status  string        `json:"status"`
}

func loadEnrichmentMetrics(ctx context.Context, cfg config.Config, pool *pgxpool.Pool) (*EnrichmentMetric, error) {
	femaRepo := femarepo.NewRepository(pool, cfg.FEMA.FloodStaleDays)
	geoRepo := geocoderepo.NewRepository(pool)

	stale, err := femaRepo.CountStale(ctx, "")
	if err != nil {
		return nil, err
	}
	nullCoords, err := femaRepo.CountNullFEMAWithCoords(ctx, "")
	if err != nil {
		return nil, err
	}
	femaByReason, err := femaRepo.CountByOutcome(ctx, "")
	if err != nil {
		return nil, err
	}
	femaByDatasetRows, err := femaRepo.CountByDatasetOutcome(ctx)
	if err != nil {
		return nil, err
	}
	femaSamples, err := femaRepo.ListNullWithCoordsSamples(ctx, 25)
	if err != nil {
		return nil, err
	}
	femaActive, err := femarepo.HasActiveFloodEnrichJob(ctx, pool, cfg.Queue.Table)
	if err != nil {
		return nil, err
	}

	geoPending, err := geoRepo.CountPending(ctx, "")
	if err != nil {
		return nil, err
	}
	missingCoords, err := geoRepo.CountMissingCoords(ctx, "")
	if err != nil {
		return nil, err
	}
	badAddress, err := geoRepo.CountBadAddress(ctx, "")
	if err != nil {
		return nil, err
	}
	highAttempt, err := geoRepo.CountHighAttemptRequestErrors(ctx, 5)
	if err != nil {
		return nil, err
	}
	geoByReason, err := geoRepo.CountByFailureReason(ctx, "")
	if err != nil {
		return nil, err
	}
	geoByDatasetRows, err := geoRepo.CountByDatasetFailureReason(ctx)
	if err != nil {
		return nil, err
	}
	geoSamples, err := geoRepo.ListFailureSamples(ctx, 25)
	if err != nil {
		return nil, err
	}
	geoActive, err := geocoderepo.HasActiveGeocodeJob(ctx, pool, cfg.Queue.Table)
	if err != nil {
		return nil, err
	}

	out := &EnrichmentMetric{
		FEMA: FEMAMetric{
			Stale:          stale,
			NullWithCoords: nullCoords,
			ByReason:       mapFEMAOutcomeCounts(femaByReason),
			ByDataset:      groupFEMADatasetOutcomes(femaByDatasetRows),
			ActiveJob:      femaActive,
			NullSamples:    mapFEMANullSamples(femaSamples),
			Status:         femaStatus(stale),
		},
		Geocode: GeocodeMetric{
			Pending:           geoPending,
			MissingCoords:     missingCoords,
			BadAddress:        badAddress,
			HighAttemptErrors: highAttempt,
			ByReason:          mapGeocodeOutcomeCounts(geoByReason),
			ByDataset:         groupGeocodeDatasetOutcomes(geoByDatasetRows),
			ActiveJob:         geoActive,
			FailureSamples:    mapGeocodeFailureSamples(geoSamples),
			Status:            geocodeStatus(geoPending, highAttempt),
		},
	}
	out.Status = enrichmentStatus(out.FEMA.Status, out.Geocode.Status)
	return out, nil
}

func mapFEMAOutcomeCounts(rows []femarepo.OutcomeCount) []OutcomeCount {
	out := make([]OutcomeCount, 0, len(rows))
	for _, r := range rows {
		out = append(out, OutcomeCount{Reason: r.Reason, Count: r.Count})
	}
	return out
}

func mapGeocodeOutcomeCounts(rows []geocoderepo.OutcomeCount) []OutcomeCount {
	out := make([]OutcomeCount, 0, len(rows))
	for _, r := range rows {
		out = append(out, OutcomeCount{Reason: r.Reason, Count: r.Count})
	}
	return out
}

func groupFEMADatasetOutcomes(rows []femarepo.DatasetOutcomeCount) []DatasetEnrichmentBreakdown {
	byDS := map[string][]OutcomeCount{}
	order := []string{}
	for _, r := range rows {
		if _, ok := byDS[r.DatasetSlug]; !ok {
			order = append(order, r.DatasetSlug)
		}
		byDS[r.DatasetSlug] = append(byDS[r.DatasetSlug], OutcomeCount{Reason: r.Reason, Count: r.Count})
	}
	out := make([]DatasetEnrichmentBreakdown, 0, len(order))
	for _, ds := range order {
		out = append(out, DatasetEnrichmentBreakdown{DatasetSlug: ds, ByReason: byDS[ds]})
	}
	return out
}

func groupGeocodeDatasetOutcomes(rows []geocoderepo.DatasetOutcomeCount) []DatasetEnrichmentBreakdown {
	byDS := map[string][]OutcomeCount{}
	order := []string{}
	for _, r := range rows {
		if _, ok := byDS[r.DatasetSlug]; !ok {
			order = append(order, r.DatasetSlug)
		}
		byDS[r.DatasetSlug] = append(byDS[r.DatasetSlug], OutcomeCount{Reason: r.Reason, Count: r.Count})
	}
	out := make([]DatasetEnrichmentBreakdown, 0, len(order))
	for _, ds := range order {
		out = append(out, DatasetEnrichmentBreakdown{DatasetSlug: ds, ByReason: byDS[ds]})
	}
	return out
}

func mapFEMANullSamples(rows []femarepo.NullWithCoordsSample) []FEMANullSample {
	out := make([]FEMANullSample, 0, len(rows))
	for _, r := range rows {
		reason := femarepo.EffectiveFEMAFailureReason(r.FEMAFailureReason, r.FloodZoneUpdatedAt)
		attempts := r.FEMAAttemptCount
		if attempts == 0 && r.FloodZoneUpdatedAt != nil {
			attempts = 1
		}
		var updated *string
		if r.FloodZoneUpdatedAt != nil {
			s := r.FloodZoneUpdatedAt.UTC().Format(time.RFC3339)
			updated = &s
		}
		out = append(out, FEMANullSample{
			ListingKey:         r.ListingKey,
			DatasetSlug:        r.DatasetSlug,
			FailureReason:      reason,
			AttemptCount:       attempts,
			FloodZoneUpdatedAt: updated,
		})
	}
	return out
}

func mapGeocodeFailureSamples(rows []geocoderepo.FailureSample) []GeocodeFailureSample {
	out := make([]GeocodeFailureSample, 0, len(rows))
	for _, r := range rows {
		reason := "unknown"
		if r.GeocodeFailureReason != nil {
			reason = *r.GeocodeFailureReason
		}
		query := ""
		if r.GeocodeQuery != nil {
			query = truncateQuery(*r.GeocodeQuery, 80)
		}
		var failed *string
		if r.GeocodeFailedAt != nil {
			s := r.GeocodeFailedAt.UTC().Format(time.RFC3339)
			failed = &s
		}
		out = append(out, GeocodeFailureSample{
			ListingKey:    r.ListingKey,
			DatasetSlug:   r.DatasetSlug,
			FailureReason: reason,
			AttemptCount:  r.GeocodeAttemptCount,
			QuerySnippet:  query,
			FailedAt:      failed,
		})
	}
	return out
}

func truncateQuery(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func femaStatus(stale int64) string {
	if stale > 10000 {
		return "stale"
	}
	return "healthy"
}

func geocodeStatus(pending, highAttempt int64) string {
	if pending > 5000 || highAttempt > 100 {
		return "stale"
	}
	return "healthy"
}

func enrichmentStatus(fema, geocode string) string {
	if fema == "stale" || geocode == "stale" {
		return "stale"
	}
	return "healthy"
}
