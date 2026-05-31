package bridge

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/quantyralabs/idx-api/internal/config"
	dom "github.com/quantyralabs/idx-api/internal/domain"
	"github.com/quantyralabs/idx-api/internal/mlspoxy/bridge"
)

func TestApplyPropertiesConvenienceBody_setsODataQuery(t *testing.T) {
	app := fiber.New()
	var gotFilter, gotTop string
	app.Post("/properties", func(c *fiber.Ctx) error {
		if err := applyPropertiesConvenienceBody(c); err != nil {
			return err
		}
		args := c.Request().URI().QueryArgs()
		gotFilter = string(args.Peek("$filter"))
		gotTop = string(args.Peek("$top"))
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/properties", strings.NewReader(`{"city":"Largo","limit":2}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	if gotFilter != "City eq 'Largo'" {
		t.Fatalf("$filter = %q", gotFilter)
	}
	if gotTop != "2" {
		t.Fatalf("$top = %q", gotTop)
	}
}

func TestDecodePropertiesCursor(t *testing.T) {
	raw := base64.RawURLEncoding.EncodeToString([]byte(`{"top":10,"skip":20}`))
	top, skip, ok := decodePropertiesCursor(raw)
	if !ok {
		t.Fatal("expected ok")
	}
	if top != 10 || skip != 20 {
		t.Fatalf("top=%d skip=%d", top, skip)
	}
}

func TestApplyPropertiesConvenienceQuery_mergesExistingFilter(t *testing.T) {
	app := fiber.New()
	var gotFilter string
	app.Get("/properties", func(c *fiber.Ctx) error {
		c.Request().URI().QueryArgs().Set("$filter", "StandardStatus eq 'Active'")
		applyPropertiesConvenienceQuery(c)
		gotFilter = string(c.Request().URI().QueryArgs().Peek("$filter"))
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/properties?city=Largo&limit=5", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	want := "(StandardStatus eq 'Active') and (City eq 'Largo')"
	if gotFilter != want {
		t.Fatalf("$filter = %q want %q", gotFilter, want)
	}
}

func TestBridgeProxyMethod_propertiesPostUsesGETWithoutBody(t *testing.T) {
	var gotMethod string
	var gotBody []byte
	var gotFilter, gotTop string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotBody, _ = io.ReadAll(r.Body)
		gotFilter = r.URL.Query().Get("$filter")
		gotTop = r.URL.Query().Get("$top")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer srv.Close()

	cfg := config.Config{
		Bridge: config.BridgeConfig{
			Host:       srv.URL,
			PathPrefix: "api/v2",
			Dataset:    "stellar",
		},
	}
	cli := bridge.NewClient(cfg, dom.FeedDefinition{Provider: "bridge", Dataset: "stellar"})

	app := fiber.New()
	app.Post("/properties", func(c *fiber.Ctx) error {
		if err := applyPropertiesConvenienceBody(c); err != nil {
			return err
		}
		applyPropertiesConvenienceQuery(c)
		status, body, _, err := cli.ProxyMethod(c, cli.ResoURL("Property", "stellar"), http.MethodGet)
		if err != nil {
			return err
		}
		return c.Status(status).Send(body)
	})

	req := httptest.NewRequest(http.MethodPost, "/properties?dataset=stellar", strings.NewReader(`{"city":"Largo","limit":2}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, body)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("upstream method %q want GET", gotMethod)
	}
	if len(gotBody) != 0 {
		t.Fatalf("upstream body len %d want 0", len(gotBody))
	}
	if gotFilter != "City eq 'Largo'" {
		t.Fatalf("$filter = %q", gotFilter)
	}
	if gotTop != "2" {
		t.Fatalf("$top = %q", gotTop)
	}
}
