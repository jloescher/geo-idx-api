package sync

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestKickoff_dispatchSparkRequiresToken(t *testing.T) {
	t.Setenv("SPARK_ACCESS_TOKEN", "")
	t.Setenv("SPARK_API_KEY", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.MLS.BeachesEnabled = true
	cfg.Spark.Datasets = []string{"beaches"}

	k := NewKickoff(cfg, nil, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err := k.dispatchSpark(context.Background()); err != nil {
		t.Fatal(err)
	}
}
