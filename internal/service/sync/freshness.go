package sync

import (
	"context"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

const (
	ModeCatchUp = "catch_up"
	ModeSteady  = "steady"
)

// Freshness decides catch-up vs steady replication scheduling.
// Revenue impact: catch-up maximizes throughput until maps meet freshness SLA.
type Freshness struct {
	cfg     config.Config
	cursors *CursorStore
	pages   *ReplicaPageStore
}

func NewFreshness(cfg config.Config, db *repository.DB) *Freshness {
	return &Freshness{
		cfg:     cfg,
		cursors: NewCursorStore(db),
		pages:   NewReplicaPageStore(db, cfg),
	}
}

func (f *Freshness) Mode(ctx context.Context, dataset, provider string) (string, error) {
	current, err := f.IsCurrent(ctx, dataset, provider)
	if err != nil {
		return "", err
	}
	if current {
		return ModeSteady, nil
	}
	return ModeCatchUp, nil
}

func (f *Freshness) IsCurrent(ctx context.Context, dataset, provider string) (bool, error) {
	active, err := f.pages.HasActivePage(ctx, provider, dataset)
	if err != nil || active {
		return false, err
	}

	c, err := f.cursors.ForDataset(ctx, dataset)
	if err != nil {
		return false, err
	}
	if c.ReplicationInProgress {
		return false, nil
	}
	if c.ReplicationNextURL != nil && *c.ReplicationNextURL != "" {
		return false, nil
	}

	seeded := c.LastSyncFinishedAt != nil
	if !seeded {
		ok, err := f.cursors.MirrorSeeded(ctx, dataset)
		if err != nil {
			return false, err
		}
		seeded = ok
	}
	if !seeded {
		return false, nil
	}
	if c.LastModificationTimestamp == nil {
		return false, nil
	}
	if c.LastSyncFinishedAt == nil {
		return false, nil
	}

	// "Current" = last completed sync cycle (replication or incremental) finished within the freshness window.
	// Do not use LastModificationTimestamp here — listing mod times can be recent even when we have not polled Bridge.
	threshold := f.cfg.MLS.ReplicationFreshnessMinutes
	if threshold < 1 {
		threshold = 15
	}
	cutoff := time.Now().Add(-time.Duration(threshold) * time.Minute)
	return !c.LastSyncFinishedAt.Before(cutoff), nil
}
