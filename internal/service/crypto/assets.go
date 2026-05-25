package crypto

// coingeckoTrackedAssets maps CoinGecko /simple/price ids to mirror asset_key values.
var coingeckoTrackedAssets = []struct {
	coingeckoID string
	assetKey    string
}{
	{coingeckoID: "bitcoin", assetKey: "btc"},
	{coingeckoID: "ethereum", assetKey: "eth"},
	{coingeckoID: "solana", assetKey: "sol"},
	{coingeckoID: "ripple", assetKey: "xrp"},
}

func coingeckoIDs() []string {
	ids := make([]string, len(coingeckoTrackedAssets))
	for i, a := range coingeckoTrackedAssets {
		ids[i] = a.coingeckoID
	}
	return ids
}

func assetKeyForCoingeckoID(id string) (string, bool) {
	for _, a := range coingeckoTrackedAssets {
		if a.coingeckoID == id {
			return a.assetKey, true
		}
	}
	return "", false
}
