package images

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/quantyralabs/idx-api/internal/config"
)

// Rewriter replaces Bridge/Spark CDN URLs with idx-images public URLs.
// Revenue impact: controlled image delivery protects MLS compliance and enables edge caching.
type Rewriter struct {
	publicBase string
	hosts      map[string]struct{}
}

func NewRewriter(cfg config.Config) *Rewriter {
	hosts := map[string]struct{}{
		"api.bridgedataoutput.com": {},
		"bridgedataoutput.com":     {},
	}
	for _, h := range cfg.Bridge.ImageRewriteHosts {
		hosts[strings.ToLower(strings.TrimSpace(h))] = struct{}{}
	}
	return &Rewriter{
		publicBase: strings.TrimRight(cfg.Idx.ImagesPublic, "/"),
		hosts:      hosts,
	}
}

func (r *Rewriter) RewriteJSON(body []byte, dataset, listingKey string) []byte {
	if len(body) == 0 {
		return body
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return body
	}
	r.walk(v, dataset, listingKey)
	out, err := json.Marshal(v)
	if err != nil {
		return body
	}
	return out
}

func (r *Rewriter) walk(v any, dataset, listingKey string) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if s, ok := val.(string); ok && r.shouldRewrite(s) {
				if photoID := extractPhotoID(s); photoID != "" && listingKey != "" {
					t[k] = r.publicURL(listingKey, photoID)
				}
			} else {
				r.walk(val, dataset, listingKey)
			}
		}
	case []any:
		for _, item := range t {
			r.walk(item, dataset, listingKey)
		}
	}
}

func (r *Rewriter) shouldRewrite(s string) bool {
	lower := strings.ToLower(s)
	if !strings.HasPrefix(lower, "http") {
		return false
	}
	for h := range r.hosts {
		if strings.Contains(lower, h) {
			return true
		}
	}
	return strings.Contains(lower, "bridgedataoutput") || strings.Contains(lower, "sparkplatform")
}

func extractPhotoID(u string) string {
	parts := strings.Split(strings.TrimRight(u, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func (r *Rewriter) publicURL(listingKey, photoID string) string {
	return r.publicBase + "/images/" + listingKey + "/" + photoID
}

// RewriteBytes is a no-op passthrough when body is not JSON.
func RewriteBytes(rw *Rewriter, body []byte, dataset, listingKey string) []byte {
	if bytes.HasPrefix(bytes.TrimSpace(body), []byte("{")) || bytes.HasPrefix(bytes.TrimSpace(body), []byte("[")) {
		return rw.RewriteJSON(body, dataset, listingKey)
	}
	return body
}
