package fema

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestClientQueryPoint_hit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Fatal("missing User-Agent")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"features": [{
				"attributes": {
					"FLD_ZONE": "AE",
					"SFHA_TF": "T",
					"DFIRM_ID": "12057C"
				}
			}]
		}`))
	}))
	defer srv.Close()

	cfg := config.FEMAConfig{
		NFHLBaseURL:          srv.URL,
		NFHLLayerID:          28,
		MaxRequestsPerSecond: 100,
		UserAgent:            "test-agent",
	}
	c := NewClient(cfg)
	attrs, err := c.QueryPoint(context.Background(), 27.95, -82.45)
	if err != nil {
		t.Fatal(err)
	}
	if attrs == nil || attrs.FLDZone == nil || *attrs.FLDZone != "AE" {
		t.Fatalf("FLD_ZONE: %+v", attrs)
	}
	if attrs.SFHA_TF == nil || *attrs.SFHA_TF != "T" {
		t.Fatalf("SFHA_TF: %+v", attrs)
	}
}

func TestClientQueryPoint_miss(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"features":[]}`))
	}))
	defer srv.Close()

	c := NewClient(config.FEMAConfig{NFHLBaseURL: srv.URL, NFHLLayerID: 28, MaxRequestsPerSecond: 100})
	attrs, err := c.QueryPoint(context.Background(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if attrs != nil {
		t.Fatalf("expected nil attrs on miss, got %+v", attrs)
	}
}

func TestClientQueryPoint_arcgisError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":{"code":500,"message":"server error"}}`))
	}))
	defer srv.Close()

	c := NewClient(config.FEMAConfig{
		NFHLBaseURL:          srv.URL,
		NFHLLayerID:          28,
		MaxRequestsPerSecond: 100,
		CircuitFailThreshold: 10,
	})
	_, err := c.QueryPoint(context.Background(), 1, 1)
	if err == nil {
		t.Fatal("expected error")
	}
}
