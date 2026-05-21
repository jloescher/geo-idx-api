package bridge

import (
	"context"
	"io"
	"log/slog"
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
