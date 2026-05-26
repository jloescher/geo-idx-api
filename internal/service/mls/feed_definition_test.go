package mls

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestFeedDefinitionFromCodeSpark(t *testing.T) {
	cfg := config.Config{
		Bridge: config.BridgeConfig{Dataset: "stellar", Datasets: []string{"stellar"}},
		Spark:  config.SparkConfig{Datasets: []string{"beaches"}},
	}
	f := FeedDefinitionFromCode(cfg, "spark_beaches")
	if f.Provider != "spark" || f.Dataset != "beaches" {
		t.Fatalf("got %+v", f)
	}
}

func TestFeedDefinitionFromCodeBridge(t *testing.T) {
	cfg := config.Config{
		Bridge: config.BridgeConfig{Dataset: "stellar", Datasets: []string{"stellar"}},
		Spark:  config.SparkConfig{Datasets: []string{"beaches"}},
	}
	f := FeedDefinitionFromCode(cfg, "bridge_stellar")
	if f.Provider != "bridge" || f.Dataset != "stellar" {
		t.Fatalf("got %+v", f)
	}
}

func TestDatasetSlugFromFeedCode(t *testing.T) {
	if DatasetSlugFromFeedCode("spark_beaches") != "beaches" {
		t.Fatal("spark beaches slug")
	}
	if DatasetSlugFromFeedCode("bridge_stellar") != "stellar" {
		t.Fatal("bridge stellar slug")
	}
}
