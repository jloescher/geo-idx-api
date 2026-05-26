package sync

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func TestSkipReplicationFetchWhenPageActive_nonReplicationMode(t *testing.T) {
	skip, err := skipReplicationFetchWhenPageActive(
		context.Background(),
		nil,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		"bridge",
		"stellar",
		"incremental",
	)
	if err != nil {
		t.Fatal(err)
	}
	if skip {
		t.Fatal("expected no skip for incremental mode without store")
	}
}
