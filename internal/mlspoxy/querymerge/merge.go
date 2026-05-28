package querymerge

import (
	"net/url"
	"strings"
)

// excludedKeys are IDX routing or response-shaping params; never forward to MLS upstream.
var excludedKeys = map[string]struct{}{
	"dataset":         {},
	"domain":          {},
	"include_pricing": {},
}

// IsInternal reports whether key is an IDX-only query param (case-insensitive).
func IsInternal(key string) bool {
	_, ok := excludedKeys[strings.ToLower(strings.TrimSpace(key))]
	return ok
}

// IntoUpstream copies client query params into dst, omitting internal IDX keys.
func IntoUpstream(dst url.Values, client map[string]string) {
	for k, v := range client {
		if IsInternal(k) {
			continue
		}
		dst.Set(k, v)
	}
}
