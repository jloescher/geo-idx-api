package cache

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"time"

	"github.com/quantyralabs/idx-api/internal/config"
	"github.com/quantyralabs/idx-api/internal/repository"
)

// ProxyCache stores gzip upstream JSON in mls_search_cache (on-demand, per request fingerprint).
// Revenue impact: repeat identical live proxy queries avoid Bridge/Spark round-trips.
type ProxyCache struct {
	cfg config.Config
	db  *repository.DB
}

func NewProxyCache(cfg config.Config, db *repository.DB) *ProxyCache {
	return &ProxyCache{cfg: cfg, db: db}
}

func (p *ProxyCache) TTLForPartition(partition string) time.Duration {
	if stringsHasSuffix(partition, ":lookup") {
		return p.cfg.Bridge.LookupCacheTTL
	}
	return p.cfg.Bridge.ListingsCacheTTL
}

func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// Get returns decompressed body when a fresh row exists.
func (p *ProxyCache) Get(ctx context.Context, partition, fingerprint string) ([]byte, bool, error) {
	ttl := p.TTLForPartition(partition)
	var compressed []byte
	var lastUpdated time.Time
	err := p.db.Pool.QueryRow(ctx, `
		SELECT compressed_data, last_updated FROM mls_search_cache
		WHERE partition_key = $1 AND fingerprint = $2
	`, partition, fingerprint).Scan(&compressed, &lastUpdated)
	if err != nil {
		return nil, false, nil
	}
	if time.Since(lastUpdated) > ttl {
		return nil, false, nil
	}
	body, err := gunzip(compressed)
	if err != nil {
		return nil, false, err
	}
	return body, true, nil
}

// Put stores gzip-compressed upstream JSON (after image rewrite).
func (p *ProxyCache) Put(ctx context.Context, partition, fingerprint string, body []byte) error {
	compressed, err := gzipBytes(body)
	if err != nil {
		return err
	}
	_, err = p.db.Pool.Exec(ctx, `
		INSERT INTO mls_search_cache (partition_key, fingerprint, compressed_data, last_updated)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (partition_key, fingerprint) DO UPDATE
		SET compressed_data = EXCLUDED.compressed_data, last_updated = NOW()
	`, partition, fingerprint, compressed)
	return err
}

// PurgeExpired removes rows older than MLS proxy cache retention.
func (p *ProxyCache) PurgeExpired(ctx context.Context) (int64, error) {
	days := p.cfg.MLS.ProxyCacheRetentionDays
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	tag, err := p.db.Pool.Exec(ctx, `DELETE FROM mls_search_cache WHERE last_updated < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func gzipBytes(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(b); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gunzip(b []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}
