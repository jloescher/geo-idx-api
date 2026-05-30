package job

import (
	"context"
	"fmt"

	"github.com/quantyralabs/idx-api/internal/debuglog"
	"github.com/quantyralabs/idx-api/internal/queue"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
	"github.com/quantyralabs/idx-api/internal/service/fema"
	"github.com/quantyralabs/idx-api/internal/service/geocode"
	"github.com/quantyralabs/idx-api/internal/service/gis"
)

func (r *Registry) handleReplicationKickoff(ctx context.Context, job *queue.ReservedJob) error {
	r.logger.Info("running replication kickoff", "job_id", job.ID)
	return r.replicationKickoff.Run(ctx)
}

func (r *Registry) handleReplicationResume(ctx context.Context, job *queue.ReservedJob) error {
	r.logger.Info("running replication resume", "job_id", job.ID)
	return r.replicationKickoff.ResumeStalledReplication(ctx)
}

func (r *Registry) handleProxyCachePurge(ctx context.Context, job *queue.ReservedJob) error {
	return r.proxyCachePurge.Run(ctx, job)
}

func (r *Registry) handleBridgeFetchPage(ctx context.Context, job *queue.ReservedJob) error {
	return r.bridgeSync.FetchPage(ctx, job)
}

func (r *Registry) handleBridgePersistChunk(ctx context.Context, job *queue.ReservedJob) error {
	return r.bridgeSync.PersistChunk(ctx, job)
}

func (r *Registry) handleBridgePersistFinalize(ctx context.Context, job *queue.ReservedJob) error {
	return r.bridgeSync.PersistFinalize(ctx, job)
}

func (r *Registry) handleSparkFetchPage(ctx context.Context, job *queue.ReservedJob) error {
	return r.sparkSync.FetchPage(ctx, job)
}

func (r *Registry) handleSparkPersistChunk(ctx context.Context, job *queue.ReservedJob) error {
	return r.sparkSync.PersistChunk(ctx, job)
}

func (r *Registry) handleSparkPersistFinalize(ctx context.Context, job *queue.ReservedJob) error {
	return r.sparkSync.PersistFinalize(ctx, job)
}

func (r *Registry) handlePurgeClosed(ctx context.Context, job *queue.ReservedJob) error {
	return r.mirrorPurge.Run(ctx)
}

func (r *Registry) handleMirrorKeyReconcile(ctx context.Context, job *queue.ReservedJob) error {
	r.logger.Info("running mirror key reconcile kickoff", "job_id", job.ID)
	return r.keyReconcile.Kickoff(ctx)
}

func (r *Registry) handleBridgeReconcileKeys(ctx context.Context, job *queue.ReservedJob) error {
	return r.keyReconcile.RunBridgePage(ctx, job)
}

func (r *Registry) handleSparkReconcileKeys(ctx context.Context, job *queue.ReservedJob) error {
	return r.keyReconcile.RunSparkPage(ctx, job)
}

func (r *Registry) handlePurgeReplicaPages(ctx context.Context, job *queue.ReservedJob) error {
	return r.replicaStore.PurgeEligible(ctx)
}

func (r *Registry) handleGISProbe(ctx context.Context, job *queue.ReservedJob) error {
	result := r.gisMeta.ProbeAll(ctx)
	if len(result.Failed) > 0 {
		return fmt.Errorf("gis probe failed for %d source(s)", len(result.Failed))
	}
	return nil
}

func (r *Registry) handleGISMonthlyParcelRefresh(ctx context.Context, job *queue.ReservedJob) error {
	return r.gisPersistent.RunMonthlyParcelRefresh(ctx)
}

func (r *Registry) handleGISAnnualBoundariesRefresh(ctx context.Context, job *queue.ReservedJob) error {
	return r.gisPersistent.RunAnnualBoundariesRefresh(ctx)
}

func (r *Registry) handleGISInitialSync(ctx context.Context, job *queue.ReservedJob) error {
	r.logger.Info("gis initial sync starting", "job_id", job.ID)
	return r.gisPersistent.RunInitialSync(ctx)
}

func (r *Registry) handleGISZipSync(ctx context.Context, job *queue.ReservedJob) error {
	return r.gisPersistent.RunZipSync(ctx)
}

func (r *Registry) handleGISParcelSyncPage(ctx context.Context, job *queue.ReservedJob) error {
	args, err := gis.UnmarshalParcelSyncPageArgs(job.Payload.Args)
	if err != nil {
		return err
	}
	return r.gisPersistent.SyncParcelPage(ctx, args)
}

func (r *Registry) handleGISShapefileImport(ctx context.Context, job *queue.ReservedJob) error {
	args, err := gis.UnmarshalShapefileImportArgs(job.Payload.Args)
	if err != nil {
		return err
	}
	r.logger.Info("gis shapefile import job reserved",
		"job_id", job.ID,
		"queue", job.Queue,
		"attempt", job.Attempts(),
		"source_key", args.SourceKey,
		"upload_id", args.UploadID,
		"path", args.StoragePath,
	)
	// #region agent log
	debuglog.Agent("SHP-C", "handlers.go:handleGISShapefileImport", "job started", map[string]any{
		"job_id": job.ID, "source_key": args.SourceKey, "upload_id": args.UploadID, "path": args.StoragePath,
	})
	// #endregion
	repo := gisrepo.New(r.db)
	svc := gis.NewShapefileImportService(r.cfg, repo, r.logger)
	importErr := svc.Import(ctx, args)
	// #region agent log
	debuglog.Agent("SHP-C", "handlers.go:handleGISShapefileImport", "job finished", map[string]any{
		"job_id": job.ID, "source_key": args.SourceKey, "ok": importErr == nil,
		"error": func() string {
			if importErr != nil {
				return importErr.Error()
			}
			return ""
		}(),
	})
	// #endregion
	return importErr
}

func (r *Registry) handleCryptoRefresh(ctx context.Context, job *queue.ReservedJob) error {
	return r.crypto.Refresh(ctx)
}

func (r *Registry) handleFEMAFloodEnrichKickoff(ctx context.Context, job *queue.ReservedJob) error {
	args, err := fema.UnmarshalKickoffArgs(job.Payload.Args)
	if err != nil {
		return err
	}
	return r.femaEnrich.Kickoff(ctx, args)
}

func (r *Registry) handleFEMAFloodEnrichBatch(ctx context.Context, job *queue.ReservedJob) error {
	args, err := fema.UnmarshalBatchArgs(job.Payload.Args)
	if err != nil {
		return err
	}
	return r.femaEnrich.RunBatch(ctx, args)
}

func (r *Registry) handleGeocodeListingsKickoff(ctx context.Context, job *queue.ReservedJob) error {
	args, err := geocode.UnmarshalGeocodeKickoffArgs(job.Payload.Args)
	if err != nil {
		return err
	}
	return r.geocodeEnrich.Kickoff(ctx, args)
}

func (r *Registry) handleGeocodeListingsBatch(ctx context.Context, job *queue.ReservedJob) error {
	args, err := geocode.UnmarshalGeocodeBatchArgs(job.Payload.Args)
	if err != nil {
		return err
	}
	return r.geocodeEnrich.RunBatch(ctx, args)
}
