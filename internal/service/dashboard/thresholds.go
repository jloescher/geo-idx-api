package dashboard

import (
	"strconv"
	"strings"

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

// ListingDatasetStatus labels listing mirror health for the ingest tab.
// Active replica_pages work (pending/processing) is catch-up, not a stale mirror.
func ListingDatasetStatus(isCurrent, hasActiveReplica bool) string {
	if isCurrent {
		return "healthy"
	}
	if hasActiveReplica {
		return "catching_up"
	}
	return "stale"
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
	SchedulerHolderPID     *int32
	SchedulerLastEnqueue   string
	SchedulerBackendCount  int64
	TotalStaleReserved     int64
	StaleReservedAfterSecs int
	StaleInFlight          []repository.InFlightJob
	TotalFailed            int64
	SyncPipelineStatus     string
}

// StaleReservedIncidentDetail builds warning copy for stale reserved queue jobs.
func StaleReservedIncidentDetail(in IncidentInput) string {
	base := "Jobs reserved longer than " + formatStaleReservedThreshold(in.StaleReservedAfterSecs) + " (half of DB_QUEUE_RESERVATION_TIMEOUT). Workers may have crashed or the job is stuck."
	if len(in.StaleInFlight) == 0 {
		return base + " Check Queues → In-flight jobs."
	}
	const max = 3
	var parts []string
	for i, job := range in.StaleInFlight {
		if i >= max {
			break
		}
		parts = append(parts, job.Queue+" "+job.JobType+" (id "+formatInt64(job.JobID)+", "+formatInt64(job.AgeSeconds)+"s)")
	}
	return base + " Examples: " + strings.Join(parts, "; ") + "."
}

func formatStaleReservedThreshold(sec int) string {
	if sec < 60 {
		return strconv.Itoa(sec) + "s"
	}
	return strconv.Itoa(sec/60) + "m"
}

func formatInt64(n int64) string {
	return strconv.FormatInt(n, 10)
}

// filterStaleInFlight returns reserved jobs past the stale threshold for incident copy.
func filterStaleInFlight(jobs []repository.InFlightJob) []repository.InFlightJob {
	out := make([]repository.InFlightJob, 0, len(jobs))
	for _, j := range jobs {
		if j.Stale {
			out = append(out, j)
		}
	}
	return out
}

func schedulerLeaderIncidentDetail(in IncidentInput) string {
	if in.SchedulerLeaderActive {
		return ""
	}
	if in.SchedulerBackendCount > 0 {
		return "Scheduler processes are connected (pg_stat_activity application_name idx-scheduler) but no advisory lock is held on the primary. Restart both scheduler containers; confirm DB_RW_DSN reaches the Patroni primary (prefer :5432 or keepalives on :5000)."
	}
	if in.SchedulerLastEnqueue != "" {
		return "No advisory lock on the Patroni primary, but scheduler-owned jobs were enqueued recently (" + in.SchedulerLastEnqueue + "). The leader session may have lost its lock (HAProxy idle on :5000). Restart schedulers or use Patroni :5432 / keepalives on DB_RW_DSN."
	}
	if in.SchedulerHolderPID != nil && *in.SchedulerHolderPID > 0 {
		return "Unexpected: holder PID present without leader_active. Check pg_locks on the primary and restart API containers if an old pool probe leaked a lock."
	}
	return "No granted advisory lock on the Patroni primary (classid=0). Confirm schedulers on re-db and re-node-02 log leader acquired; restart API containers once after the pg_locks monitoring fix to clear leaked locks."
}

// BuildIncidents returns warning/critical incidents from monitoring health signals.
func BuildIncidents(in IncidentInput) []IncidentMetric {
	var out []IncidentMetric
	if in.InfraStatus == "critical" || !in.SchedulerLeaderActive {
		out = append(out, IncidentMetric{
			Severity: "critical",
			Source:   "infrastructure",
			Title:    "Scheduler leader not detected",
			Detail:   schedulerLeaderIncidentDetail(in),
		})
	}
	if in.TotalStaleReserved > 0 {
		out = append(out, IncidentMetric{
			Severity: "warning",
			Source:   "queues",
			Title:    "Stale reserved jobs detected",
			Detail:   StaleReservedIncidentDetail(in),
		})
	}
	if in.TotalFailed > 0 {
		out = append(out, IncidentMetric{
			Severity: "warning",
			Source:   "queues",
			Title:    "Failed jobs present (7d)",
			Detail:   "failed_jobs has rows from the last 7 days. After workers handle sync-kickoff, purge resolved historical rows on the primary or fix recurring failures (see Queues tab).",
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
