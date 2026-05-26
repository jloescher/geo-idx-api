package queue_test

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/queue"
)

func TestIsLegacyLaravelPayload(t *testing.T) {
	laravel := []byte(`{"uuid":"x","displayName":"App\\Jobs\\BridgeSyncFetchPageJob","job":"Illuminate\\Queue\\CallQueuedHandler@call","data":{"commandName":"App\\Jobs\\BridgeSyncFetchPageJob"}}`)
	if !queue.IsLegacyLaravelPayload(laravel) {
		t.Fatal("expected Laravel payload detected")
	}
	if queue.LegacyLaravelJobName(laravel) != `App\Jobs\BridgeSyncFetchPageJob` {
		t.Fatalf("unexpected name: %s", queue.LegacyLaravelJobName(laravel))
	}

	goJob, err := queue.MarshalPayload(queue.TypeNoop, nil)
	if err != nil {
		t.Fatal(err)
	}
	if queue.IsLegacyLaravelPayload(goJob) {
		t.Fatal("Go payload should not be treated as Laravel")
	}
}
