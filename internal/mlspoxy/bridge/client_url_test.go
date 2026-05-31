package bridge

import (
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
	dom "github.com/quantyralabs/idx-api/internal/domain"
)

func testBridgeClient(t *testing.T) *Client {
	t.Helper()
	return NewClient(config.Config{
		Bridge: config.BridgeConfig{
			Host:       "https://api.bridgedataoutput.com",
			PathPrefix: "api/v2",
			ResoRoot:   "reso/odata",
		},
	}, dom.FeedDefinition{Provider: "bridge", Dataset: "stellar"})
}

func TestResoBase_prodShape(t *testing.T) {
	cli := testBridgeClient(t)
	got := cli.ResoBase("stellar")
	want := "https://api.bridgedataoutput.com/api/v2/OData/stellar"
	if got != want {
		t.Fatalf("ResoBase = %q want %q", got, want)
	}
}

func TestResoURL_Property(t *testing.T) {
	cli := testBridgeClient(t)
	got := cli.ResoURL("Property", "stellar")
	want := "https://api.bridgedataoutput.com/api/v2/OData/stellar/Property"
	if got != want {
		t.Fatalf("ResoURL = %q want %q", got, want)
	}
}

func TestPubURL_parcels(t *testing.T) {
	cli := testBridgeClient(t)
	got := cli.PubURL("pub/parcels")
	want := "https://api.bridgedataoutput.com/api/v2/pub/parcels"
	if got != want {
		t.Fatalf("PubURL = %q want %q", got, want)
	}
}

func TestLegacyResoURL_differsFromPrimary(t *testing.T) {
	cli := testBridgeClient(t)
	primary := cli.ResoURL("Property", "stellar")
	legacy := cli.LegacyResoURL("Property", "stellar")
	if primary == legacy {
		t.Fatalf("legacy URL should differ from primary: %q", primary)
	}
	if legacy != "https://api.bridgedataoutput.com/api/v2/reso/odata/stellar/Property" {
		t.Fatalf("LegacyResoURL = %q", legacy)
	}
}

func TestWebURL_listings(t *testing.T) {
	cli := testBridgeClient(t)
	got := cli.WebURL("listings", "stellar")
	want := "https://api.bridgedataoutput.com/api/v2/stellar/listings"
	if got != want {
		t.Fatalf("WebURL = %q want %q", got, want)
	}
}
