package dashboard

import (
	"sort"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// KnownQueueNames returns configured worker and sync queue names for monitoring rollups.
func KnownQueueNames(cfg config.Config) []string {
	seen := make(map[string]struct{})
	var names []string
	add := func(q string) {
		if q == "" {
			return
		}
		if _, ok := seen[q]; ok {
			return
		}
		seen[q] = struct{}{}
		names = append(names, q)
	}
	for _, q := range cfg.Queue.WorkerQueues {
		add(q)
	}
	add(cfg.MLS.SyncKickoffQueue)
	add(cfg.Bridge.SyncFetchQueue)
	add(cfg.Bridge.SyncPersistQueue)
	add(cfg.Spark.SyncFetchQueue)
	add(cfg.Spark.SyncPersistQueue)
	add(cfg.GIS.SyncQueue)
	add(cfg.GIS.ImportQueue)
	add(cfg.GIS.Queue)
	add(cfg.FEMA.EnrichQueue)
	add(cfg.Geocode.EnrichQueue)
	add(cfg.Coingecko.Queue)
	sort.Strings(names)
	return names
}

// MergeKnownQueues fills missing configured queue names with zero counts.
func MergeKnownQueues(cfg config.Config, counts []repository.QueueCount) []repository.QueueCount {
	byName := make(map[string]repository.QueueCount, len(counts)+8)
	for _, q := range counts {
		byName[q.Queue] = q
	}
	for _, name := range KnownQueueNames(cfg) {
		if _, ok := byName[name]; !ok {
			byName[name] = repository.QueueCount{Queue: name}
		}
	}
	out := make([]repository.QueueCount, 0, len(byName))
	for _, q := range byName {
		out = append(out, q)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Queue < out[j].Queue })
	return out
}
