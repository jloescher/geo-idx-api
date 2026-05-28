package openapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestOpenAPIRoutes(t *testing.T) {
	if len(specJSON) == 0 {
		t.Fatal("embedded openapi spec is empty; run make openapi-sync")
	}
	var doc struct {
		OpenAPI string `json:"openapi"`
		Paths   map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(specJSON, &doc); err != nil {
		t.Fatalf("spec JSON: %v", err)
	}
	if doc.OpenAPI == "" {
		t.Fatal("missing openapi version")
	}
	if _, ok := doc.Paths["/api/v1/gis/autocomplete/cities"]; !ok {
		t.Fatal("spec missing autocomplete cities path")
	}

	app := fiber.New()
	Register(app)

	t.Run("openapi.json", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d", resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Fatalf("content-type %q", ct)
		}
		body, _ := io.ReadAll(resp.Body)
		if len(body) == 0 {
			t.Fatal("empty body")
		}
	})

	t.Run("swagger", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/swagger", nil))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "swagger-ui") {
			t.Fatal("expected swagger-ui markup")
		}
	})
}
