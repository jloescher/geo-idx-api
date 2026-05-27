package comps

import (
	"encoding/json"
	"testing"
)

func TestFilterCompRecordsForPublicHomeValue(t *testing.T) {
	comps := []CompRecord{
		{ListingKey: "a", Property: json.RawMessage(`{"InternetEntireListingDisplayYN":false}`)},
		{ListingKey: "b", Property: json.RawMessage(`{"InternetEntireListingDisplayYN":true}`)},
	}
	out := FilterCompRecordsForPublicHomeValue(comps, "beaches")
	if len(out) != 1 || out[0].ListingKey != "b" {
		t.Fatalf("got %+v", out)
	}
}
