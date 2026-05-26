package cache

import (
	"testing"
)

func TestGzipRoundTrip(t *testing.T) {
	in := []byte(`{"value":[{"ListingKey":"1"}]}`)
	out, err := gzipBytes(in)
	if err != nil {
		t.Fatal(err)
	}
	back, err := gunzip(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(back) != string(in) {
		t.Fatalf("got %q", back)
	}
}

func TestTTLForPartitionLookupSuffix(t *testing.T) {
	if !stringsHasSuffix("a:b:lookup", ":lookup") {
		t.Fatal("suffix check")
	}
}
