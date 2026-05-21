package queue

import "testing"

func TestPartitionWorkerQueues(t *testing.T) {
	fetch, persist, other := partitionWorkerQueues([]string{
		"default",
		"sync-kickoff",
		"bridge-sync-fetch",
		"spark-sync-fetch",
		"bridge-sync-persist",
		"spark-sync-persist",
	})
	if len(fetch) != 2 || len(persist) != 2 || len(other) != 2 {
		t.Fatalf("unexpected partition fetch=%v persist=%v other=%v", fetch, persist, other)
	}
}
