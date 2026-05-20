package sync

import (
	"net/url"
	"testing"
)

func TestOdataQueryMap(t *testing.T) {
	if odataQueryMap(nil) != nil {
		t.Fatal("expected nil for empty query")
	}
	m := odataQueryMap(url.Values{
		"$top":    {"2000"},
		"$filter": {"StandardStatus eq 'Active'"},
	})
	if m["$top"] != "2000" {
		t.Fatalf("$top = %q", m["$top"])
	}
	if m["$filter"] == "" {
		t.Fatal("expected $filter")
	}
}

func TestReplicaPageMetaFromResult(t *testing.T) {
	meta := replicaPageMetaFromResult(PageResult{
		FetchURL:    "https://api.example/Property",
		UpstreamURL: "https://api.example/Property?$top=10",
		ODataQuery:  map[string]string{"$top": "10"},
	})
	if meta.FetchURL != "https://api.example/Property" {
		t.Fatalf("fetch_url = %q", meta.FetchURL)
	}
	if meta.UpstreamURL == "" {
		t.Fatal("expected upstream_url")
	}
	if meta.ODataQuery["$top"] != "10" {
		t.Fatalf("odata_query = %#v", meta.ODataQuery)
	}
}
