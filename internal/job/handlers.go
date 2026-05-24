package job

import (
	"context"

	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/service/gis"
)

func (r *Registry) handleReplicationKickoff(ctx context.Context, job *queue.ReservedJob) error {
	r.logger.Info("running replication kickoff", "job_id", job.ID)
	return r.replicationKickoff.Run(ctx)
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

func (r *Registry) handlePurgeReplicaPages(ctx context.Context, job *queue.ReservedJob) error {
	return r.replicaStore.PurgeEligible(ctx)
}

func (r *Registry) handleGISProbe(ctx context.Context, job *queue.ReservedJob) error {
	return r.gisMeta.ProbeAll(ctx)
}

func (r *Registry) handleGISMonthlyParcelRefresh(ctx context.Context, job *queue.ReservedJob) error {
	return r.gisParcelSync.RunMonthlyRefresh(ctx)
}

func (r *Registry) handleGISAnnualBoundariesRefresh(ctx context.Context, job *queue.ReservedJob) error {
	return r.gisBoundarySync.RunAnnualRefresh(ctx)
}

func (r *Registry) handleGISInitialSync(ctx context.Context, job *queue.ReservedJob) error {
	return r.gisInitialSync.Run(ctx)
}

func (r *Registry) handleGISParcelSyncPage(ctx context.Context, job *queue.ReservedJob) error {
	args, err := gis.UnmarshalParcelSyncPageArgs(job.Payload.Args)
	if err != nil {
		return err
	}
	return r.gisParcelSync.SyncPage(ctx, args)
}

func (r *Registry) handleCryptoRefresh(ctx context.Context, job *queue.ReservedJob) error {
	return r.crypto.Refresh(ctx)
}
