package db

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultProbeInterval    = 15 * time.Second
	defaultPatroniPort      = "8008"
	defaultPatroniTimeout   = 5 * time.Second
	defaultPingTimeout      = 2 * time.Second
	defaultValidateTimeout  = 500 * time.Millisecond
	defaultFailThreshold    = 3
	defaultRecoverThreshold = 2
	defaultMaxConns         = int32(10)
	defaultMinConns         = int32(1)
	defaultMaxConnLifetime  = 30 * time.Minute
)

// ReplicaSnapshot is exported state for health endpoints and observability.
type ReplicaSnapshot struct {
	Replicas        []ReplicaStatus `json:"replicas"`
	Selected        SelectedReplica `json:"selected"`
	FallbackActive  bool            `json:"fallback_active"`
	LastDiscoveryAt *time.Time      `json:"last_discovery_at,omitempty"`
}

// ReplicaStatus describes one Patroni replica and its last probe result.
type ReplicaStatus struct {
	Name      string  `json:"name"`
	Host      string  `json:"host"`
	LatencyMs float64 `json:"latency_ms"`
	Healthy   bool    `json:"healthy"`
}

// SelectedReplica is the host last chosen by GetBestReplica.
type SelectedReplica struct {
	Host   string `json:"host"`
	Source string `json:"source"` // "replica" or "fallback"
}

type patroniMember struct {
	Name  string `json:"name"`
	Role  string `json:"role"`
	State string `json:"state"`
	Host  string `json:"host"`
	Port  int    `json:"port"`
}

type patroniCluster struct {
	Members []patroniMember `json:"members"`
}

type replicaEntry struct {
	host      string
	name      string
	pool      *pgxpool.Pool
	latencyMs atomic.Int64 // stored as micro-ms * 1000 for atomic int64; we use float via Load/Store as int64 bits... simpler: store millis * 1000
	healthy   atomic.Bool
	failures  atomic.Int32
	successes atomic.Int32
}

func (e *replicaEntry) latencyFloat() float64 {
	return float64(e.latencyMs.Load()) / 1000.0
}

func (e *replicaEntry) setLatency(ms float64) {
	e.latencyMs.Store(int64(ms * 1000))
}

// ReplicaSelector discovers Patroni replicas, probes latency, and routes reads
// to the lowest-latency healthy pool with HAProxy RW fallback.
type ReplicaSelector struct {
	patroniHosts     []string
	readOnlyBase     string
	fallbackPool     *pgxpool.Pool
	mu               sync.RWMutex
	replicas         map[string]*replicaEntry
	lastDiscovery    time.Time
	lastSelected     string
	lastSelectedSrc  string
	fallbackActive    atomic.Bool
	lastFallbackWarn  atomic.Int64 // unix nano; rate-limit fallback WARN logs
	lastDiscoveryWarn atomic.Int64
	probeInterval     time.Duration
	failThreshold    int
	recoverThreshold int
	logger           *slog.Logger
	httpClient       *http.Client
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	closed           atomic.Bool
}

// Option configures optional ReplicaSelector behavior (primarily for tests).
type Option func(*ReplicaSelector)

// WithProbeInterval overrides the default 15s probe interval.
func WithProbeInterval(d time.Duration) Option {
	return func(s *ReplicaSelector) {
		if d > 0 {
			s.probeInterval = d
		}
	}
}

// NewReplicaSelector starts background Patroni discovery and latency probing.
// fallbackPool is the shared RW HAProxy pool used when no replica is healthy.
func NewReplicaSelector(
	ctx context.Context,
	patroniHosts []string,
	readOnlyBaseDSN string,
	fallbackPool *pgxpool.Pool,
	logger *slog.Logger,
	opts ...Option,
) (*ReplicaSelector, error) {
	if fallbackPool == nil {
		return nil, fmt.Errorf("fallback pool is required")
	}
	if strings.TrimSpace(readOnlyBaseDSN) == "" {
		return nil, fmt.Errorf("read-only base DSN is required")
	}
	if len(patroniHosts) == 0 {
		return nil, fmt.Errorf("at least one Patroni host is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	probeCtx, cancel := context.WithCancel(ctx)
	s := &ReplicaSelector{
		patroniHosts:     append([]string(nil), patroniHosts...),
		readOnlyBase:     strings.TrimSpace(readOnlyBaseDSN),
		fallbackPool:     fallbackPool,
		replicas:         make(map[string]*replicaEntry),
		probeInterval:    defaultProbeInterval,
		failThreshold:    defaultFailThreshold,
		recoverThreshold: defaultRecoverThreshold,
		logger:           logger,
		httpClient:       &http.Client{Timeout: defaultPatroniTimeout},
		ctx:              probeCtx,
		cancel:           cancel,
	}
	for _, opt := range opts {
		opt(s)
	}

	s.wg.Add(1)
	go s.probeLoop()

	return s, nil
}

// GetBestReplica returns the healthy replica pool with lowest measured latency,
// or the HAProxy RW fallback pool when no replica is healthy.
func (s *ReplicaSelector) GetBestReplica(ctx context.Context) (*pgxpool.Pool, error) {
	if s.closed.Load() {
		return nil, fmt.Errorf("replica selector is closed")
	}

	for attempt := 0; attempt < 2; attempt++ {
		pool, host, source, ok := s.selectBest()
		if !ok {
			break
		}
		if err := s.quickPing(ctx, pool); err != nil {
			s.logger.Warn("selected replica failed validation ping; marking unhealthy", "host", host, "error", err)
			s.markUnhealthy(host)
			continue
		}
		s.fallbackActive.Store(false)
		s.recordSelection(host, source)
		incReplicaSelected(host)
		return pool, nil
	}

	s.fallbackActive.Store(true)
	s.logFallbackOnce()
	s.recordSelection("fallback", "fallback")
	if err := s.quickPing(ctx, s.fallbackPool); err != nil {
		return nil, fmt.Errorf("fallback pool unreachable: %w", err)
	}
	return s.fallbackPool, nil
}

func (s *ReplicaSelector) selectBest() (*pgxpool.Pool, string, string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type candidate struct {
		host string
		pool *pgxpool.Pool
		lat  float64
	}
	var candidates []candidate
	for host, entry := range s.replicas {
		if entry.pool == nil || !entry.healthy.Load() {
			continue
		}
		candidates = append(candidates, candidate{host: host, pool: entry.pool, lat: entry.latencyFloat()})
	}
	if len(candidates) == 0 {
		return nil, "", "", false
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].lat != candidates[j].lat {
			return candidates[i].lat < candidates[j].lat
		}
		return candidates[i].host < candidates[j].host
	})
	return candidates[0].pool, candidates[0].host, "replica", true
}

func (s *ReplicaSelector) recordSelection(host, source string) {
	s.mu.Lock()
	s.lastSelected = host
	s.lastSelectedSrc = source
	s.mu.Unlock()
}

func (s *ReplicaSelector) markUnhealthy(host string) {
	s.mu.RLock()
	entry, ok := s.replicas[host]
	s.mu.RUnlock()
	if !ok {
		return
	}
	entry.healthy.Store(false)
	entry.failures.Store(int32(s.failThreshold))
	entry.successes.Store(0)
}

// Snapshot returns a copy of current replica health for observability endpoints.
func (s *ReplicaSelector) Snapshot() ReplicaSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := ReplicaSnapshot{
		FallbackActive: s.fallbackActive.Load(),
		Selected: SelectedReplica{
			Host:   s.lastSelected,
			Source: s.lastSelectedSrc,
		},
	}
	if !s.lastDiscovery.IsZero() {
		t := s.lastDiscovery
		out.LastDiscoveryAt = &t
	}
	for _, entry := range s.replicas {
		out.Replicas = append(out.Replicas, ReplicaStatus{
			Name:      entry.name,
			Host:      entry.host,
			LatencyMs: entry.latencyFloat(),
			Healthy:   entry.healthy.Load(),
		})
	}
	sort.Slice(out.Replicas, func(i, j int) bool {
		return out.Replicas[i].Host < out.Replicas[j].Host
	})
	return out
}

// Close stops the probe goroutine and closes all per-replica pools.
// The fallback pool is owned by repository.DB and is not closed here.
func (s *ReplicaSelector) Close() {
	if !s.closed.CompareAndSwap(false, true) {
		return
	}
	s.cancel()
	s.wg.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()
	for host, entry := range s.replicas {
		if entry.pool != nil {
			entry.pool.Close()
		}
		deleteReplicaLatency(host)
	}
	s.replicas = make(map[string]*replicaEntry)
}

func (s *ReplicaSelector) probeLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.probeInterval)
	defer ticker.Stop()

	s.runProbeCycle()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.runProbeCycle()
		}
	}
}

func (s *ReplicaSelector) runProbeCycle() {
	members, err := s.discoverReplicas()
	if err != nil {
		incDiscoveryErrors()
		s.logDiscoveryIssueOnce("patroni discovery failed", err)
		return
	}
	if len(members) == 0 {
		s.logDiscoveryIssueOnce("patroni discovery returned no running replicas", nil)
	}

	s.syncReplicaPools(members)
	s.probeAllReplicas()

	s.mu.Lock()
	s.lastDiscovery = time.Now()
	s.mu.Unlock()
}

func (s *ReplicaSelector) discoverReplicas() ([]patroniMember, error) {
	var lastErr error
	for _, host := range s.patroniHosts {
		members, err := s.fetchPatroniCluster(host)
		if err != nil {
			lastErr = err
			s.logger.Debug("patroni cluster fetch failed", "host", host, "error", err)
			continue
		}
		return filterRunningReplicas(members), nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no patroni hosts configured")
	}
	return nil, lastErr
}

func (s *ReplicaSelector) fetchPatroniCluster(host string) ([]patroniMember, error) {
	url := fmt.Sprintf("http://%s:%s/cluster", host, defaultPatroniPort)
	req, err := http.NewRequestWithContext(s.ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("patroni %s returned %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var cluster patroniCluster
	if err := json.NewDecoder(resp.Body).Decode(&cluster); err != nil {
		return nil, fmt.Errorf("decode patroni cluster: %w", err)
	}
	return cluster.Members, nil
}

func patroniReplicaStateHealthy(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "running", "streaming", "synced":
		return true
	default:
		return false
	}
}

func filterRunningReplicas(members []patroniMember) []patroniMember {
	out := make([]patroniMember, 0, len(members))
	for _, m := range members {
		if strings.EqualFold(m.Role, "replica") && patroniReplicaStateHealthy(m.State) && m.Host != "" {
			out = append(out, m)
		}
	}
	return out
}

func (s *ReplicaSelector) syncReplicaPools(members []patroniMember) {
	desired := make(map[string]patroniMember, len(members))
	for _, m := range members {
		desired[m.Host] = m
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for host, entry := range s.replicas {
		if _, ok := desired[host]; !ok {
			if entry.pool != nil {
				entry.pool.Close()
			}
			deleteReplicaLatency(host)
			delete(s.replicas, host)
			s.logger.Info("removed replica pool", "host", host)
		}
	}

	for host, member := range desired {
		if entry, ok := s.replicas[host]; ok {
			entry.name = member.Name
			continue
		}
		pool, err := s.newReplicaPool(host)
		if err != nil {
			incDiscoveryErrors()
			s.logger.Warn("failed to create replica pool", "host", host, "error", err)
			continue
		}
		entry := &replicaEntry{host: host, name: member.Name, pool: pool}
		entry.healthy.Store(true)
		s.replicas[host] = entry
		s.logger.Info("created replica pool", "host", host, "name", member.Name)
	}
}

func (s *ReplicaSelector) newReplicaPool(host string) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("host=%s %s", host, s.readOnlyBase)
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse replica dsn for %s: %w", host, err)
	}
	cfg.MaxConns = defaultMaxConns
	cfg.MinConns = defaultMinConns
	cfg.MaxConnLifetime = defaultMaxConnLifetime

	pool, err := pgxpool.NewWithConfig(s.ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect replica %s: %w", host, err)
	}
	return pool, nil
}

func (s *ReplicaSelector) probeAllReplicas() {
	s.mu.RLock()
	entries := make([]*replicaEntry, 0, len(s.replicas))
	for _, e := range s.replicas {
		entries = append(entries, e)
	}
	s.mu.RUnlock()

	for _, entry := range entries {
		s.probeReplica(entry)
	}
}

func (s *ReplicaSelector) probeReplica(entry *replicaEntry) {
	if entry.pool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(s.ctx, defaultPingTimeout)
	defer cancel()

	start := time.Now()
	var one int
	err := entry.pool.QueryRow(ctx, `SELECT 1`).Scan(&one)
	latencyMs := float64(time.Since(start).Microseconds()) / 1000.0

	if err != nil || one != 1 {
		entry.setLatency(-1)
		setReplicaLatency(entry.host, -1)
		updateCircuitBreaker(entry, false, s.failThreshold, s.recoverThreshold)
		s.logger.Debug("replica probe failed", "host", entry.host, "failures", entry.failures.Load(), "error", err)
		return
	}

	entry.setLatency(latencyMs)
	setReplicaLatency(entry.host, latencyMs)
	updateCircuitBreaker(entry, true, s.failThreshold, s.recoverThreshold)
}

func (s *ReplicaSelector) quickPing(ctx context.Context, pool *pgxpool.Pool) error {
	pingCtx, cancel := context.WithTimeout(ctx, defaultValidateTimeout)
	defer cancel()
	return pool.Ping(pingCtx)
}

// updateCircuitBreaker applies probe result to circuit breaker state (exported for tests).
const replicaWarnInterval = 60 * time.Second

func (s *ReplicaSelector) logFallbackOnce() {
	now := time.Now().UnixNano()
	last := s.lastFallbackWarn.Load()
	if now-last < int64(replicaWarnInterval) {
		return
	}
	if !s.lastFallbackWarn.CompareAndSwap(last, now) {
		return
	}

	replicaCount, healthyCount := s.replicaHealthSummary()
	s.logger.Warn("no healthy replicas; using HAProxy RW fallback for read",
		"replica_count", replicaCount,
		"healthy_count", healthyCount,
		"patroni_endpoints", len(s.patroniHosts),
	)
}

func (s *ReplicaSelector) logDiscoveryIssueOnce(msg string, err error) {
	now := time.Now().UnixNano()
	last := s.lastDiscoveryWarn.Load()
	if now-last < int64(replicaWarnInterval) {
		return
	}
	if !s.lastDiscoveryWarn.CompareAndSwap(last, now) {
		return
	}

	replicaCount, healthyCount := s.replicaHealthSummary()
	args := []any{
		"replica_count", replicaCount,
		"healthy_count", healthyCount,
		"patroni_endpoints", len(s.patroniHosts),
	}
	if err != nil {
		args = append(args, "error", err)
	}
	s.logger.Warn(msg, args...)
}

func (s *ReplicaSelector) replicaHealthSummary() (replicaCount, healthyCount int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	replicaCount = len(s.replicas)
	for _, entry := range s.replicas {
		if entry.healthy.Load() {
			healthyCount++
		}
	}
	return replicaCount, healthyCount
}

func updateCircuitBreaker(entry *replicaEntry, success bool, failThreshold, recoverThreshold int) {
	if success {
		entry.failures.Store(0)
		successes := entry.successes.Add(1)
		if successes >= int32(recoverThreshold) {
			entry.healthy.Store(true)
		}
		return
	}
	entry.successes.Store(0)
	failures := entry.failures.Add(1)
	if failures >= int32(failThreshold) {
		entry.healthy.Store(false)
	}
}
