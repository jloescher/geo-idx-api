package sync

import (
	"context"
	"hash/fnv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// reconcileRunLockClass is the pg_advisory_xact_lock class id for mirror key reconcile pages.
const reconcileRunLockClass int32 = 84729103

func reconcileRunLockKeys(runID uuid.UUID) (int32, int32) {
	h := fnv.New64a()
	_, _ = h.Write(runID[:])
	sum := h.Sum64()
	return reconcileRunLockClass, int32(sum >> 32) //nolint:gosec // advisory lock key space, not security
}

type reconcileLockQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// TryAcquireReconcileRunLock serializes reconcile page jobs for one run_id within a transaction.
func TryAcquireReconcileRunLock(ctx context.Context, q reconcileLockQueryer, runID uuid.UUID) (bool, error) {
	k1, k2 := reconcileRunLockKeys(runID)
	var ok bool
	err := q.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock($1, $2)`, k1, k2).Scan(&ok)
	return ok, err
}
