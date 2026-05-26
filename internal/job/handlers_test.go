package job_test

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/queue"
)

func TestGISZipSyncJobTypeRegistered(t *testing.T) {
	if queue.TypeGISZipSync != "gis.zip_sync" {
		t.Fatalf("unexpected zip sync type: %s", queue.TypeGISZipSync)
	}
}
