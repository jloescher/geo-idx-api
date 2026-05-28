package bridge

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	dom "github.com/quantyralabs/idx-api/internal/domain"
)

func TestProxyForwardsPOSTBody(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	cfg := config.Config{Bridge: config.BridgeConfig{APIKey: "test-key", Host: srv.URL}}
	cli := NewClient(cfg, dom.FeedDefinition{Provider: "bridge", Dataset: "stellar"})

	app := fiber.New()
	app.Post("/t", func(c *fiber.Ctx) error {
		st, body, _, err := cli.Proxy(c, srv.URL+"/Property")
		if err != nil {
			return err
		}
		if st != 200 {
			t.Fatalf("status %d body %s", st, body)
		}
		return c.SendStatus(200)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/t", strings.NewReader(`{"filter":"City eq 'Tampa'"}`)))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if !strings.Contains(gotBody, "Tampa") {
		t.Fatalf("upstream body %q", gotBody)
	}
}

func TestProxyStripsInternalQueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	cfg := config.Config{Bridge: config.BridgeConfig{APIKey: "test-key", Host: srv.URL}}
	cli := NewClient(cfg, dom.FeedDefinition{Provider: "bridge", Dataset: "stellar"})

	app := fiber.New()
	app.Get("/t", func(c *fiber.Ctx) error {
		st, _, _, err := cli.Proxy(c, srv.URL+"/stellar/listings")
		if err != nil {
			return err
		}
		if st != 200 {
			t.Fatalf("status %d", st)
		}
		return c.SendStatus(200)
	})

	req := httptest.NewRequest(http.MethodGet, "/t?dataset=stellar&domain=example.com&include_pricing=1&%24top=5", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if strings.Contains(gotQuery, "dataset=") {
		t.Fatalf("dataset forwarded: %q", gotQuery)
	}
	if strings.Contains(gotQuery, "domain=") {
		t.Fatalf("domain forwarded: %q", gotQuery)
	}
	if strings.Contains(gotQuery, "include_pricing=") {
		t.Fatalf("include_pricing forwarded: %q", gotQuery)
	}
	if !strings.Contains(gotQuery, "%24top=5") && !strings.Contains(gotQuery, "$top=5") {
		t.Fatalf("expected $top in query, got %q", gotQuery)
	}
}
