package comps

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

func TestSubjectFromLiveClosedSpark(t *testing.T) {
	var gotAuth string
	var gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotFilter = r.URL.Query().Get("$filter")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[{"ListingKey":"K1","ListingId":"MLS-123","StandardStatus":"Closed","Latitude":27.95,"Longitude":-82.46,"BedroomsTotal":3,"BathroomsTotalDecimal":2,"LivingArea":1800}]}`))
	}))
	defer srv.Close()

	e := NewEngine(config.Config{
		Spark: config.SparkConfig{
			AccessToken: "spark-secret",
			APIHost:     srv.URL,
			APIVersion:  "v1",
			LiveResoRoot: "Reso/OData",
			Datasets:    []string{"beaches"},
		},
	}, &repository.DB{})

	sub, err := e.subjectFromLiveClosed(context.Background(), "beaches", "MLS-123", SubjectInput{ListingID: "MLS-123"})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer spark-secret" {
		t.Fatalf("auth %q", gotAuth)
	}
	if !strings.Contains(gotFilter, "StandardStatus eq 'Closed'") || !strings.Contains(gotFilter, "ListingId eq 'MLS-123'") {
		t.Fatalf("filter %q", gotFilter)
	}
	if sub.ListingKey != "K1" || sub.Bedrooms != 3 || sub.Lat == 0 || sub.Lng == 0 {
		t.Fatalf("subject %+v", sub)
	}
}

func TestSubjectFromLiveClosedBridge(t *testing.T) {
	var gotAuth string
	var gotFilter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotFilter = r.URL.Query().Get("$filter")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[{"ListingKey":"BK1","ListingId":"B-MLS-1","StandardStatus":"Closed","Latitude":27.96,"Longitude":-82.47}]}`))
	}))
	defer srv.Close()

	e := NewEngine(config.Config{
		Bridge: config.BridgeConfig{
			APIKey:   "bridge-secret",
			Host:     srv.URL,
			Dataset:  "stellar",
			Datasets: []string{"stellar"},
		},
	}, &repository.DB{})

	sub, err := e.subjectFromLiveClosed(context.Background(), "stellar", "B-MLS-1", SubjectInput{ListingID: "B-MLS-1"})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer bridge-secret" {
		t.Fatalf("auth %q", gotAuth)
	}
	if !strings.Contains(gotFilter, "StandardStatus eq 'Closed'") || !strings.Contains(gotFilter, "ListingId eq 'B-MLS-1'") {
		t.Fatalf("filter %q", gotFilter)
	}
	if sub.ListingKey != "BK1" || sub.Lat == 0 || sub.Lng == 0 {
		t.Fatalf("subject %+v", sub)
	}
}
