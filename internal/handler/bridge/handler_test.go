package bridge

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/api/ctxkeys"
	"github.com/quantyralabs/idx-api/internal/config"
	dom "github.com/quantyralabs/idx-api/internal/domain"
	"github.com/quantyralabs/idx-api/internal/mlspoxy"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/images"
	"github.com/quantyralabs/idx-api/internal/service/audit"
)

type stubMLSClient struct {
	calls int
	body  []byte
}

func (s *stubMLSClient) Proxy(_ *fiber.Ctx, _ string) (int, []byte, map[string][]string, error) {
	s.calls++
	if s.body == nil {
		s.body = []byte(`{"value":[]}`)
	}
	return fiber.StatusOK, s.body, nil, nil
}

func (s *stubMLSClient) ProxyUpstream(c *fiber.Ctx, url string) (int, []byte, map[string][]string, error) {
	return s.Proxy(c, url)
}

type stubFactory struct {
	client mlspoxy.ProxyClient
}

func (f *stubFactory) ForRequest(*fiber.Ctx) mlspoxy.ProxyClient {
	return f.client
}

type memProxyCache struct {
	entries map[string][]byte
}

func (m *memProxyCache) key(partition, fingerprint string) string {
	return partition + "\x00" + fingerprint
}

func (m *memProxyCache) Get(_ context.Context, partition, fingerprint string) ([]byte, bool, error) {
	if m.entries == nil {
		return nil, false, nil
	}
	b, ok := m.entries[m.key(partition, fingerprint)]
	return b, ok, nil
}

func (m *memProxyCache) Put(_ context.Context, partition, fingerprint string, body []byte) error {
	if m.entries == nil {
		m.entries = make(map[string][]byte)
	}
	m.entries[m.key(partition, fingerprint)] = append([]byte(nil), body...)
	return nil
}

func testHandler(t *testing.T, cache proxyCacheStore, client *stubMLSClient) *Handler {
	t.Helper()
	cfg := config.Config{
		Bridge: config.BridgeConfig{
			Dataset:          "stellar",
			Host:             "https://api.bridgedataoutput.com",
			PathPrefix:       "api/v2",
			ResoRoot:         "reso/odata",
			ListingsCacheTTL: time.Hour,
		},
		Idx: config.IdxURLsConfig{
			ImagesPublic: "https://idx-images.example.com",
		},
	}
	return &Handler{
		cfg:        cfg,
		factory:    &stubFactory{client: client},
		resolver:   mlspoxy.NewUpstreamResolver(cfg),
		rewriter:   images.NewRewriter(cfg),
		audit:      audit.NewLogger(nil),
		proxyCache: cache,
		pricing:    nil,
		logger:     slog.Default(),
	}
}

func fiberWithFeed(app *fiber.App, h *Handler) {
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(ctxkeys.MLSDomainSlug, "example.com")
		c.Locals(ctxkeys.MLSFeedCode, "bridge_stellar")
		c.Locals(ctxkeys.MLSFeedDef, dom.FeedDefinition{
			Code: "bridge_stellar", Provider: "bridge", Dataset: "stellar",
		})
		return c.Next()
	})
	app.Get("/listings", h.Listings)
	app.Get("/listings/:listingId", h.Listing)
}

func TestPropertiesProxyResoFallback(t *testing.T) {
	cli := &seqMLSClient{statuses: []int{404, 200}}
	cfg := config.Config{
		Bridge: config.BridgeConfig{
			Dataset:          "stellar",
			Host:             "https://api.bridgedataoutput.com",
			PathPrefix:       "api/v2",
			ResoRoot:         "reso/odata",
			ListingsCacheTTL: time.Hour,
		},
	}
	h := &Handler{
		cfg:        cfg,
		factory:    &stubFactory{client: cli},
		resolver:   mlspoxy.NewUpstreamResolver(cfg),
		rewriter:   images.NewRewriter(cfg),
		audit:      audit.NewLogger(nil),
		proxyCache: &memProxyCache{},
		logger:     slog.Default(),
	}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(ctxkeys.MLSDomainSlug, "example.com")
		c.Locals(ctxkeys.MLSFeedCode, "bridge_stellar")
		c.Locals(ctxkeys.MLSFeedDef, dom.FeedDefinition{
			Code: "bridge_stellar", Provider: "bridge", Dataset: "stellar",
		})
		return c.Next()
	})
	app.Get("/properties", h.Properties)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/properties", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if got := resp.Header.Get("X-IDX-Upstream-Leg"); got == "" {
		t.Fatal("missing X-IDX-Upstream-Leg")
	}
	if cli.calls < 2 {
		t.Fatalf("expected fallback retry, calls=%d", cli.calls)
	}
}

type seqMLSClient struct {
	statuses []int
	calls    int
	body     []byte
}

func (s *seqMLSClient) Proxy(_ *fiber.Ctx, _ string) (int, []byte, map[string][]string, error) {
	idx := s.calls
	s.calls++
	if idx >= len(s.statuses) {
		idx = len(s.statuses) - 1
	}
	if s.body == nil {
		s.body = []byte(`{"value":[]}`)
	}
	return s.statuses[idx], s.body, nil, nil
}

func (s *seqMLSClient) ProxyUpstream(c *fiber.Ctx, url string) (int, []byte, map[string][]string, error) {
	return s.Proxy(c, url)
}

func TestListingsProxyCacheMissThenHit(t *testing.T) {
	client := &stubMLSClient{body: []byte(`{"value":[{"ListingKey":"1"}]}`)}
	cache := &memProxyCache{}
	h := testHandler(t, cache, client)

	app := fiber.New()
	fiberWithFeed(app, h)

	req := httptest.NewRequest("GET", "/listings", nil)
	resp1, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp1.StatusCode != fiber.StatusOK {
		t.Fatalf("first status %d", resp1.StatusCode)
	}
	if got := resp1.Header.Get("X-IDX-Cache"); got != "MISS" {
		t.Fatalf("first X-IDX-Cache %q want MISS", got)
	}
	if client.calls != 1 {
		t.Fatalf("upstream calls %d want 1", client.calls)
	}

	resp2, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.StatusCode != fiber.StatusOK {
		t.Fatalf("second status %d", resp2.StatusCode)
	}
	if got := resp2.Header.Get("X-IDX-Cache"); got != "HIT" {
		t.Fatalf("second X-IDX-Cache %q want HIT", got)
	}
	if client.calls != 1 {
		t.Fatalf("upstream calls after cache %d want 1", client.calls)
	}
	_, _ = io.ReadAll(resp2.Body)
}

// TestListingDetail_SmokeODataSanitized verifies the detail route (Listing handler):
//   - Returns HTTP 200 for a valid upstream response.
//   - Strips @odata.* metadata keys via SanitizeUpstreamPropertyJSONWithDataset.
//   - Preserves RESO property fields (ListingKey, ListPrice, StandardStatus).
//   - Rewrites media image URLs through the image proxy.
func TestListingDetail_SmokeODataSanitized(t *testing.T) {
	upstreamBody := []byte(`{
		"@odata.context": "https://api.bridgedataoutput.com/api/v2/reso/odata/$metadata#Property",
		"@odata.etag":    "W/\"abc123\"",
		"ListingKey":     "stellar:TEST001",
		"ListPrice":      425000,
		"StandardStatus": "Active",
		"InternetEntireListingDisplayYN": true,
		"IDXParticipationYN": true,
		"Media": [{"MediaURL": "https://cdn.bridgedataoutput.com/images/test.jpg"}]
	}`)

	client := &stubMLSClient{body: upstreamBody}
	h := testHandler(t, &memProxyCache{}, client)

	app := fiber.New()
	fiberWithFeed(app, h)

	req := httptest.NewRequest("GET", "/listings/stellar:TEST001", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("response not valid JSON: %v\nbody: %s", err, body)
	}

	// OData metadata keys must be stripped.
	for _, forbidden := range []string{"@odata.context", "@odata.etag"} {
		if _, has := m[forbidden]; has {
			t.Errorf("detail response must not include %q", forbidden)
		}
	}

	// Core RESO fields must survive sanitization.
	if m["ListingKey"] != "stellar:TEST001" {
		t.Errorf("ListingKey = %v, want stellar:TEST001", m["ListingKey"])
	}
	if m["ListPrice"] != float64(425000) {
		t.Errorf("ListPrice = %v, want 425000", m["ListPrice"])
	}
	if m["StandardStatus"] != "Active" {
		t.Errorf("StandardStatus = %v, want Active", m["StandardStatus"])
	}

	// Image URLs must be rewritten through the proxy.
	media, ok := m["Media"].([]any)
	if !ok || len(media) == 0 {
		t.Fatalf("Media absent or empty in detail response: %v", m["Media"])
	}
	mediaItem, ok := media[0].(map[string]any)
	if !ok {
		t.Fatalf("Media[0] not an object: %v", media[0])
	}
	mediaURL, _ := mediaItem["MediaURL"].(string)
	if mediaURL == "" {
		t.Error("MediaURL absent from detail response Media[0]")
	}
	// Original CDN URL must be replaced by proxy URL.
	if mediaURL == "https://cdn.bridgedataoutput.com/images/test.jpg" {
		t.Errorf("MediaURL was not rewritten through image proxy: %q", mediaURL)
	}
}

// TestListingDetail_SmokeCacheHitSkipsUpstream verifies that a warm cache returns the cached
// response without calling the upstream, and sets X-IDX-Cache: HIT.
func TestListingDetail_SmokeCacheHitSkipsUpstream(t *testing.T) {
	upstreamBody := []byte(`{"ListingKey":"stellar:CACHED","ListPrice":300000,"StandardStatus":"Active","InternetEntireListingDisplayYN":true,"IDXParticipationYN":true}`)
	client := &stubMLSClient{body: upstreamBody}
	cache := &memProxyCache{}
	h := testHandler(t, cache, client)

	app := fiber.New()
	fiberWithFeed(app, h)

	req1 := httptest.NewRequest("GET", "/listings/stellar:CACHED", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp1.Body)
	if resp1.StatusCode != 200 {
		t.Fatalf("miss status %d", resp1.StatusCode)
	}
	if client.calls != 1 {
		t.Fatalf("expected 1 upstream call on cache miss, got %d", client.calls)
	}

	req2 := httptest.NewRequest("GET", "/listings/stellar:CACHED", nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp2.Body)
	if resp2.StatusCode != 200 {
		t.Fatalf("hit status %d", resp2.StatusCode)
	}
	if got := resp2.Header.Get("X-IDX-Cache"); got != "HIT" {
		t.Errorf("X-IDX-Cache = %q, want HIT", got)
	}
	if client.calls != 1 {
		t.Fatalf("upstream must not be called on cache hit, got %d calls", client.calls)
	}
}
