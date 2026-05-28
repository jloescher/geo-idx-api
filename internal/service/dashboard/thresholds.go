package dashboard

import (
	"github.com/quantyralabs/idx-api/internal/repository"
)

// Monitoring thresholds (aligned with dashboard.js where noted).
const (
	QueuePendingStaleThreshold   int64 = 500
	ReplicaPendingStaleThreshold int64 = 500
	StaleReservedMinSeconds      int   = 600 // floor when deriving from reservation timeout
)

// StaleReservedAfter returns how long a reserved job must sit before counting as stale.
// Uses half of the worker reservation timeout, with a 10-minute minimum.
func StaleReservedAfter(reservationTimeoutSeconds int) int {
	if reservationTimeoutSeconds <= 0 {
		reservationTimeoutSeconds = 3600
	}
	after := reservationTimeoutSeconds / 2
	if after < StaleReservedMinSeconds {
		after = StaleReservedMinSeconds
	}
	return after
}

// CacheStatus derives cache health from 15m audit volume.
func CacheStatus(total int64, hitRatePct float64) string {
	if total == 0 {
		return "no_data"
	}
	if hitRatePct >= 0 {
		return "healthy"
	}
	return "unknown"
}

// QueueStatus aggregates queue depth signals for tiles and incidents.
func QueueStatus(pending, staleReserved, totalFailed int64) string {
	if totalFailed > 0 {
		return "stale"
	}
	if pending > QueuePendingStaleThreshold || staleReserved > 0 {
		return "stale"
	}
	return "healthy"
}

// SyncPipelineStatus evaluates replica_pages pipeline rows.
func SyncPipelineStatus(rows []repository.ReplicaPageStatusCount) string {
	for _, row := range rows {
		switch row.Status {
		case "failed":
			return "stale"
		case "pending":
			if row.Count > ReplicaPendingStaleThreshold {
				return "stale"
			}
		}
	}
	return "healthy"
}

// InfraStatus maps scheduler lock probe to infrastructure rollup.
func InfraStatus(leaderActive bool) string {
	if leaderActive {
		return "healthy"
	}
	return "critical"
}

// ReplicaPageChipStatus maps a single replica_pages row to a UI chip severity.
func ReplicaPageChipStatus(status string, count int64) string {
	switch status {
	case "failed":
		return "critical"
	case "pending":
		if count > ReplicaPendingStaleThreshold {
			return "stale"
		}
		return "healthy"
	default:
		return "healthy"
	}
}

// QueueTileChipStatus maps per-queue counts to a UI chip (mirrors QueueStatus per queue).
func QueueTileChipStatus(pending, failed int64) string {
	if failed > 0 {
		return "stale"
	}
	if pending > QueuePendingStaleThreshold {
		return "stale"
	}
	return "healthy"
}

// IncidentInput carries snapshot fields needed to build open incidents.
type IncidentInput struct {
	InfraStatus            string
	SchedulerLeaderActive  bool
	TotalStaleReserved     int64
	StaleReservedAfterSecs int
	TotalFailed            int64
	SyncPipelineStatus     string
}

// BuildIncidents returns warning/critical incidents from monitoring health signals.
func BuildIncidents(in IncidentInput) []IncidentMetric {
	var out []IncidentMetric
	if in.InfraStatus == "critical" || !in.SchedulerLeaderActive {
		out = append(out, IncidentMetric{
			Severity: "critical",
			Source:   "infrastructure",
			Title:    "Scheduler leader not detected",
			Detail:   "No process currently holds the scheduler advisory lock.",
		})
	}
	if in.TotalStaleReserved > 0 {
		out = append(out, IncidentMetric{
			Severity: "warning",
			Source:   "queues",
			Title:    "Stale reserved jobs detected",
			Detail:   "At least one queue has jobs reserved longer than the monitoring stale threshold.",
		})
	}
	if in.TotalFailed > 0 {
		out = append(out, IncidentMetric{
			Severity: "warning",
			Source:   "queues",
			Title:    "Failed jobs present",
			Detail:   "One or more queues have rows in failed_jobs; review the Queues tab for types and exceptions.",
		})
	}
	if in.SyncPipelineStatus == "stale" {
		out = append(out, IncidentMetric{
			Severity: "warning",
			Source:   "sync",
			Title:    "Replica page backlog is degraded",
			Detail:   "Replica pages include failed statuses or unusually high pending volume.",
		})
	}
	return out
}
