package job

import (
	"github.com/quantyralabs/idx-api/internal/queue"
	gisrepo "github.com/quantyralabs/idx-api/internal/repository/gis"
	"github.com/quantyralabs/idx-api/internal/service/cache"
	"github.com/quantyralabs/idx-api/internal/service/crypto"
	"github.com/quantyralabs/idx-api/internal/service/fema"
	"github.com/quantyralabs/idx-api/internal/service/gis"
	"github.com/quantyralabs/idx-api/internal/service/sync"
)

// InitServices attaches service implementations (called from cmd/worker).
func (r *Registry) InitServices(q *queue.Client) {
	r.replicationKickoff = sync.NewKickoff(r.cfg, r.db, q, r.logger)
	r.proxyCachePurge = cache.NewRefreshJob(r.cfg, r.db, r.logger)
	r.bridgeSync = sync.NewBridgeWorker(r.cfg, r.db, q, r.logger)
	r.sparkSync = sync.NewSparkWorker(r.cfg, r.db, q, r.logger)
	r.mirrorPurge = sync.NewPurgeClosed(r.cfg, r.db)
	r.replicaStore = sync.NewReplicaPageStore(r.db, r.cfg)
	r.keyReconcile = sync.NewKeyReconcile(r.cfg, r.db, q, r.logger)
	r.gisMeta = gis.NewMetadataService(r.cfg, r.db, r.logger)
	gisRepo := gisrepo.New(r.db)
	r.gisPersistent = gis.NewPersistentGISService(r.cfg, gisRepo, q, r.logger)
	r.crypto = crypto.NewPricingService(r.cfg, r.db, r.logger)
	r.femaEnrich = fema.NewEnrichmentService(r.cfg, r.db, q, r.logger)
	r.bridgeSync.SetFEMAEnrichment(r.femaEnrich)
	r.sparkSync.SetFEMAEnrichment(r.femaEnrich)
}
