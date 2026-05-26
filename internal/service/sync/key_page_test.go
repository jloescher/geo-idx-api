package sync

import (
	"encoding/json"
	"testing"
)

func TestListingKeysFromRows(t *testing.T) {
	rows := []json.RawMessage{
		json.RawMessage(`{"ListingKey":"abc","StandardStatus":"Active"}`),
		json.RawMessage(`{"ListingKey":"def"}`),
		json.RawMessage(`{"StandardStatus":"Pending"}`),
		json.RawMessage(`{"ListingKey":"abc"}`),
	}
	got := listingKeysFromRows(rows)
	want := []string{"abc", "def", "abc"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestDedupeListingKeys(t *testing.T) {
	in := []string{"a", "b", "a", "", "c", "b"}
	got := dedupeListingKeys(in)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestBridgeKeyPageFromResult_complete(t *testing.T) {
	next := "https://example/next"
	page := PageResult{
		Rows:                 []json.RawMessage{json.RawMessage(`{"ListingKey":"k1"}`)},
		NextReplicationURL:   &next,
		ReplicationComplete:  false,
	}
	got := bridgeKeyPageFromResult(page)
	if got.Complete {
		t.Fatal("expected incomplete when next link present")
	}
	if len(got.Keys) != 1 || got.Keys[0] != "k1" {
		t.Fatalf("keys = %v", got.Keys)
	}
}

func TestBridgeKeyPageFromResult_fallbackComplete(t *testing.T) {
	page := PageResult{
		Rows:                []json.RawMessage{json.RawMessage(`{"ListingKey":"k1"}`)},
		ReplicationComplete: true,
	}
	got := bridgeKeyPageFromResult(page)
	if !got.Complete {
		t.Fatal("expected complete")
	}
}
