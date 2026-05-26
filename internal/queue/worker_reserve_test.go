package queue

import "testing"

func TestReserveWeightedPrefersAlternatingPools(t *testing.T) {
	w := &Worker{
		client: nil,
		queues: []string{"bridge-sync-fetch", "spark-sync-fetch", "bridge-sync-persist", "spark-sync-persist", "default"},
	}
	fetchQ, persistQ, otherQ := partitionWorkerQueues(w.queues)
	if len(fetchQ) != 2 || len(persistQ) != 2 || len(otherQ) != 1 {
		t.Fatalf("partition mismatch fetch=%v persist=%v other=%v", fetchQ, persistQ, otherQ)
	}
}
