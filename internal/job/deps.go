package job

import (
	"github.com/quantyralabs/idx-api/internal/queue"
	"github.com/quantyralabs/idx-api/internal/service/cache"
	"github.com/quantyralabs/idx-api/internal/service/crypto"
	"github.com/quantyralabs/idx-api/internal/service/gis"
	"github.com/quantyralabs/idx-api/internal/service/sync"
)

// InitServices attaches service implementations (called from cmd/worker).
func (r *Registry) InitServices(q *queue.Client) {
	r.replicationKickoff = sync.NewKickoff(r.cfg, r.db, q, r.logger)
	r.listingsCache = cache.NewRefreshJob(r.cfg, r.db, r.logger)
	r.bridgeSync = sync.NewBridgeWorker(r.cfg, r.db, q, r.logger)
	r.sparkSync = sync.NewSparkWorker(r.cfg, r.db, q, r.logger)
	r.mirrorPurge = sync.NewPurgeClosed(r.cfg, r.db)
	r.replicaStore = sync.NewReplicaPageStore(r.db, r.cfg)
	r.gisMeta = gis.NewMetadataService(r.cfg, r.db, r.logger)
	r.crypto = crypto.NewPricingService(r.cfg, r.db, r.logger)
}
