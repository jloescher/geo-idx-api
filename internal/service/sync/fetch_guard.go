package sync

import (
	"context"
	"log/slog"
)

// skipReplicationFetchWhenPageActive avoids overlapping replication HTTP while a replica page is staged.
func skipReplicationFetchWhenPageActive(
	ctx context.Context,
	store *ReplicaPageStore,
	logger *slog.Logger,
	provider, dataset, mode string,
) (bool, error) {
	if mode != "replication" {
		return false, nil
	}
	active, err := store.HasActivePage(ctx, provider, dataset)
	if err != nil {
		return true, err
	}
	if active {
		logger.Debug("fetch skipped: active replica page",
			"provider", provider, "dataset", dataset, "mode", mode)
		return true, nil
	}
	return false, nil
}
