package dashboard_test

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/dashboard"
)

func TestMergeKnownQueuesFillsZeros(t *testing.T) {
	cfg := config.Config{
		Queue: config.QueueConfig{
			WorkerQueues: []string{"default", "sync-kickoff"},
		},
		MLS:    config.MLSConfig{SyncKickoffQueue: "sync-kickoff"},
		Bridge: config.BridgeConfig{SyncFetchQueue: "bridge-sync-fetch", SyncPersistQueue: "bridge-sync-persist"},
		Spark:  config.SparkConfig{SyncFetchQueue: "spark-sync-fetch", SyncPersistQueue: "spark-sync-persist"},
		GIS:    config.GISConfig{SyncQueue: "default", Queue: "default"},
		FEMA:   config.FEMAConfig{EnrichQueue: "default"},
		Geocode: config.GeocodeConfig{EnrichQueue: "default"},
		Coingecko: config.CoingeckoConfig{Queue: "default"},
	}
	merged := dashboard.MergeKnownQueues(cfg, nil)
	if len(merged) == 0 {
		t.Fatal("expected merged queues")
	}
	byName := map[string]repository.QueueCount{}
	for _, q := range merged {
		byName[q.Queue] = q
	}
	if _, ok := byName["sync-kickoff"]; !ok {
		t.Fatal("missing sync-kickoff")
	}
	if byName["sync-kickoff"].Pending != 0 {
		t.Fatalf("expected zero pending, got %d", byName["sync-kickoff"].Pending)
	}
}

func TestKnownQueueNamesDedupesKickoff(t *testing.T) {
	cfg := config.Config{
		Queue: config.QueueConfig{WorkerQueues: []string{"sync-kickoff", "default"}},
		MLS:   config.MLSConfig{SyncKickoffQueue: "sync-kickoff"},
	}
	names := dashboard.KnownQueueNames(cfg)
	var count int
	for _, n := range names {
		if n == "sync-kickoff" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one sync-kickoff entry, got %d", count)
	}
}
