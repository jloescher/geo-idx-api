package sync

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
)

func TestSparkSync_FetchReplicationPageFromFixture(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "spark", "beaches_50_listings.json"))
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-spark-token" {
			t.Errorf("authorization = %q", got)
		}
		if !strings.Contains(r.URL.Path, "/Property") {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	cfg := config.Config{
		Spark: config.SparkConfig{
			AccessToken:        "test-spark-token",
			ReplicationHost:    srv.URL,
			ReplicationReso:    "Reso/OData",
			SyncReplicationTop: 50,
			SyncExpand:         "Media,Unit,Room,OpenHouse",
		},
	}

	syncer := NewSparkSync(config.Config{
		Spark: cfg.Spark,
		MLS:   config.MLSConfig{LocalMirrorRollingMonths: 0},
	}, nil)
	result, err := syncer.FetchReplicationPage(t.Context(), SyncCursor{})
	if err != nil {
		t.Fatal(err)
	}
	if result.HTTPError || result.Forbidden {
		t.Fatalf("unexpected error status=%d forbidden=%v", result.HTTPStatus, result.Forbidden)
	}
	if len(result.Rows) != 50 {
		t.Fatalf("rows = %d, want 50", len(result.Rows))
	}
	if result.NextReplicationURL == nil || *result.NextReplicationURL == "" {
		t.Fatal("expected next replication URL")
	}
	if result.ReplicationComplete {
		t.Fatal("expected more replication pages")
	}
	if result.FetchURL == "" {
		t.Fatal("expected fetch_url")
	}
	if result.UpstreamURL == "" || !strings.Contains(result.UpstreamURL, "filter=") {
		t.Fatalf("expected upstream_url with OData filter query, got %q", result.UpstreamURL)
	}
	if result.ODataQuery["$filter"] == "" {
		t.Fatalf("expected odata_query $filter, got %#v", result.ODataQuery)
	}
}

func TestSparkSync_PropertyCollectionURL(t *testing.T) {
	syncer := NewSparkSync(config.Config{
		Spark: config.SparkConfig{
			ReplicationHost: "https://replication.sparkapi.com",
			ReplicationReso: "Reso/OData",
		},
	}, nil)

	got := syncer.propertyCollectionURL()
	want := "https://replication.sparkapi.com/Reso/OData/Property"
	if got != want {
		t.Fatalf("url = %q, want %q", got, want)
	}
}

func TestSparkSync_FetchIncrementalPageBuildsWindowFilter(t *testing.T) {
	var filter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filter = r.URL.Query().Get("$filter")
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer srv.Close()

	lower := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	upper := time.Date(2024, 7, 2, 0, 0, 0, 0, time.UTC)

	cfg := config.Config{
		Spark: config.SparkConfig{
			AccessToken:        "tok",
			ReplicationHost:    srv.URL,
			ReplicationReso:    "Reso/OData",
			SyncIncrementalTop: 100,
		},
		MLS: config.MLSConfig{LocalMirrorRollingMonths: 0},
	}
	syncer := NewSparkSync(cfg, nil)
	_, err := syncer.FetchIncrementalPage(t.Context(), SyncCursor{
		LastModificationTimestamp: &lower,
		IncrementalWindowEnd:            &upper,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(filter, "ModificationTimestamp gt 2024-07-01T00:00:00Z") {
		t.Fatalf("filter missing lower bound: %q", filter)
	}
	if !strings.Contains(filter, "ModificationTimestamp lt 2024-07-02T00:00:00Z") {
		t.Fatalf("filter missing upper bound: %q", filter)
	}
	if !strings.Contains(filter, "StandardStatus eq 'Active'") {
		t.Fatalf("filter missing status: %q", filter)
	}
}

func TestSparkSync_FetchIncrementalPageRollingMonthsSingleModificationGt(t *testing.T) {
	var filter string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filter = r.URL.Query().Get("$filter")
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer srv.Close()

	lower := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	upper := time.Date(2024, 7, 2, 0, 0, 0, 0, time.UTC)

	cfg := config.Config{
		Spark: config.SparkConfig{
			AccessToken:        "tok",
			ReplicationHost:    srv.URL,
			ReplicationReso:    "Reso/OData",
			SyncIncrementalTop: 100,
		},
		MLS: config.MLSConfig{LocalMirrorRollingMonths: 3},
	}
	syncer := NewSparkSync(cfg, nil)
	_, err := syncer.FetchIncrementalPage(t.Context(), SyncCursor{
		LastModificationTimestamp: &lower,
		IncrementalWindowEnd:      &upper,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(filter, "ModificationTimestamp gt") != 1 {
		t.Fatalf("expected one ModificationTimestamp gt (cursor only), filter=%q", filter)
	}
	if strings.Contains(filter, activePendingReplicationBaseFilter(cfg)) {
		t.Fatalf("incremental must not embed replication rolling base filter: %q", filter)
	}
}

func TestSparkSync_FetchIncrementalPageSkipsEmptyWindow(t *testing.T) {
	now := time.Date(2026, 5, 26, 6, 37, 0, 0, time.UTC)
	syncer := NewSparkSync(config.Config{
		Spark: config.SparkConfig{AccessToken: "tok"},
	}, nil)
	result, err := syncer.FetchIncrementalPage(t.Context(), SyncCursor{
		LastModificationTimestamp: &now,
		IncrementalWindowEnd:      &now,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReplicationComplete || len(result.Rows) != 0 {
		t.Fatalf("expected empty completed incremental, got %#v", result)
	}
}

func TestSparkSync_FetchIncrementalPageRetriesWithoutExpandOn400(t *testing.T) {
	var calls int
	var hadExpand bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		hadExpand = r.URL.Query().Get("$expand") != ""
		if hadExpand {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"code":"400","message":"expand not allowed"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer srv.Close()

	lower := time.Date(2026, 5, 26, 6, 35, 0, 0, time.UTC)
	upper := time.Date(2026, 5, 26, 6, 37, 0, 0, time.UTC)
	syncer := NewSparkSync(config.Config{
		Spark: config.SparkConfig{
			AccessToken:     "tok",
			ReplicationHost: srv.URL,
			ReplicationReso: "Reso/OData",
		},
		MLS: config.MLSConfig{
			LocalMirrorRollingMonths: 0,
			SyncExpand:               "Media,Unit,Room,OpenHouse",
		},
	}, nil)
	result, err := syncer.FetchIncrementalPage(t.Context(), SyncCursor{
		LastModificationTimestamp: &lower,
		IncrementalWindowEnd:      &upper,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2 (expand then no-expand)", calls)
	}
	if result.HTTPError {
		t.Fatalf("expected success on no-expand retry, status=%d msg=%q", result.HTTPStatus, result.ODataError)
	}
}
