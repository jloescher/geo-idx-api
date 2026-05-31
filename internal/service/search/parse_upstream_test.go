package search

import (
	"encoding/json"
	"testing"
)

func TestParseWebListingsBody_bridgeBundle(t *testing.T) {
	body := []byte(`{"bundle":[{"ListingKey":"a"}],"total":1}`)
	got, ok := parseWebListingsBody(body)
	if !ok || len(got) != 1 {
		t.Fatalf("parseWebListingsBody = %v ok=%v", got, ok)
	}
}

func TestParseWebListingsBody_sparkStandardFields(t *testing.T) {
	body := []byte(`{"D":{"Success":true,"Results":[{"StandardFields":{"ListingKey":"b"}}]}}`)
	got, ok := parseWebListingsBody(body)
	if !ok || len(got) != 1 {
		t.Fatalf("parseWebListingsBody = %v ok=%v", got, ok)
	}
	var m map[string]any
	if err := json.Unmarshal(got[0], &m); err != nil || m["ListingKey"] != "b" {
		t.Fatalf("result %s", got[0])
	}
}

func TestParseSearchBodyFromUpstream_web(t *testing.T) {
	body := []byte(`{"bundle":[{"ListingKey":"x","InternetEntireListingDisplayYN":true}]}`)
	res, err := parseSearchBodyFromUpstream(body, "stellar", "web")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Results) != 1 {
		t.Fatalf("len %d", len(res.Results))
	}
}
