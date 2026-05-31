package spark

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
)

func testSparkClient(t *testing.T) *Client {
	t.Helper()
	return NewClient(config.Config{
		Spark: config.SparkConfig{
			APIHost:      "https://sparkapi.com",
			APIVersion:   "v1",
			LiveResoRoot: "Reso/OData",
		},
	}, nil)
}

func TestWebURL_listings(t *testing.T) {
	cli := testSparkClient(t)
	got := cli.WebURL("listings")
	want := "https://sparkapi.com/v1/listings"
	if got != want {
		t.Fatalf("WebURL = %q want %q", got, want)
	}
}

func TestLiveResoURL_Property(t *testing.T) {
	cli := testSparkClient(t)
	got := cli.LiveResoURL("Property", "beaches")
	want := "https://sparkapi.com/v1/Reso/OData/Property"
	if got != want {
		t.Fatalf("LiveResoURL = %q want %q", got, want)
	}
}

func TestResoV3URL_Property(t *testing.T) {
	cli := testSparkClient(t)
	got := cli.ResoV3URL("Property")
	want := "https://sparkapi.com/v1/Version/3/Reso/OData/Property"
	if got != want {
		t.Fatalf("ResoV3URL = %q want %q", got, want)
	}
}
