package apiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("missing auth header")
		}
		if r.Header.Get("X-Domain-Slug") != "example" {
			t.Fatalf("missing domain slug")
		}
		if r.URL.Path != "/api/v1/bridge/stats" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New(Config{
		BaseURL:      srv.URL,
		DomainSlug:   "example",
		ServiceToken: "test-token",
	})
	body, status, err := c.Get(context.Background(), "/api/v1/bridge/stats", nil)
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body=%s", body)
	}
}
