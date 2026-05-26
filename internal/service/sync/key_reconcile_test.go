package sync

import (
	"testing"

	"github.com/google/uuid"
	"github.com/quantyralabs/idx-api/internal/queue"
)

func TestValidateReconcileDelete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		keysSeen   int64
		mirror     int64
		wantErr    bool
	}{
		{"empty mirror", 0, 0, false},
		{"zero keys with mirror", 0, 100, true},
		{"below half large mirror", 100, 500, true},
		{"at half large mirror", 250, 500, false},
		{"small mirror zero keys", 0, 10, true},
		{"small mirror any keys", 5, 10, false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateReconcileDelete(tc.keysSeen, tc.mirror)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestReconcileRunLockKeysStable(t *testing.T) {
	t.Parallel()
	runID := uuid.MustParse("00000000-0000-0000-0000-000000000102")
	k1a, k2a := reconcileRunLockKeys(runID)
	k1b, k2b := reconcileRunLockKeys(runID)
	if k1a != k1b || k2a != k2b {
		t.Fatalf("lock keys not stable: (%d,%d) vs (%d,%d)", k1a, k2a, k1b, k2b)
	}
	if k1a != reconcileRunLockClass {
		t.Fatalf("class = %d want %d", k1a, reconcileRunLockClass)
	}
}

func TestQueueIsReconcileBusy(t *testing.T) {
	t.Parallel()
	err := queue.ErrReconcileBusy{RunID: "abc"}
	if !queue.IsReconcileBusy(err) {
		t.Fatal("expected reconcile busy")
	}
}
