package crypto

import "testing"

func TestAssetKeyForCoingeckoID(t *testing.T) {
	tests := map[string]string{
		"bitcoin":  "btc",
		"ethereum": "eth",
		"solana":   "sol",
		"ripple":   "xrp",
	}
	for id, want := range tests {
		got, ok := assetKeyForCoingeckoID(id)
		if !ok || got != want {
			t.Fatalf("%s: got %q ok=%v want %q", id, got, ok, want)
		}
	}
	if _, ok := assetKeyForCoingeckoID("cardano"); ok {
		t.Fatal("unexpected mapping for cardano")
	}
}

func TestCoingeckoIDsIncludesXRP(t *testing.T) {
	ids := coingeckoIDs()
	if len(ids) != 4 {
		t.Fatalf("len(ids)=%d want 4", len(ids))
	}
	found := false
	for _, id := range ids {
		if id == "ripple" {
			found = true
		}
	}
	if !found {
		t.Fatalf("ids %v missing ripple", ids)
	}
}
