package db

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestFilterRunningReplicas(t *testing.T) {
	members := []patroniMember{
		{Name: "re-node-04", Role: "leader", State: "running", Host: "100.115.75.119"},
		{Name: "re-node-01", Role: "replica", State: "running", Host: "100.126.103.51"},
		{Name: "re-node-03", Role: "replica", State: "stopped", Host: "100.114.117.46"},
		{Name: "re-node-02", Role: "replica", State: "running", Host: "100.114.117.46"},
	}
	got := filterRunningReplicas(members)
	if len(got) != 2 {
		t.Fatalf("expected 2 replicas, got %d", len(got))
	}
	if got[0].Host != "100.126.103.51" && got[1].Host != "100.126.103.51" {
		t.Fatalf("expected re-node-01 host in results: %+v", got)
	}
}

func TestUpdateCircuitBreaker(t *testing.T) {
	entry := &replicaEntry{host: "10.0.0.1"}
	entry.healthy.Store(true)

	for i := 0; i < defaultFailThreshold-1; i++ {
		updateCircuitBreaker(entry, false, defaultFailThreshold, defaultRecoverThreshold)
	}
	if !entry.healthy.Load() {
		t.Fatal("expected still healthy before threshold")
	}
	updateCircuitBreaker(entry, false, defaultFailThreshold, defaultRecoverThreshold)
	if entry.healthy.Load() {
		t.Fatal("expected unhealthy after threshold failures")
	}

	for i := 0; i < defaultRecoverThreshold-1; i++ {
		updateCircuitBreaker(entry, true, defaultFailThreshold, defaultRecoverThreshold)
	}
	if entry.healthy.Load() {
		t.Fatal("expected still unhealthy before recover threshold")
	}
	updateCircuitBreaker(entry, true, defaultFailThreshold, defaultRecoverThreshold)
	if !entry.healthy.Load() {
		t.Fatal("expected healthy after recover threshold")
	}
}

func TestSelectBestReplicaLatency(t *testing.T) {
	s := &ReplicaSelector{
		replicas: map[string]*replicaEntry{
			"10.0.0.1": {host: "10.0.0.1", pool: &pgxpool.Pool{}, healthy: atomic.Bool{}},
			"10.0.0.2": {host: "10.0.0.2", pool: &pgxpool.Pool{}, healthy: atomic.Bool{}},
		},
	}
	s.replicas["10.0.0.1"].healthy.Store(true)
	s.replicas["10.0.0.2"].healthy.Store(true)
	s.replicas["10.0.0.1"].setLatency(12.5)
	s.replicas["10.0.0.2"].setLatency(4.2)

	pool, host, source, ok := s.selectBest()
	if !ok {
		t.Fatal("expected candidate")
	}
	if host != "10.0.0.2" {
		t.Fatalf("expected lowest latency host 10.0.0.2, got %s", host)
	}
	if source != "replica" || pool == nil {
		t.Fatalf("unexpected selection: source=%s pool=%v", source, pool)
	}
}

func TestPatroniDiscoveryFromMockServer(t *testing.T) {
	cluster := patroniCluster{
		Members: []patroniMember{
			{Name: "re-node-04", Role: "leader", State: "running", Host: "100.115.75.119"},
			{Name: "re-node-01", Role: "replica", State: "running", Host: "100.126.103.51"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cluster" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(cluster)
	}))
	defer srv.Close()

	host := srv.Listener.Addr().String()
	// strip port for host field — use 127.0.0.1 and custom client pointing to srv
	s := &ReplicaSelector{
		patroniHosts: []string{"127.0.0.1"},
		httpClient:   srv.Client(),
		ctx:          context.Background(),
	}
	// Override fetch to use test server URL
	origPort := defaultPatroniPort
	_ = host
	_ = origPort

	members, err := s.fetchPatroniClusterFromURL(srv.URL + "/cluster")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	replicas := filterRunningReplicas(members)
	if len(replicas) != 1 || replicas[0].Host != "100.126.103.51" {
		t.Fatalf("unexpected replicas: %+v", replicas)
	}
}

func (s *ReplicaSelector) fetchPatroniClusterFromURL(url string) ([]patroniMember, error) {
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var cluster patroniCluster
	if err := json.NewDecoder(resp.Body).Decode(&cluster); err != nil {
		return nil, err
	}
	return cluster.Members, nil
}

func TestCloseStopsProbeLoop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fallback := &pgxpool.Pool{}
	s := &ReplicaSelector{
		patroniHosts:     []string{"127.0.0.1"},
		readOnlyBase:     "user=u password=p dbname=d port=5432 sslmode=disable",
		fallbackPool:     fallback,
		replicas:         make(map[string]*replicaEntry),
		probeInterval:    20 * time.Millisecond,
		failThreshold:    defaultFailThreshold,
		recoverThreshold: defaultRecoverThreshold,
		logger:           slog.Default(),
		httpClient:       &http.Client{Timeout: 50 * time.Millisecond},
		ctx:              ctx,
		cancel:           cancel,
	}
	s.wg.Add(1)
	go s.probeLoop()

	time.Sleep(30 * time.Millisecond)
	s.Close()
	if !s.closed.Load() {
		t.Fatal("expected closed flag")
	}
}

func TestReplicaEntryLatencyFloat(t *testing.T) {
	e := &replicaEntry{}
	e.setLatency(4.256)
	if got := e.latencyFloat(); got < 4.255 || got > 4.257 {
		t.Fatalf("unexpected latency float: %v", got)
	}
}
