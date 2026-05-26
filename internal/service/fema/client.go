package fema

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
)

// PointAttributes holds NFHL Layer 28 fields from a point intersect query.
type PointAttributes struct {
	FLDZone   *string
	SFHA_TF   *string
	DFIRM_ID  *string
	StaticBFE *float64
	Depth     *float64
	Raw       json.RawMessage
}

// Client queries FEMA NFHL MapServer (Layer 28) for flood zone at a lat/lng.
type Client struct {
	http       *http.Client
	baseURL    string
	layerID    int
	userAgent  string
	limiter    *rateLimiter
	breaker    *circuitBreaker
	maxRetries int
}

// NewClient builds an NFHL point-query client from FEMAConfig.
func NewClient(cfg config.FEMAConfig) *Client {
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	threshold := cfg.CircuitFailThreshold
	if threshold <= 0 {
		threshold = 5
	}
	return &Client{
		http: &http.Client{
			Timeout: timeout,
		},
		baseURL:    strings.TrimRight(cfg.NFHLBaseURL, "/"),
		layerID:    cfg.NFHLLayerID,
		userAgent:  cfg.UserAgent,
		limiter:    newRateLimiter(cfg.MaxRequestsPerSecond),
		breaker:    newCircuitBreaker(threshold),
		maxRetries: 4,
	}
}

// QueryPoint returns attributes for the first intersecting flood zone feature, or nil if none.
func (c *Client) QueryPoint(ctx context.Context, lat, lng float64) (*PointAttributes, error) {
	if err := c.breaker.allow(); err != nil {
		nfhlErrorsTotal.WithLabelValues("circuit_open").Inc()
		return nil, err
	}
	if err := c.limiter.wait(ctx); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s/%d/query", c.baseURL, c.layerID)
	q := url.Values{}
	q.Set("geometry", fmt.Sprintf("%f,%f", lng, lat))
	q.Set("geometryType", "esriGeometryPoint")
	q.Set("inSR", "4326")
	q.Set("spatialRel", "esriSpatialRelIntersects")
	q.Set("outFields", "FLD_ZONE,SFHA_TF,DFIRM_ID,STATIC_BFE,DEPTH")
	q.Set("returnGeometry", "false")
	q.Set("f", "json")

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := backoff(ctx, attempt); err != nil {
				return nil, err
			}
		}
		attrs, status, err := c.doQuery(ctx, u+"?"+q.Encode())
		if err == nil {
			c.breaker.recordSuccess()
			nfhlRequestsTotal.WithLabelValues(status).Inc()
			return attrs, nil
		}
		lastErr = err
		if !retryable(err) {
			c.breaker.recordFailure()
			nfhlErrorsTotal.WithLabelValues(errorReason(err)).Inc()
			nfhlRequestsTotal.WithLabelValues("error").Inc()
			return nil, err
		}
	}
	c.breaker.recordFailure()
	nfhlErrorsTotal.WithLabelValues("retry_exhausted").Inc()
	nfhlRequestsTotal.WithLabelValues("error").Inc()
	return nil, lastErr
}

func (c *Client) doQuery(ctx context.Context, rawURL string) (*PointAttributes, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", &retryableError{err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, "", &retryableError{err: fmt.Errorf("fema nfhl http %d", resp.StatusCode)}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("fema nfhl http %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var parsed arcgisQueryResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, "", fmt.Errorf("fema nfhl json: %w", err)
	}
	if parsed.Error != nil {
		if parsed.Error.Code == 429 || parsed.Error.Code >= 500 {
			return nil, "", &retryableError{err: fmt.Errorf("fema nfhl arcgis error %d: %s", parsed.Error.Code, parsed.Error.Message)}
		}
		return nil, "", fmt.Errorf("fema nfhl arcgis error %d: %s", parsed.Error.Code, parsed.Error.Message)
	}

	if len(parsed.Features) == 0 {
		return nil, "miss", nil
	}
	attrs := parsed.Features[0].Attributes
	if attrs == nil {
		return nil, "miss", nil
	}
	raw, _ := json.Marshal(attrs)
	out := &PointAttributes{Raw: raw}
	if v, ok := attrs["FLD_ZONE"]; ok {
		s := stringifyAttr(v)
		if s != "" {
			out.FLDZone = &s
		}
	}
	if v, ok := attrs["SFHA_TF"]; ok {
		s := stringifyAttr(v)
		if s != "" {
			out.SFHA_TF = &s
		}
	}
	if v, ok := attrs["DFIRM_ID"]; ok {
		s := stringifyAttr(v)
		if s != "" {
			out.DFIRM_ID = &s
		}
	}
	if v, ok := attrs["STATIC_BFE"]; ok {
		if f, ok := toFloat64(v); ok {
			out.StaticBFE = &f
		}
	}
	if v, ok := attrs["DEPTH"]; ok {
		if f, ok := toFloat64(v); ok {
			out.Depth = &f
		}
	}
	return out, "hit", nil
}

type arcgisQueryResponse struct {
	Features []struct {
		Attributes map[string]any `json:"attributes"`
	} `json:"features"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type retryableError struct {
	err error
}

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

func retryable(err error) bool {
	var re *retryableError
	return errors.As(err, &re)
}

func errorReason(err error) string {
	if err != nil && strings.Contains(err.Error(), "circuit") {
		return "circuit_open"
	}
	return "request_failed"
}

func backoff(ctx context.Context, attempt int) error {
	d := time.Duration(1<<attempt) * 500 * time.Millisecond
	if d > 8*time.Second {
		d = 8 * time.Second
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func stringifyAttr(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func toFloat64(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

type rateLimiter struct {
	minInterval time.Duration
	mu          sync.Mutex
	last        time.Time
}

func newRateLimiter(maxPerSecond int) *rateLimiter {
	if maxPerSecond <= 0 {
		return nil
	}
	return &rateLimiter{minInterval: time.Second / time.Duration(maxPerSecond)}
}

func (r *rateLimiter) wait(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.last.IsZero() {
		wait := r.minInterval - time.Since(r.last)
		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	r.last = time.Now()
	return nil
}

type circuitBreaker struct {
	threshold int
	mu        sync.Mutex
	failures  int
	open      bool
	openUntil time.Time
}

func newCircuitBreaker(threshold int) *circuitBreaker {
	return &circuitBreaker{threshold: threshold}
}

func (b *circuitBreaker) allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.open {
		return nil
	}
	if time.Now().After(b.openUntil) {
		b.open = false
		b.failures = 0
		circuitBreakerOpen.Set(0)
		return nil
	}
	return fmt.Errorf("fema nfhl circuit breaker open")
}

func (b *circuitBreaker) recordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	if b.open {
		b.open = false
		circuitBreakerOpen.Set(0)
	}
}

func (b *circuitBreaker) recordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.threshold {
		b.open = true
		b.openUntil = time.Now().Add(30 * time.Second)
		circuitBreakerOpen.Set(1)
	}
}
