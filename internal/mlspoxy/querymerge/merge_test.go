package querymerge

import (
	"net/url"
	"testing"
)

func TestIntoUpstreamOmitsInternalKeys(t *testing.T) {
	dst := url.Values{}
	IntoUpstream(dst, map[string]string{
		"dataset":         "stellar",
		"domain":          "example.com",
		"include_pricing": "1",
		"$top":            "10",
	})
	if dst.Get("dataset") != "" {
		t.Fatalf("dataset forwarded: %v", dst)
	}
	if dst.Get("domain") != "" {
		t.Fatalf("domain forwarded: %v", dst)
	}
	if dst.Get("include_pricing") != "" {
		t.Fatalf("include_pricing forwarded: %v", dst)
	}
	if dst.Get("$top") != "10" {
		t.Fatalf("$top = %q", dst.Get("$top"))
	}
}

func TestIsInternalCaseInsensitive(t *testing.T) {
	if !IsInternal("Dataset") {
		t.Fatal("expected Dataset internal")
	}
}
