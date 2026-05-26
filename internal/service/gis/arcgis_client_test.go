package gis

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestArcGISClientDetectsErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":{"code":400,"message":"Invalid query","details":["Unable to perform query. Please check your parameters."]}}`))
	}))
	defer srv.Close()

	client := NewArcGISClient(config.GISConfig{SyncPageSize: 500})
	_, err := client.FetchLayerPage(srv.URL, "1=1", 0, 500)
	if err == nil {
		t.Fatal("expected arcgis error body to fail")
	}
}
