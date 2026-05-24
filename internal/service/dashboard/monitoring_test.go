package dashboard

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func TestProviderForDataset(t *testing.T) {
	cfg := config.Config{
		Bridge: config.BridgeConfig{Datasets: []string{"stellar"}},
		Spark:  config.SparkConfig{Datasets: []string{"beaches"}},
	}
	resolver := mls.NewResolver(cfg)
	if got := providerForDataset(resolver, "stellar"); got != "bridge" {
		t.Fatalf("stellar provider: %q", got)
	}
	if got := providerForDataset(resolver, "beaches"); got != "spark" {
		t.Fatalf("beaches provider: %q", got)
	}
}
