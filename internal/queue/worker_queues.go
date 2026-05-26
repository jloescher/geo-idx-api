package queue

import "strings"

// partitionWorkerQueues splits configured queues into MLS fetch, MLS persist, and other pools.
func partitionWorkerQueues(queues []string) (fetch, persist, other []string) {
	for _, q := range queues {
		switch {
		case strings.HasSuffix(q, "-sync-fetch"):
			fetch = append(fetch, q)
		case strings.HasSuffix(q, "-sync-persist"):
			persist = append(persist, q)
		default:
			other = append(other, q)
		}
	}
	return fetch, persist, other
}
