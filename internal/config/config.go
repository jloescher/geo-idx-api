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
	Geocode   GeocodeConfig
	Coingecko CoingeckoConfig
	FEMA      FEMAConfig
	Comps     CompsConfig
	Scheduler SchedulerConfig
}

// CompsConfig controls comparables engine behavior.
type CompsConfig struct {
	ClosedCacheDays    int
	ClosedCacheMinHits int
}

// FEMAConfig controls NFHL Layer 28 flood zone enrichment for the listings mirror.
type FEMAConfig struct {
	NFHLBaseURL          string
	NFHLLayerID          int
	EnrichBatchSize      int
	FloodStaleDays       int
	MaxRequestsPerSecond int
	HTTPTimeout          time.Duration
	UserAgent            string
	EnrichQueue          string
	CircuitFailThreshold int
}

// SchedulerConfig controls cluster-wide cron leadership (PostgreSQL advisory lock).
type SchedulerConfig struct {
	LeaderLockKey       int64
	StandbyPollInterval time.Duration
}

type AppConfig struct {
	Name    string
	Env     string
	Debug   bool
	Port    string
	LogJSON bool
}

type DBConfig struct {
	Host            string
	Port            string
	Database        string
	User            string
	Password        string
	SSLMode         string
	RWDSN           string   // DB_RW_DSN — HAProxy write port
	ReadOnlyBaseDSN string   // DB_READONLY_BASE_DSN — key=value without host
	PatroniHosts    []string // PATRONI_ENDPOINTS — comma-separated Tailscale IPs
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Database, d.SSLMode)
}

// RWDSNOrDefault returns DB_RW_DSN when set, otherwise the legacy DSN() from DB_* fields.
func (d DBConfig) RWDSNOrDefault() string {
	if strings.TrimSpace(d.RWDSN) != "" {
		return strings.TrimSpace(d.RWDSN)
	}
	return d.DSN()
}

type QueueConfig struct {
	Table               string
	PollInterval        time.Duration
	WorkerQueues        []string
	RetryAfter          time.Duration
	ReservationTimeout  time.Duration
	NotifyChannel       string
}

type IdxURLsConfig struct {
	PlatformURL   string
	APIPublic     string
	ImagesPublic  string
	PlatformHosts []string
	APIHosts      []string
}

type BridgeConfig struct {
	APIKey                         string
	Host                           string
	Dataset                        string
	PathPrefix                     string
	ResoRoot                       string
	ListingPhotoPath               string
	Timeout                        time.Duration
	Datasets                       []string
	ListingsCacheTTL               time.Duration
	LookupCacheTTL                 time.Duration
	SyncFetchQueue                 string
	SyncPersistQueue               string
	SyncPersistChunk               int
	SyncUpsertChunk                int
	SyncReplicationTop             int
	SyncIncrementalTop             int
	SyncIncludeMedia               bool
	SyncFullProperty               bool
	SyncNavHydrateAfterReplication bool
	SyncExpand                     string
	SyncMaxChainedFetch            int
	SyncMaxRequestsPerSecond       int
	SyncMaxHTTPRetries             int
	ImageRewriteHosts              []string
}

type SparkConfig struct {
	AccessToken                string
	ReplicationHost            string
	ReplicationReso            string
	APIHost                    string
	APIVersion                 string
	LiveResoRoot               string
	Datasets                   []string
	Timeout                    time.Duration
	SyncFetchQueue             string
	SyncPersistQueue           string
	SyncPersistChunk           int
	SyncUpsertChunk            int
	SyncReplicationTop         int
	SyncIncrementalTop         int
	SyncExpand                 string
	SyncMaxChainedFetch        int
	SyncMaxRequestsPerSecond   int
	SyncMaxRequestsPer5Min     int
	SyncMaxHTTPRetries         int
}

type MLSConfig struct {
	LocalMirrorRollingMonths       int
	ReplicaPageRetentionHours      int
	ReplicaPageFailedRetentionDays int
	ProxyCacheRetentionDays        int
	ReplicationFreshnessMinutes    int
	StellarEnabled                 bool
	StellarPersistChunk            int
	BeachesEnabled                 bool
	BeachesPersistChunk            int
	SyncExpand                     string
	SyncReplicationExpand          string
	SyncKickoffQueue               string
	RateLimitRetrySeconds          int
	TimeoutRetrySeconds            int
	RateLimitMaxAttempts           int
	ReplicationResumeStallMinutes  int
	ReplicationResumeCron          string
	MirrorKeyReconcileRetryMinutes int
}

type GISConfig struct {
	EdgeCacheTTL         time.Duration
	OriginMaxDaysPrimary int
	OriginMaxDaysCounty  int
	TeaserMaxFeatures    int
	TeaserCoordDecimals  int
	MaxBboxSpanDeg       float64
	MaxFeatures          int
	SyncPageSize            int
	SyncUpsertChunk         int
	HTTPTimeout             time.Duration
	SyncQueue               string
	SyncPinellasEnterprise  bool
	FloridaMLSCodes         []string
	Queue                   string
	BoundaryStaleDays       int
	ImportPath              string
	ImportMaxBytes          int64
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

type GeocodeConfig struct {
	GoogleMapsAPIKey     string
	HTTPTimeout          time.Duration
	EnrichQueue          string
	EnrichBatchSize      int
	MaxRequestsPerSecond int
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
	reservationTimeout, _ := strconv.Atoi(env("DB_QUEUE_RESERVATION_TIMEOUT", "3600"))

	cfg := Config{
		App: AppConfig{
			Name:    env("APP_NAME", "Quantyra IDX API"),
			Env:     env("APP_ENV", "local"),
			Debug:   envBool("APP_DEBUG", true),
			Port:    port,
			LogJSON: env("LOG_FORMAT", "") == "json" || env("APP_ENV", "local") == "production",
		},
		DB: DBConfig{
			Host:            env("DB_HOST", "127.0.0.1"),
			Port:            env("DB_PORT", "5432"),
			Database:        env("DB_DATABASE", "idx_api"),
			User:            env("DB_USERNAME", "postgres"),
			Password:        env("DB_PASSWORD", ""),
			SSLMode:         defaultDBSSLMode(),
			RWDSN:           env("DB_RW_DSN", ""),
			ReadOnlyBaseDSN: env("DB_READONLY_BASE_DSN", ""),
			PatroniHosts:    envPatroniHosts(),
		},
		Queue: QueueConfig{
			Table:              env("DB_QUEUE_TABLE", "jobs"),
			PollInterval:       time.Duration(pollMs) * time.Millisecond,
			WorkerQueues:       splitCSV(env("WORKER_QUEUES", "default,sync-kickoff,bridge-sync-fetch,bridge-sync-persist,spark-sync-fetch,spark-sync-persist")),
			RetryAfter:         time.Duration(retryAfter) * time.Second,
			ReservationTimeout: time.Duration(reservationTimeout) * time.Second,
			NotifyChannel:      env("QUEUE_NOTIFY_CHANNEL", "idx_jobs_wakeup"),
		},
		Idx: IdxURLsConfig{
			PlatformURL:   firstNonEmpty(env("IDX_PLATFORM_URL", ""), env("APP_URL", "http://localhost:8000")),
			APIPublic:     firstNonEmpty(env("IDX_API_PUBLIC_URL", ""), env("API_URL", ""), env("APP_URL", "http://localhost:8000")),
			ImagesPublic:  firstNonEmpty(env("IDX_IMAGES_PUBLIC_URL", ""), env("IMAGE_URL", "http://localhost:8080")),
			PlatformHosts: splitCSV(env("IDX_PLATFORM_HOSTS", "localhost")),
			APIHosts:      splitCSV(env("IDX_API_HOSTS", "localhost")),
		},
		Bridge: BridgeConfig{
			APIKey:                         env("BRIDGE_API_KEY", ""),
			Host:                           env("BRIDGE_HOST", "https://api.bridgedataoutput.com"),
			Dataset:                        env("BRIDGE_DATASET", "stellar"),
			PathPrefix:                     env("BRIDGE_PATH_PREFIX", "api/v2"),
			ResoRoot:                       env("BRIDGE_RESO_ROOT", "reso/odata"),
			ListingPhotoPath:               env("BRIDGE_LISTING_PHOTO_PATH", "/api/v2/{dataset}/listings/{listingKey}/photos/{photoId}"),
			Timeout:                        envDuration("BRIDGE_TIMEOUT", 30*time.Second),
			Datasets:                       splitCSV(env("BRIDGE_DATASETS", "stellar")),
			ListingsCacheTTL:               envDuration("LISTINGS_CACHE_TTL", 900*time.Second),
			LookupCacheTTL:                 envDuration("MLS_LOOKUP_CACHE_TTL", 720*time.Hour),
			SyncFetchQueue:                 env("BRIDGE_SYNC_FETCH_QUEUE", "bridge-sync-fetch"),
			SyncPersistQueue:               env("BRIDGE_SYNC_PERSIST_QUEUE", "bridge-sync-persist"),
			SyncPersistChunk:               envInt("BRIDGE_SYNC_PERSIST_JOB_CHUNK", 75),
			SyncUpsertChunk:                envInt("BRIDGE_SYNC_UPSERT_CHUNK", 375),
			SyncMaxRequestsPerSecond:       envInt("BRIDGE_SYNC_MAX_REQUESTS_PER_SECOND", 2),
			SyncMaxHTTPRetries:             envInt("BRIDGE_SYNC_MAX_HTTP_RETRIES", 4),
			SyncReplicationTop:             envInt("BRIDGE_SYNC_REPLICATION_TOP", 2000),
			SyncIncrementalTop:             envInt("BRIDGE_SYNC_INCREMENTAL_TOP", 200),
			SyncIncludeMedia:               envBool("BRIDGE_SYNC_INCLUDE_MEDIA", true),
			SyncFullProperty:               envBool("BRIDGE_SYNC_FULL_PROPERTY", true),
			SyncNavHydrateAfterReplication: envBool("BRIDGE_SYNC_NAV_HYDRATE_AFTER_REPLICATION", true),
			SyncExpand:                     envBridgeSyncExpand(),
			SyncMaxChainedFetch:            envInt("BRIDGE_SYNC_MAX_CHAINED_FETCH_PAGES", 0),
			ImageRewriteHosts:              splitCSV(env("BRIDGE_IMAGE_REWRITE_HOSTS", "")),
		},
		Spark: SparkConfig{
			AccessToken:         firstNonEmpty(env("SPARK_ACCESS_TOKEN", ""), env("SPARK_API_KEY", "")),
			ReplicationHost:     env("SPARK_REPLICATION_HOST", "https://replication.sparkapi.com"),
			ReplicationReso:     env("SPARK_REPLICATION_RESO_ROOT", "Reso/OData"),
			APIHost:             env("SPARK_API_HOST", "https://sparkapi.com"),
			APIVersion:          env("SPARK_API_VERSION", "v1"),
			LiveResoRoot:        env("SPARK_LIVE_RESO_ROOT", "Reso/OData"),
			Datasets:            splitCSV(env("SPARK_DATASETS", "beaches")),
			Timeout:                  envDuration("SPARK_TIMEOUT", 90*time.Second),
			SyncFetchQueue:           env("SPARK_SYNC_FETCH_QUEUE", "spark-sync-fetch"),
			SyncPersistQueue:         env("SPARK_SYNC_PERSIST_QUEUE", "spark-sync-persist"),
			SyncPersistChunk:         envInt("SPARK_SYNC_PERSIST_JOB_CHUNK", 50),
			SyncUpsertChunk:          envInt("MLS_BEACHES_UPSERT_CHUNK_SIZE", envInt("SPARK_SYNC_UPSERT_CHUNK", 375)),
			SyncReplicationTop:       envInt("SPARK_SYNC_REPLICATION_TOP", 1000),
			SyncIncrementalTop:       envInt("SPARK_SYNC_INCREMENTAL_TOP", 1000),
			SyncExpand:               envSyncExpand(),
			SyncMaxChainedFetch:      envInt("SPARK_SYNC_MAX_CHAINED_FETCH_PAGES", 0),
			SyncMaxRequestsPerSecond: envInt("SPARK_SYNC_MAX_REQUESTS_PER_SECOND", 4),
			SyncMaxRequestsPer5Min:   envInt("SPARK_SYNC_MAX_REQUESTS_PER_5MIN", 1200),
			SyncMaxHTTPRetries:       envInt("SPARK_SYNC_MAX_HTTP_RETRIES", 4),
		},
		MLS: MLSConfig{
			LocalMirrorRollingMonths:       envMirrorRollingMonths(),
			ReplicaPageRetentionHours:      envInt("MLS_REPLICA_PAGE_RETENTION_HOURS", 24),
			ReplicaPageFailedRetentionDays: envInt("MLS_REPLICA_PAGE_FAILED_RETENTION_DAYS", 7),
			ProxyCacheRetentionDays:        envInt("MLS_PROXY_CACHE_RETENTION_DAYS", 30),
			ReplicationFreshnessMinutes:    envInt("MLS_REPLICATION_FRESHNESS_MINUTES", 15),
			StellarEnabled:                 envBool("MLS_STELLAR_ENABLED", true),
			StellarPersistChunk:            envInt("MLS_STELLAR_PERSIST_CHUNK_SIZE", 75),
			BeachesEnabled:                 envBool("MLS_BEACHES_ENABLED", true),
			BeachesPersistChunk:            envInt("MLS_BEACHES_PERSIST_CHUNK_SIZE", 50),
			SyncExpand:                     envSyncExpand(),
			SyncReplicationExpand:          env("MLS_SYNC_REPLICATION_EXPAND", ""),
			SyncKickoffQueue:              env("MLS_SYNC_KICKOFF_QUEUE", "sync-kickoff"),
			RateLimitRetrySeconds:         envInt("MLS_SYNC_RATE_LIMIT_RETRY_SECONDS", 300),
			TimeoutRetrySeconds:           envInt("MLS_SYNC_TIMEOUT_RETRY_SECONDS", 60),
			RateLimitMaxAttempts:          envInt("MLS_SYNC_RATE_LIMIT_MAX_ATTEMPTS", 50),
			ReplicationResumeStallMinutes: envInt("MLS_REPLICATION_RESUME_STALL_MINUTES", 3),
			ReplicationResumeCron:         env("MLS_REPLICATION_RESUME_CRON", "0 */2 * * * *"),
			MirrorKeyReconcileRetryMinutes: envInt("MLS_MIRROR_KEY_RECONCILE_RETRY_MINUTES", 30),
		},
		GIS: GISConfig{
			EdgeCacheTTL:         envDuration("GIS_EDGE_CACHE_TTL", 900*time.Second),
			OriginMaxDaysPrimary: envInt("GIS_ORIGIN_MAX_DAYS_PRIMARY", 90),
			OriginMaxDaysCounty:  envInt("GIS_ORIGIN_MAX_DAYS_COUNTY", 30),
			TeaserMaxFeatures:    envInt("GIS_TEASER_MAX_FEATURES", 40),
			TeaserCoordDecimals:  envInt("GIS_TEASER_COORD_DECIMALS", 4),
			MaxBboxSpanDeg:       envFloat("GIS_MAX_BBOX_SPAN_DEG", 0.35),
			MaxFeatures:          envInt("GIS_MAX_FEATURES", 500),
			SyncPageSize:           envInt("GIS_SYNC_PAGE_SIZE", 2000),
			SyncUpsertChunk:        envInt("GIS_SYNC_UPSERT_CHUNK", 500),
			HTTPTimeout:            envDuration("GIS_HTTP_TIMEOUT", 120*time.Second),
			SyncQueue:              env("GIS_SYNC_QUEUE", "default"),
			SyncPinellasEnterprise: envBool("GIS_SYNC_PINELLAS_ENTERPRISE", false),
			FloridaMLSCodes:        splitCSV(env("GIS_FLORIDA_MLS_CODES", "stellar")),
			Queue:                  env("GIS_QUEUE", "default"),
			BoundaryStaleDays:      envInt("GIS_BOUNDARY_STALE_DAYS", 90),
			ImportPath:             env("GIS_IMPORT_PATH", "/var/cache/geoidx/gis-imports"),
			ImportMaxBytes:         envInt64("GIS_IMPORT_MAX_BYTES", 512*1024*1024),
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
		Geocode: GeocodeConfig{
			GoogleMapsAPIKey:     env("GOOGLE_MAPS_GEOCODING_API_KEY", ""),
			HTTPTimeout:          envDuration("GEOCODING_TIMEOUT", 12*time.Second),
			EnrichQueue:          env("GEOCODE_QUEUE", "default"),
			EnrichBatchSize:      envInt("GEOCODE_BATCH_SIZE", 200),
			MaxRequestsPerSecond: envInt("GEOCODE_MAX_REQUESTS_PER_SECOND", 5),
		},
		Coingecko: CoingeckoConfig{
			APIKey:      env("COINGECKO_API_KEY", ""),
			BaseURL:     env("COINGECKO_BASE_URL", "https://api.coingecko.com/api/v3"),
			CacheTTL:    envDuration("COINGECKO_CACHE_TTL_SECONDS", 1200*time.Second),
			Queue:       env("COINGECKO_QUEUE", "default"),
			HTTPTimeout: envDuration("COINGECKO_HTTP_TIMEOUT", 12*time.Second),
		},
		FEMA: FEMAConfig{
			NFHLBaseURL:          env("FEMA_NFHL_BASE_URL", "https://hazards.fema.gov/arcgis/rest/services/public/NFHL/MapServer"),
			NFHLLayerID:          envInt("FEMA_NFHL_LAYER_ID", 28),
			EnrichBatchSize:      envInt("FEMA_FLOOD_ENRICH_BATCH_SIZE", 2000),
			FloodStaleDays:       envInt("FEMA_FLOOD_STALE_DAYS", 30),
			MaxRequestsPerSecond: envInt("FEMA_MAX_REQUESTS_PER_SECOND", 8),
			HTTPTimeout:          envDuration("FEMA_HTTP_TIMEOUT", 15*time.Second),
			UserAgent:            firstNonEmpty(env("FEMA_USER_AGENT", ""), "Quantyra-IDX-API/1.0 (+https://quantyralabs.com)"),
			EnrichQueue:          env("FEMA_ENRICH_QUEUE", "default"),
			CircuitFailThreshold: envInt("FEMA_CIRCUIT_FAIL_THRESHOLD", 5),
		},
		Comps: CompsConfig{
			ClosedCacheDays:    envInt("COMPS_CLOSED_CACHE_DAYS", 30),
			ClosedCacheMinHits: envInt("COMPS_CLOSED_CACHE_MIN_HITS", 3),
		},
		Scheduler: SchedulerConfig{
			LeaderLockKey:       envSchedulerLeaderLockKey(),
			StandbyPollInterval: time.Duration(envInt("SCHEDULER_STANDBY_POLL_SECONDS", 15)) * time.Second,
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

func envInt64(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
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

func envSchedulerLeaderLockKey() int64 {
	v := strings.TrimSpace(env("SCHEDULER_LEADER_LOCK_ID", ""))
	if v == "" {
		return 913374211
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 913374211
	}
	return id
}

// envBridgeSyncExpand is the OData $expand list for Bridge (Stellar navigation property names).
func envBridgeSyncExpand() string {
	return firstNonEmpty(
		env("BRIDGE_SYNC_EXPAND", ""),
		env("MLS_SYNC_EXPAND", ""),
		"Media,OpenHouses,Rooms,UnitTypes",
	)
}

func envPatroniHosts() []string {
	raw := env("PATRONI_ENDPOINTS", "100.126.103.51,100.114.117.46,100.115.75.119")
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
