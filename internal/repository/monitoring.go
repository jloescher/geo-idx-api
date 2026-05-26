package repository

import (
	"context"
	"time"
)

// MonitoringRepo aggregates ops metrics for the admin dashboard.
type MonitoringRepo struct {
	db *DB
}

// NewMonitoringRepo creates a monitoring repository.
func NewMonitoringRepo(db *DB) *MonitoringRepo {
	return &MonitoringRepo{db: db}
}

// ListingDatasetRow is per-dataset listing mirror stats.
type ListingDatasetRow struct {
	DatasetSlug            string
	Total                  int64
	ActivePending          int64
	OldestModification     *time.Time
	LatestModification     *time.Time
	CursorLastModification *time.Time
	LastSyncFinishedAt     *time.Time
	ReplicationInProgress  bool
}

// ListListingStats returns listing counts and sync cursor fields per dataset.
func (r *MonitoringRepo) ListListingStats(ctx context.Context) ([]ListingDatasetRow, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT COALESCE(c.dataset_slug, s.dataset_slug) AS dataset_slug,
		       COALESCE(s.total, 0) AS total,
		       COALESCE(s.active_pending, 0) AS active_pending,
		       s.oldest_mod,
		       s.latest_mod,
		       c.last_modification_timestamp,
		       c.last_sync_finished_at,
		       COALESCE(c.replication_in_progress, false) AS replication_in_progress
		FROM listing_sync_cursors c
		FULL OUTER JOIN (
			SELECT dataset_slug,
			       COUNT(*) AS total,
			       COUNT(*) FILTER (WHERE LOWER(TRIM(COALESCE(standard_status,''))) IN ('active','pending')) AS active_pending,
			       MIN(modification_timestamp) AS oldest_mod,
			       MAX(modification_timestamp) AS latest_mod
			FROM listings
			GROUP BY dataset_slug
		) s ON s.dataset_slug = c.dataset_slug
		ORDER BY dataset_slug
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ListingDatasetRow
	for rows.Next() {
		var row ListingDatasetRow
		if err := rows.Scan(&row.DatasetSlug, &row.Total, &row.ActivePending, &row.OldestModification,
			&row.LatestModification, &row.CursorLastModification, &row.LastSyncFinishedAt, &row.ReplicationInProgress); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// CacheHitStats is MLS proxy cache hit rate for a time window.
type CacheHitStats struct {
	Total      int64
	Hits       int64
	Misses     int64
	HitRatePct float64
}

// CacheHitRate15m aggregates cache hits from audit logs in the last 15 minutes.
func (r *MonitoringRepo) CacheHitRate15m(ctx context.Context) (CacheHitStats, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return CacheHitStats{}, err
	}
	var s CacheHitStats
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) AS total,
		       COUNT(*) FILTER (WHERE cache_hit = 'hit') AS hits,
		       COUNT(*) FILTER (WHERE cache_hit = 'miss') AS misses
		FROM mls_proxy_audit_logs
		WHERE logged_at >= NOW() - INTERVAL '15 minutes'
	`).Scan(&s.Total, &s.Hits, &s.Misses)
	if err != nil {
		return CacheHitStats{}, err
	}
	if s.Total > 0 {
		s.HitRatePct = float64(s.Hits) / float64(s.Total) * 100
	}
	return s, nil
}

// QueueCount is pending/reserved/failed jobs for a queue name.
type QueueCount struct {
	Queue    string `json:"queue"`
	Pending  int64  `json:"pending"`
	Reserved int64  `json:"reserved"`
	Failed   int64  `json:"failed"`
}

// ListQueueCounts returns job counts grouped by queue.
func (r *MonitoringRepo) ListQueueCounts(ctx context.Context) ([]QueueCount, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT q.queue,
		       COALESCE(j.pending, 0) AS pending,
		       COALESCE(j.reserved, 0) AS reserved,
		       COALESCE(f.failed, 0) AS failed
		FROM (
			SELECT DISTINCT queue FROM jobs
			UNION
			SELECT DISTINCT queue FROM failed_jobs
		) q
		LEFT JOIN (
			SELECT queue,
			       COUNT(*) FILTER (WHERE reserved_at IS NULL AND available_at <= EXTRACT(EPOCH FROM NOW())::bigint) AS pending,
			       COUNT(*) FILTER (WHERE reserved_at IS NOT NULL) AS reserved
			FROM jobs
			GROUP BY queue
		) j ON j.queue = q.queue
		LEFT JOIN (
			SELECT queue, COUNT(*) AS failed FROM failed_jobs GROUP BY queue
		) f ON f.queue = q.queue
		ORDER BY q.queue
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QueueCount
	for rows.Next() {
		var row QueueCount
		if err := rows.Scan(&row.Queue, &row.Pending, &row.Reserved, &row.Failed); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// JobTypeCount is pending jobs grouped by payload type.
type JobTypeCount struct {
	Queue   string `json:"queue"`
	JobType string `json:"job_type"`
	Count   int64  `json:"count"`
}

// TopPendingJobTypes returns the busiest pending job types (limit 10).
func (r *MonitoringRepo) TopPendingJobTypes(ctx context.Context, limit int) ([]JobTypeCount, error) {
	if limit < 1 {
		limit = 10
	}
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, `
		SELECT queue,
		       COALESCE(NULLIF(payload::json->>'type', ''), 'unknown') AS job_type,
		       COUNT(*) AS pending
		FROM jobs
		WHERE reserved_at IS NULL AND available_at <= EXTRACT(EPOCH FROM NOW())::bigint
		GROUP BY queue, job_type
		ORDER BY pending DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JobTypeCount
	for rows.Next() {
		var row JobTypeCount
		if err := rows.Scan(&row.Queue, &row.JobType, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// ActivationStats counts onboarding-related platform metrics.
type ActivationStats struct {
	DomainCount           int64
	VerifiedDomainCount   int64
	TokenCount            int64
	InvitationsAccepted   int64
	DomainsWithTraffic30d int64
}

// Activation returns global activation counters (all users).
func (r *MonitoringRepo) Activation(ctx context.Context) (ActivationStats, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return ActivationStats{}, err
	}
	var s ActivationStats
	err = pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM domains),
			(SELECT COUNT(*) FROM domains WHERE verification_status IN ('verified', 'verified_ghl')),
			(SELECT COUNT(*) FROM personal_access_tokens WHERE tokenable_type = 'App\\Models\\User'),
			(SELECT COUNT(*) FROM user_invitations WHERE accepted_at IS NOT NULL),
			(SELECT COUNT(DISTINCT domain_slug) FROM mls_proxy_audit_logs
			 WHERE logged_at >= NOW() - INTERVAL '30 days' AND domain_slug IS NOT NULL)
	`).Scan(&s.DomainCount, &s.VerifiedDomainCount, &s.TokenCount, &s.InvitationsAccepted, &s.DomainsWithTraffic30d)
	return s, err
}

// UserSetupStats tracks setup progress for a dashboard user.
type UserSetupStats struct {
	DomainCount         int64
	VerifiedDomainCount int64
	HasAuditTraffic30d  bool
}

// UserSetup returns per-user setup progress signals.
func (r *MonitoringRepo) UserSetup(ctx context.Context, userID int64) (UserSetupStats, error) {
	pool, err := r.db.ReadPool(ctx)
	if err != nil {
		return UserSetupStats{}, err
	}
	var s UserSetupStats
	err = pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM domains WHERE user_id = $1),
			(SELECT COUNT(*) FROM domains WHERE user_id = $1 AND verification_status IN ('verified', 'verified_ghl')),
			EXISTS (
				SELECT 1 FROM mls_proxy_audit_logs a
				INNER JOIN domains d ON d.domain_slug = a.domain_slug
				WHERE d.user_id = $1 AND a.logged_at >= NOW() - INTERVAL '30 days'
			)
	`, userID).Scan(&s.DomainCount, &s.VerifiedDomainCount, &s.HasAuditTraffic30d)
	return s, err
}
