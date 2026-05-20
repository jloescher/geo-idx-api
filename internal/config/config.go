// Package config loads environment variables matching .env.example (Laravel parity).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds application configuration.
type Config struct {
	App       AppConfig
	DB        DBConfig
	Queue     QueueConfig
	Idx       IdxURLsConfig
	Bridge    BridgeConfig
	Spark     SparkConfig
	MLS       MLSConfig
	GIS       GISConfig
	Images    ImageCacheConfig
	Auth      AuthConfig
	Coingecko CoingeckoConfig
}

type AppConfig struct {
	Name    string
	Env     string
	Debug   bool
	Port    string
	LogJSON bool
}

type DBConfig struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
	SSLMode  string
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Database, d.SSLMode)
}

type QueueConfig struct {
	Table         string
	PollInterval  time.Duration
	WorkerQueues  []string
	RetryAfter    time.Duration
	NotifyChannel string
}

type IdxURLsConfig struct {
	PlatformURL   string
	APIPublic     string
	ImagesPublic  string
	PlatformHosts []string
	APIHosts      []string
}

type BridgeConfig struct {
	APIKey              string
	Host                string
	Dataset             string
	PathPrefix          string
	ResoRoot            string
	ListingPhotoPath    string
	Timeout             time.Duration
	Datasets            []string
	ListingsCacheTTL    time.Duration
	SyncFetchQueue      string
	SyncPersistQueue    string
	SyncPersistChunk    int
	SyncUpsertChunk     int
	SyncReplicationTop  int
	SyncIncrementalTop  int
	SyncIncludeMedia    bool
	SyncFullProperty    bool
	SyncExpand          string
	SyncMaxChainedFetch int
	ImageRewriteHosts   []string
}

type SparkConfig struct {
	AccessToken         string
	ReplicationHost     string
	ReplicationReso     string
	APIHost             string
	APIVersion          string
	LiveResoRoot        string
	Datasets            []string
	Timeout             time.Duration
	SyncFetchQueue      string
	SyncPersistQueue    string
	SyncPersistChunk    int
	SyncUpsertChunk     int
	SyncReplicationTop  int
	SyncIncrementalTop  int
	SyncExpand          string
	SyncMaxChainedFetch int
	PersistSequential   bool
}

type MLSConfig struct {
	LocalMirrorRollingMonths    int
	ReplicaPageRetentionHours   int
	ListingsCacheRetentionDays  int
	ReplicationFreshnessMinutes int
	StellarEnabled              bool
	StellarPersistChunk         int
	BeachesEnabled              bool
	BeachesPersistChunk         int
	SyncExpand                  string
}

type GISConfig struct {
	EdgeCacheTTL         time.Duration
	OriginMaxDaysPrimary int
	OriginMaxDaysCounty  int
	TeaserMaxFeatures    int
	TeaserCoordDecimals  int
	MaxBboxSpanDeg       float64
	FloridaMLSCodes      []string
	Queue                string
}

type ImageCacheConfig struct {
	Path string
	TTL  time.Duration
}

type AuthConfig struct {
	SessionLifetime time.Duration
	InvitationTTL   time.Duration
	TurnstileSite   string
	TurnstileSecret string
	AdminSeedEmail  string
	AdminSeedPass   string
	AdminSeedName   string
}

type CoingeckoConfig struct {
	APIKey      string
	BaseURL     string
	CacheTTL    time.Duration
	Queue       string
	HTTPTimeout time.Duration
}

// Load reads configuration from environment (optionally .env via godotenv in main).
func Load() (Config, error) {
	port := env("APP_PORT", "8000")
	if port == "" {
		port = "8000"
	}

	pollMs, _ := strconv.Atoi(env("QUEUE_POLL_INTERVAL_MS", "1000"))
	retryAfter, _ := strconv.Atoi(env("DB_QUEUE_RETRY_AFTER", "120"))

	cfg := Config{
		App: AppConfig{
			Name:    env("APP_NAME", "Quantyra IDX API"),
			Env:     env("APP_ENV", "local"),
			Debug:   envBool("APP_DEBUG", true),
			Port:    port,
			LogJSON: env("LOG_FORMAT", "") == "json" || env("APP_ENV", "local") == "production",
		},
		DB: DBConfig{
			Host:     env("DB_HOST", "127.0.0.1"),
			Port:     env("DB_PORT", "5432"),
			Database: env("DB_DATABASE", "idx_api"),
			User:     env("DB_USERNAME", "postgres"),
			Password: env("DB_PASSWORD", ""),
			SSLMode:  defaultDBSSLMode(),
		},
		Queue: QueueConfig{
			Table:         env("DB_QUEUE_TABLE", "jobs"),
			PollInterval:  time.Duration(pollMs) * time.Millisecond,
			WorkerQueues:  splitCSV(env("WORKER_QUEUES", "default,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist")),
			RetryAfter:    time.Duration(retryAfter) * time.Second,
			NotifyChannel: env("QUEUE_NOTIFY_CHANNEL", "idx_jobs_wakeup"),
		},
		Idx: IdxURLsConfig{
			PlatformURL:   firstNonEmpty(env("IDX_PLATFORM_URL", ""), env("APP_URL", "http://localhost:8000")),
			APIPublic:     firstNonEmpty(env("IDX_API_PUBLIC_URL", ""), env("API_URL", ""), env("APP_URL", "http://localhost:8000")),
			ImagesPublic:  firstNonEmpty(env("IDX_IMAGES_PUBLIC_URL", ""), env("IMAGE_URL", "http://localhost:8080")),
			PlatformHosts: splitCSV(env("IDX_PLATFORM_HOSTS", "localhost")),
			APIHosts:      splitCSV(env("IDX_API_HOSTS", "localhost")),
		},
		Bridge: BridgeConfig{
			APIKey:              env("BRIDGE_API_KEY", ""),
			Host:                env("BRIDGE_HOST", "https://api.bridgedataoutput.com"),
			Dataset:             env("BRIDGE_DATASET", "stellar"),
			PathPrefix:          env("BRIDGE_PATH_PREFIX", "api/v2"),
			ResoRoot:            env("BRIDGE_RESO_ROOT", "reso/odata"),
			ListingPhotoPath:    env("BRIDGE_LISTING_PHOTO_PATH", "/api/v2/{dataset}/listings/{listingKey}/photos/{photoId}"),
			Timeout:             envDuration("BRIDGE_TIMEOUT", 30*time.Second),
			Datasets:            splitCSV(env("BRIDGE_DATASETS", "stellar")),
			ListingsCacheTTL:    envDuration("LISTINGS_CACHE_TTL", 900*time.Second),
			SyncFetchQueue:      env("BRIDGE_SYNC_FETCH_QUEUE", "bridge-sync-fetch"),
			SyncPersistQueue:    env("BRIDGE_SYNC_PERSIST_QUEUE", "bridge-sync-persist"),
			SyncPersistChunk:    envInt("BRIDGE_SYNC_PERSIST_JOB_CHUNK", 50),
			SyncUpsertChunk:     envInt("BRIDGE_SYNC_UPSERT_CHUNK", 250),
			SyncReplicationTop:  envInt("BRIDGE_SYNC_REPLICATION_TOP", 2000),
			SyncIncrementalTop:  envInt("BRIDGE_SYNC_INCREMENTAL_TOP", 200),
			SyncIncludeMedia:    envBool("BRIDGE_SYNC_INCLUDE_MEDIA", true),
			SyncFullProperty:    envBool("BRIDGE_SYNC_FULL_PROPERTY", true),
			SyncExpand:          envBridgeSyncExpand(),
			SyncMaxChainedFetch: envInt("BRIDGE_SYNC_MAX_CHAINED_FETCH_PAGES", 0),
			ImageRewriteHosts:   splitCSV(env("BRIDGE_IMAGE_REWRITE_HOSTS", "")),
		},
		Spark: SparkConfig{
			AccessToken:         firstNonEmpty(env("SPARK_ACCESS_TOKEN", ""), env("SPARK_API_KEY", "")),
			ReplicationHost:     env("SPARK_REPLICATION_HOST", "https://replication.sparkapi.com"),
			ReplicationReso:     env("SPARK_REPLICATION_RESO_ROOT", "Reso/OData"),
			APIHost:             env("SPARK_API_HOST", "https://sparkapi.com"),
			APIVersion:          env("SPARK_API_VERSION", "v1"),
			LiveResoRoot:        env("SPARK_LIVE_RESO_ROOT", "Reso/OData"),
			Datasets:            splitCSV(env("SPARK_DATASETS", "beaches")),
			Timeout:             envDuration("SPARK_TIMEOUT", 30*time.Second),
			SyncFetchQueue:      env("SPARK_SYNC_FETCH_QUEUE", "spark-sync-fetch"),
			SyncPersistQueue:    env("SPARK_SYNC_PERSIST_QUEUE", "spark-sync-persist"),
			SyncPersistChunk:    envInt("SPARK_SYNC_PERSIST_JOB_CHUNK", 50),
			SyncUpsertChunk:     envInt("MLS_BEACHES_UPSERT_CHUNK_SIZE", envInt("SPARK_SYNC_UPSERT_CHUNK", 250)),
			SyncReplicationTop:  envInt("SPARK_SYNC_REPLICATION_TOP", 1000),
			SyncIncrementalTop:  envInt("SPARK_SYNC_INCREMENTAL_TOP", 1000),
			SyncExpand:          envSyncExpand(),
			SyncMaxChainedFetch: envInt("SPARK_SYNC_MAX_CHAINED_FETCH_PAGES", 0),
			PersistSequential:   envBool("SPARK_SYNC_PERSIST_SEQUENTIAL", true),
		},
		MLS: MLSConfig{
			LocalMirrorRollingMonths:    envMirrorRollingMonths(),
			ReplicaPageRetentionHours:   envInt("MLS_REPLICA_PAGE_RETENTION_HOURS", 24),
			ListingsCacheRetentionDays:  envInt("MLS_LISTINGS_CACHE_RETENTION_DAYS", 365),
			ReplicationFreshnessMinutes: envInt("MLS_REPLICATION_FRESHNESS_MINUTES", 15),
			StellarEnabled:              envBool("MLS_STELLAR_ENABLED", true),
			StellarPersistChunk:         envInt("MLS_STELLAR_PERSIST_CHUNK_SIZE", 50),
			BeachesEnabled:              envBool("MLS_BEACHES_ENABLED", true),
			BeachesPersistChunk:         envInt("MLS_BEACHES_PERSIST_CHUNK_SIZE", 25),
			SyncExpand:                  envSyncExpand(),
		},
		GIS: GISConfig{
			EdgeCacheTTL:         envDuration("GIS_EDGE_CACHE_TTL", 900*time.Second),
			OriginMaxDaysPrimary: envInt("GIS_ORIGIN_MAX_DAYS_PRIMARY", 90),
			OriginMaxDaysCounty:  envInt("GIS_ORIGIN_MAX_DAYS_COUNTY", 30),
			TeaserMaxFeatures:    envInt("GIS_TEASER_MAX_FEATURES", 40),
			TeaserCoordDecimals:  envInt("GIS_TEASER_COORD_DECIMALS", 4),
			MaxBboxSpanDeg:       envFloat("GIS_MAX_BBOX_SPAN_DEG", 0.35),
			FloridaMLSCodes:      splitCSV(env("GIS_FLORIDA_MLS_CODES", "stellar")),
			Queue:                env("GIS_QUEUE", "default"),
		},
		Images: ImageCacheConfig{
			Path: env("IMAGE_CACHE_PATH", "/var/cache/geoidx/images"),
			TTL:  envDuration("IMAGE_CACHE_TTL", 86400*time.Second),
		},
		Auth: AuthConfig{
			SessionLifetime: envDuration("SESSION_LIFETIME", 120*time.Minute),
			InvitationTTL:   time.Duration(envInt("IDX_INVITATION_TTL_HOURS", 168)) * time.Hour,
			TurnstileSite:   env("CLOUDFLARE_TURNSTILE_SITE_KEY", ""),
			TurnstileSecret: env("CLOUDFLARE_TURNSTILE_SECRET_KEY", ""),
			AdminSeedEmail:  env("ADMIN_SEED_EMAIL", ""),
			AdminSeedPass:   env("ADMIN_SEED_PASSWORD", ""),
			AdminSeedName:   firstNonEmpty(env("ADMIN_SEED_NAME", ""), "Quantyra Admin"),
		},
		Coingecko: CoingeckoConfig{
			APIKey:      env("COINGECKO_API_KEY", ""),
			BaseURL:     env("COINGECKO_BASE_URL", "https://api.coingecko.com/api/v3"),
			CacheTTL:    envDuration("COINGECKO_CACHE_TTL_SECONDS", 1200*time.Second),
			Queue:       env("COINGECKO_QUEUE", "default"),
			HTTPTimeout: envDuration("COINGECKO_HTTP_TIMEOUT", 12*time.Second),
		},
	}

	if len(cfg.Queue.WorkerQueues) == 0 {
		cfg.Queue.WorkerQueues = []string{"default"}
	}

	return cfg, nil
}

// defaultDBSSLMode uses require for non-local DB_HOST (remote staging/production).
func defaultDBSSLMode() string {
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		return v
	}
	host := strings.TrimSpace(env("DB_HOST", "127.0.0.1"))
	switch host {
	case "", "127.0.0.1", "localhost", "postgres", "::1":
		return "disable"
	default:
		return "require"
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// envMirrorRollingMonths: production defaults to all-time (0); staging to 3 months unless MLS_LOCAL_MIRROR_ROLLING_MONTHS is set.
func envMirrorRollingMonths() int {
	if v := os.Getenv("MLS_LOCAL_MIRROR_ROLLING_MONTHS"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	switch env("APP_ENV", "local") {
	case "production":
		return 0
	case "staging":
		return 3
	default:
		return 12
	}
}

func envFloat(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	if secs, err := strconv.Atoi(v); err == nil {
		return time.Duration(secs) * time.Second
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// envSyncExpand is the OData $expand list for Spark replication (MLS_SYNC_EXPAND).
func envSyncExpand() string {
	return firstNonEmpty(
		env("MLS_SYNC_EXPAND", ""),
		env("SPARK_SYNC_EXPAND", ""),
		"Media,Unit,Room,OpenHouse",
	)
}

// envBridgeSyncExpand is the OData $expand list for Bridge (Stellar navigation property names).
func envBridgeSyncExpand() string {
	return firstNonEmpty(
		env("BRIDGE_SYNC_EXPAND", ""),
		env("MLS_SYNC_EXPAND", ""),
		"Media,OpenHouses,Rooms,UnitTypes",
	)
}
