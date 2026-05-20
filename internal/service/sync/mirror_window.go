package sync

import (
	"fmt"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
)

const activePendingStatusFilter = "(StandardStatus eq 'Active' or StandardStatus eq 'Pending')"

// MirrorRollingCutoff returns the oldest modification time to include in the mirror, or nil for all-time.
func MirrorRollingCutoff(cfg config.Config) *time.Time {
	months := cfg.MLS.LocalMirrorRollingMonths
	if months <= 0 {
		return nil
	}
	t := time.Now().UTC().AddDate(0, -months, 0)
	return &t
}

// BridgeReplicationFilter is the OData $filter for Bridge Property/replication seed pages.
func BridgeReplicationFilter(cfg config.Config) string {
	cutoff := MirrorRollingCutoff(cfg)
	if cutoff == nil {
		return activePendingStatusFilter
	}
	// Bridge recommends BridgeModificationTimestamp for incremental/replication windows.
	return activePendingStatusFilter + " and BridgeModificationTimestamp gt " + bridgeRollingTimestampLiteral(*cutoff)
}

// SparkReplicationFilter is the OData $filter for Spark replication seed pages.
func SparkReplicationFilter(cfg config.Config) string {
	return replicationStatusFilter(cfg, sparkRollingTimestampLiteral)
}

func replicationStatusFilter(cfg config.Config, tsFn func(time.Time) string) string {
	cutoff := MirrorRollingCutoff(cfg)
	if cutoff == nil {
		return activePendingStatusFilter
	}
	return activePendingStatusFilter + " and ModificationTimestamp gt " + tsFn(*cutoff)
}

func bridgeRollingTimestampLiteral(t time.Time) string {
	return "datetime'" + t.UTC().Format("2006-01-02T15:04:05Z") + "'"
}

func sparkRollingTimestampLiteral(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

// MirrorRollingMonthsDescription documents effective window for ops logs.
func MirrorRollingMonthsDescription(cfg config.Config) string {
	m := cfg.MLS.LocalMirrorRollingMonths
	if m <= 0 {
		return "all-time"
	}
	return fmt.Sprintf("%d-month", m)
}
