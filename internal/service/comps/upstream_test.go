package comps

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
	"github.com/quantyralabs/idx-api/internal/service/mls"
)

func TestFetchSoldCompsUsesSparkUpstream(t *testing.T) {
	var auth string
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[{"ListingKey":"B1","StandardStatus":"Closed","ClosePrice":400000,"CloseDate":"2025-01-01","Latitude":27.95,"Longitude":-82.46,"LivingArea":1800}]}`))
	}))
	defer srv.Close()

	cfg := config.Config{
		Spark: config.SparkConfig{
			AccessToken: "spark-secret",
			APIHost:     srv.URL,
			APIVersion:  "v1",
			LiveResoRoot: "Reso/OData",
			Datasets:    []string{"beaches"},
		},
	}
	e := NewEngine(cfg, &repository.DB{})
	feed := mls.FeedDefinition{Code: "spark_beaches", Provider: "spark", Dataset: "beaches"}
	subject := SubjectProfile{Lat: 27.95, Lng: -82.46}
	radius := 5.0
	sold, err := e.fetchSoldComps(context.Background(), feed, subject, ScopeInput{Type: "radius", RadiusMiles: &radius}, FiltersInput{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer spark-secret" {
		t.Fatalf("auth %q", auth)
	}
	if !strings.Contains(path, "Property") {
		t.Fatalf("path %q", path)
	}
	if len(sold) != 1 || sold[0].ClosePrice != 400000 {
		t.Fatalf("sold %+v", sold)
	}
}
