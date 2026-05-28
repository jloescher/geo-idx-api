package dashboard

import (
	"context"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
	"github.com/quantyralabs/idx-api/internal/service/crypto"
	"github.com/quantyralabs/idx-api/internal/service/mls"
	"github.com/quantyralabs/idx-api/internal/service/sync"
)

// MonitoringService composes ops metrics for the dashboard JSON API.
type MonitoringService struct {
	cfg    config.Config
	repo   *repository.MonitoringRepo
	gis    *gisrepo.Repository
	crypto *crypto.PricingReader
	fresh  *sync.Freshness
}

// NewMonitoringService wires monitoring dependencies.
func NewMonitoringService(cfg config.Config, db *repository.DB) *MonitoringService {
	return &MonitoringService{
		cfg:    cfg,
		repo:   repository.NewMonitoringRepo(db),
		gis:    gisrepo.New(db),
		crypto: crypto.NewPricingReader(db),
		fresh:  sync.NewFreshness(cfg, db),
	}
}

// Snapshot is the monitoring JSON payload.
type Snapshot struct {
	GeneratedAt    time.Time          `json:"generated_at"`
	Listings       []ListingMetric    `json:"listings"`
	GIS            GISMetric          `json:"gis"`
	Crypto         CryptoMetric       `json:"crypto"`
	Cache          CacheMetric        `json:"cache"`
	Queues         QueuesMetric       `json:"queues"`
	SyncPipeline   SyncPipelineMetric `json:"sync_pipeline"`
	Infrastructure InfraMetric        `json:"infrastructure"`
	Incidents      []IncidentMetric   `json:"incidents"`
	Activation     ActivationMetric   `json:"activation"`
}

type ListingMetric struct {
	DatasetSlug           string     `json:"dataset_slug"`
	Total                 int64      `json:"total"`
	ActivePending         int64      `json:"active_pending"`
	OldestListingAgeDays  *float64   `json:"oldest_listing_age_days,omitempty"`
	LatestModification    *time.Time `json:"latest_modification,omitempty"`
	LastSyncFinishedAt    *time.Time `json:"last_sync_finished_at,omitempty"`
	ReplicationInProgress bool       `json:"replication_in_progress"`
	LagSeconds            *int64     `json:"lag_seconds,omitempty"`
	FreshnessMode         string     `json:"freshness_mode"`
	Status                string     `json:"status"`
}

type GISMetric struct {
	ParcelsTotal         int64                    `json:"parcels_total"`
	ParcelsLastSyncedAt  *time.Time               `json:"parcels_last_synced_at,omitempty"`
	ParcelsStatus        string                   `json:"parcels_status"`
	ByCounty             map[string]int64         `json:"by_county"`
	CitiesTotal          int64                    `json:"cities_total"`
	CitiesLastSyncedAt   *time.Time               `json:"cities_last_synced_at,omitempty"`
	CitiesStatus         string                   `json:"cities_status"`
	CountiesTotal        int64                    `json:"counties_total"`
	CountiesLastSyncedAt *time.Time               `json:"counties_last_synced_at,omitempty"`
	CountiesStatus       string                   `json:"counties_status"`
	ZipsTotal            int64                    `json:"zips_total"`
	ZipsLastSyncedAt     *time.Time               `json:"zips_last_synced_at,omitempty"`
	ZipsStatus           string                   `json:"zips_status"`
	BoundaryStaleDays    int                      `json:"boundary_stale_days"`
	Sources              []gisrepo.SourceStateRow `json:"sources"`
	Status               string                   `json:"status"`
}

// BoundaryLayerStatus reports healthy/stale/unknown for infrequently refreshed boundary tables.
func BoundaryLayerStatus(lastSynced *time.Time, total int64, staleDays int, now time.Time) string {
	if total == 0 || lastSynced == nil {
		return "unknown"
	}
	if staleDays <= 0 {
		staleDays = 90
	}
	cutoff := now.AddDate(0, 0, -staleDays)
	if lastSynced.Before(cutoff) {
		return "stale"
	}
	return "healthy"
}

type CryptoAssetMetric struct {
	AssetKey   string    `json:"asset_key"`
	PriceUSD   float64   `json:"price_usd"`
	CapturedAt time.Time `json:"captured_at"`
	AgeSeconds int64     `json:"age_seconds"`
	Stale      bool      `json:"stale"`
}

type CryptoMetric struct {
	Assets []CryptoAssetMetric `json:"assets"`
	Status string              `json:"status"`
}

type CacheMetric struct {
	WindowMinutes int     `json:"window_minutes"`
	Total         int64   `json:"total"`
	Hits          int64   `json:"hits"`
	Misses        int64   `json:"misses"`
	HitRatePct    float64 `json:"hit_rate_pct"`
	Status        string  `json:"status"`
}

type QueuesMetric struct {
	ByQueue            []repository.QueueCount      `json:"by_queue"`
	TopTypes           []repository.JobTypeCount    `json:"top_job_types"`
	FailedTop          []repository.FailedJobDetail `json:"failed_top"`
	TotalPending       int64                        `json:"total_pending"`
	TotalStaleReserved int64                        `json:"total_stale_reserved"`
	TotalFailed        int64                        `json:"total_failed"`
	TotalFailedRecent  int64                        `json:"total_failed_recent"`
	Status             string                       `json:"status"`
}

type SyncPipelineMetric struct {
	ByStatus []repository.ReplicaPageStatusCount `json:"by_status"`
	Status   string                              `json:"status"`
}

type InfraMetric struct {
	Scheduler repository.SchedulerLockHealth `json:"scheduler"`
	Status    string                         `json:"status"`
}

type IncidentMetric struct {
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
}

type ActivationMetric struct {
	DomainCount           int64 `json:"domain_count"`
	VerifiedDomainCount   int64 `json:"verified_domain_count"`
	TokenCount            int64 `json:"token_count"`
	InvitationsAccepted   int64 `json:"invitations_accepted"`
	DomainsWithTraffic30d int64 `json:"domains_with_traffic_30d"`
}

// BuildSnapshot loads all monitoring sections.
// Revenue impact: live freshness signals reduce time-to-diagnose stale maps and protect conversion.
func (s *MonitoringService) BuildSnapshot(ctx context.Context) (*Snapshot, error) {
	now := time.Now()
	snap := &Snapshot{GeneratedAt: now}

	listRows, err := s.repo.ListListingStats(ctx)
	if err != nil {
		return nil, err
	}
	pipeline, err := s.repo.ReplicaPageStatuses(ctx)
	if err != nil {
		return nil, err
	}
	activeReplica := make(map[string]bool)
	for _, row := range pipeline {
		if row.Status == "pending" || row.Status == "processing" {
			activeReplica[row.DatasetSlug] = true
		}
	}

	resolver := mls.NewResolver(s.cfg)
	for _, row := range listRows {
		m := ListingMetric{
			DatasetSlug:           row.DatasetSlug,
			Total:                 row.Total,
			ActivePending:         row.ActivePending,
			LatestModification:    row.LatestModification,
			LastSyncFinishedAt:    row.LastSyncFinishedAt,
			ReplicationInProgress: row.ReplicationInProgress,
			FreshnessMode:         sync.ModeCatchUp,
			Status:                "unknown",
		}
		if row.OldestModification != nil {
			days := now.Sub(*row.OldestModification).Hours() / 24
			m.OldestListingAgeDays = &days
		}
		if row.LastSyncFinishedAt != nil {
			lag := int64(now.Sub(*row.LastSyncFinishedAt).Seconds())
			m.LagSeconds = &lag
		}
		provider := providerForDataset(resolver, row.DatasetSlug)
		isCurrent := false
		if provider != "" {
			current, err := s.fresh.IsCurrent(ctx, row.DatasetSlug, provider)
			if err == nil {
				isCurrent = current
				if current {
					m.FreshnessMode = sync.ModeSteady
				} else {
					m.FreshnessMode = sync.ModeCatchUp
				}
			}
		}
		hasActiveReplica := row.ReplicationInProgress || activeReplica[row.DatasetSlug]
		m.Status = ListingDatasetStatus(isCurrent, hasActiveReplica)
		snap.Listings = append(snap.Listings, m)
	}

	counts, err := s.gis.MonitoringCounts(ctx)
	if err != nil {
		return nil, err
	}
	byCounty, err := s.gis.ParcelsByCounty(ctx)
	if err != nil {
		return nil, err
	}
	sources, err := s.gis.ListSourceStates(ctx)
	if err != nil {
		return nil, err
	}
	parcelsSyncedAt, err := s.gis.MaxParcelSyncedAt(ctx)
	if err != nil {
		return nil, err
	}
	citiesSyncedAt, err := s.gis.MaxCitySyncedAt(ctx)
	if err != nil {
		return nil, err
	}
	countiesSyncedAt, err := s.gis.MaxCountySyncedAt(ctx)
	if err != nil {
		return nil, err
	}
	zipsSyncedAt, err := s.gis.MaxZipSyncedAt(ctx)
	if err != nil {
		return nil, err
	}

	staleDays := s.cfg.GIS.BoundaryStaleDays
	if staleDays <= 0 {
		staleDays = 90
	}

	citiesStatus := BoundaryLayerStatus(citiesSyncedAt, counts.CitiesTotal, staleDays, now)
	countiesStatus := BoundaryLayerStatus(countiesSyncedAt, counts.CountiesTotal, staleDays, now)
	zipsStatus := BoundaryLayerStatus(zipsSyncedAt, counts.ZipsTotal, staleDays, now)

	parcelsStatus := "unknown"
	if counts.ParcelsTotal > 0 {
		parcelsStatus = "healthy"
	}
	for _, src := range sources {
		if src.Status == "stale" {
			parcelsStatus = "stale"
			break
		}
	}

	gisStatus := "healthy"
	if parcelsStatus == "stale" || citiesStatus == "stale" || countiesStatus == "stale" || zipsStatus == "stale" {
		gisStatus = "stale"
	} else if parcelsStatus == "unknown" && citiesStatus == "unknown" && countiesStatus == "unknown" && zipsStatus == "unknown" {
		gisStatus = "unknown"
	}

	snap.GIS = GISMetric{
		ParcelsTotal:         counts.ParcelsTotal,
		ParcelsLastSyncedAt:  parcelsSyncedAt,
		ParcelsStatus:        parcelsStatus,
		ByCounty:             byCounty,
		CitiesTotal:          counts.CitiesTotal,
		CitiesLastSyncedAt:   citiesSyncedAt,
		CitiesStatus:         citiesStatus,
		CountiesTotal:        counts.CountiesTotal,
		CountiesLastSyncedAt: countiesSyncedAt,
		CountiesStatus:       countiesStatus,
		ZipsTotal:            counts.ZipsTotal,
		ZipsLastSyncedAt:     zipsSyncedAt,
		ZipsStatus:           zipsStatus,
		BoundaryStaleDays:    staleDays,
		Sources:              sources,
		Status:               gisStatus,
	}

	prices, err := s.crypto.LatestPrices(ctx)
	if err != nil {
		return nil, err
	}
	cryptoStatus := "unknown"
	if len(prices) > 0 {
		cryptoStatus = "healthy"
	}
	for _, p := range prices {
		age := int64(now.Sub(p.CapturedAt).Seconds())
		stale := age > 3600
		if stale {
			cryptoStatus = "stale"
		}
		snap.Crypto.Assets = append(snap.Crypto.Assets, CryptoAssetMetric{
			AssetKey:   p.AssetKey,
			PriceUSD:   p.Price,
			CapturedAt: p.CapturedAt,
			AgeSeconds: age,
			Stale:      stale,
		})
	}
	snap.Crypto.Status = cryptoStatus

	cache, err := s.repo.CacheHitRate15m(ctx)
	if err != nil {
		return nil, err
	}
	snap.Cache = CacheMetric{
		WindowMinutes: 15,
		Total:         cache.Total,
		Hits:          cache.Hits,
		Misses:        cache.Misses,
		HitRatePct:    cache.HitRatePct,
		Status:        CacheStatus(cache.Total, cache.HitRatePct),
	}

	staleReservedAfter := StaleReservedAfter(int(s.cfg.Queue.ReservationTimeout.Seconds()))
	queues, err := s.repo.ListQueueCounts(ctx, staleReservedAfter)
	if err != nil {
		return nil, err
	}
	queues = MergeKnownQueues(s.cfg, queues)
	topTypes, err := s.repo.TopPendingJobTypes(ctx, 10)
	if err != nil {
		return nil, err
	}
	if topTypes == nil {
		topTypes = make([]repository.JobTypeCount, 0)
	}
	failedTop, err := s.repo.TopFailedJobDetails(ctx, 10)
	if err != nil {
		return nil, err
	}
	if failedTop == nil {
		failedTop = make([]repository.FailedJobDetail, 0)
	}
	var pending int64
	var staleReserved int64
	var totalFailed int64
	var totalFailedRecent int64
	for _, q := range queues {
		pending += q.Pending
		staleReserved += q.StaleReserved
		totalFailed += q.Failed
		totalFailedRecent += q.FailedRecent
	}
	snap.Queues = QueuesMetric{
		ByQueue:            queues,
		TopTypes:           topTypes,
		FailedTop:          failedTop,
		TotalPending:       pending,
		TotalStaleReserved: staleReserved,
		TotalFailed:        totalFailed,
		TotalFailedRecent:  totalFailedRecent,
		Status:             QueueStatus(pending, staleReserved, totalFailedRecent),
	}

	snap.SyncPipeline = SyncPipelineMetric{
		ByStatus: pipeline,
		Status:   SyncPipelineStatus(pipeline),
	}

	lockID := s.cfg.Scheduler.LeaderLockKey
	if lockID == 0 {
		lockID = 913374211
	}
	scheduler, err := s.repo.SchedulerLockHealth(ctx, lockID)
	if err != nil {
		return nil, err
	}
	snap.Infrastructure = InfraMetric{
		Scheduler: scheduler,
		Status:    InfraStatus(scheduler.LeaderActive),
	}

	act, err := s.repo.Activation(ctx)
	if err != nil {
		return nil, err
	}
	snap.Activation = ActivationMetric{
		DomainCount:           act.DomainCount,
		VerifiedDomainCount:   act.VerifiedDomainCount,
		TokenCount:            act.TokenCount,
		InvitationsAccepted:   act.InvitationsAccepted,
		DomainsWithTraffic30d: act.DomainsWithTraffic30d,
	}

	snap.Incidents = BuildIncidents(IncidentInput{
		InfraStatus:            snap.Infrastructure.Status,
		SchedulerLeaderActive:  scheduler.LeaderActive,
		SchedulerHolderPID:     scheduler.HolderPID,
		SchedulerLastEnqueue:   scheduler.LastEnqueue,
		SchedulerBackendCount:  scheduler.SchedulerBackends,
		TotalStaleReserved:     snap.Queues.TotalStaleReserved,
		StaleReservedAfterSecs: staleReservedAfter,
		TotalFailed:            snap.Queues.TotalFailedRecent,
		SyncPipelineStatus:     snap.SyncPipeline.Status,
	})
	for _, src := range snap.GIS.Sources {
		if !src.Enabled {
			continue
		}
		if src.APIStatus == "unreachable" {
			detail := src.LastProbeError
			if detail == "" {
				detail = "ArcGIS metadata probe failed; use GIS Sources panel to re-probe or start sync."
			}
			snap.Incidents = append(snap.Incidents, IncidentMetric{
				Severity: "warning",
				Source:   "gis",
				Title:    "GIS source unreachable: " + src.SourceKey,
				Detail:   detail,
			})
		}
	}

	return snap, nil
}

func providerForDataset(resolver *mls.Resolver, datasetSlug string) string {
	for _, f := range resolver.Catalog() {
		if f.Dataset == datasetSlug {
			return f.Provider
		}
	}
	if datasetSlug == "stellar" {
		return "bridge"
	}
	if datasetSlug == "beaches" {
		return "spark"
	}
	return ""
}
