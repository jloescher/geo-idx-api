package job

import (
	"context"
	"log/slog"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/cache"
	"github.com/quantyralabs/idx-api/internal/service/crypto"
	"github.com/quantyralabs/idx-api/internal/service/gis"
	"github.com/quantyralabs/idx-api/internal/service/sync"
)

// Registry wires queue handlers for all job types.
type Registry struct {
	cfg    config.Config
	db     *repository.DB
	logger *slog.Logger

	replicationKickoff *sync.Kickoff
	proxyCachePurge    *cache.RefreshJob
	bridgeSync         *sync.BridgeWorker
	sparkSync          *sync.SparkWorker
	mirrorPurge        *sync.PurgeClosed
	replicaStore       *sync.ReplicaPageStore
	gisMeta            *gis.MetadataService
	gisParcelSync      *gis.ParcelSyncService
	gisBoundarySync    *gis.BoundarySyncService
	gisInitialSync     *gis.InitialSyncService
	crypto             *crypto.PricingService
}

func NewRegistry(cfg config.Config, db *repository.DB, logger *slog.Logger) *Registry {
	return &Registry{cfg: cfg, db: db, logger: logger}
}

// RegisterAll attaches handlers to the worker.
func (r *Registry) RegisterAll(w *queue.Worker) {
	w.Register(queue.TypeNoop, r.handleNoop)
	w.Register(queue.TypeMLSReplicationKickoff, r.handleReplicationKickoff)
	w.Register(queue.TypeMLSProxyCachePurge, r.handleProxyCachePurge)
	w.Register(queue.TypeBridgeFetchPage, r.handleBridgeFetchPage)
	w.Register(queue.TypeBridgePersistChunk, r.handleBridgePersistChunk)
	w.Register(queue.TypeBridgePersistFinalize, r.handleBridgePersistFinalize)
	w.Register(queue.TypeSparkFetchPage, r.handleSparkFetchPage)
	w.Register(queue.TypeSparkPersistChunk, r.handleSparkPersistChunk)
	w.Register(queue.TypeSparkPersistFinalize, r.handleSparkPersistFinalize)
	w.Register(queue.TypeMLSPurgeClosed, r.handlePurgeClosed)
	w.Register(queue.TypeMLSPurgeReplicaPages, r.handlePurgeReplicaPages)
	w.Register(queue.TypeGISProbeSources, r.handleGISProbe)
	w.Register(queue.TypeGISMonthlyParcelRefresh, r.handleGISMonthlyParcelRefresh)
	w.Register(queue.TypeGISAnnualBoundariesRefresh, r.handleGISAnnualBoundariesRefresh)
	w.Register(queue.TypeGISInitialSync, r.handleGISInitialSync)
	w.Register(queue.TypeGISZipSync, r.handleGISZipSync)
	w.Register(queue.TypeGISParcelSyncPage, r.handleGISParcelSyncPage)
	w.Register(queue.TypeCryptoRefreshPricing, r.handleCryptoRefresh)
}

func (r *Registry) handleNoop(ctx context.Context, job *queue.ReservedJob) error {
	r.logger.Debug("noop job", "id", job.ID)
	return nil
}
