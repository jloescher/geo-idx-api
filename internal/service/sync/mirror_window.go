package sync

import (
	"fmt"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/service/mls"
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
//
// Stellar /replication accepts StandardStatus filters only; BridgeModificationTimestamp
// (or ModificationTimestamp) in $filter returns HTTP 400. MLS_LOCAL_MIRROR_ROLLING_MONTHS
// is enforced after persist via purge (modification_timestamp), not on replication fetch.
func BridgeReplicationFilter(cfg config.Config) string {
	_ = cfg // rolling window does not apply to Bridge replication OData
	return activePendingStatusFilter
}

// BridgeIncrementalFilter is the OData $filter for Bridge Property incremental updates (post-replication).
func BridgeIncrementalFilter(dataset string, since time.Time) string {
	return "(" + activePendingStatusFilter + ") and " + bridgeIncrementalTimestampClause(dataset, since)
}

func bridgeIncrementalTimestampClause(dataset string, since time.Time) string {
	return mls.ODataGTFilter(mls.ModificationODataField(dataset), since)
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
