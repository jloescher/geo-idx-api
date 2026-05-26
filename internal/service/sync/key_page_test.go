package sync

import (
	"encoding/json"
	"testing"
)

func TestListingKeysFromRows(t *testing.T) {
	rows := []json.RawMessage{
		json.RawMessage(`{"ListingKey":"a"}`),
		json.RawMessage(`{"ListingKey":""}`),
		json.RawMessage(`{"ListingKey":"b"}`),
		json.RawMessage(`invalid`),
	}
	got := listingKeysFromRows(rows)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("keys = %v", got)
	}
}

func TestDedupeListingKeys(t *testing.T) {
	got := dedupeListingKeys([]string{"a", "b", "a", "", "b"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("deduped = %v", got)
	}
}

func TestBridgeKeyPageFromResult_ForbiddenNotComplete(t *testing.T) {
	page := PageResult{Forbidden: true, HTTPStatus: 403, ReplicationComplete: true}
	out := bridgeKeyPageFromResult(page)
	if out.Complete {
		t.Fatal("forbidden page must not mark complete")
	}
	if !out.Forbidden {
		t.Fatal("expected Forbidden on result")
	}
}

func TestSparkKeyPageFromResult_ForbiddenNotComplete(t *testing.T) {
	page := PageResult{Forbidden: true, HTTPStatus: 403, ReplicationComplete: true}
	out := sparkKeyPageFromResult(page)
	if out.Complete {
		t.Fatal("forbidden page must not mark complete")
	}
	if !out.Forbidden {
		t.Fatal("expected Forbidden on result")
	}
}
