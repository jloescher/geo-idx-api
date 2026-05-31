# Go Types Reference

## Contents
- Domain Models
- Configuration Types
- Queue Job Types
- Fiber Handler Types
- Database Types
- Custom Type Patterns

## Domain Models

Domain types live in `internal/domain/` and use struct tags for both `db` (sqlx scanning) and `json` (API serialization). Nullable columns use pointer types.

```go
// internal/domain/domain.go
type Domain struct {
    ID                 int64           `db:"id"`
    UserID             *int64          `db:"user_id"`        // nullable → pointer
    DomainSlug         string          `db:"domain_slug"`
    IsActive           bool            `db:"is_active"`
    MLSDataset         *string         `db:"mls_dataset"`    // nullable → pointer
    AllowedMLSDatasets json.RawMessage `db:"allowed_mls_datasets"` // JSONB
    TXTVerifiedAt      *time.Time      `db:"txt_verified_at"` // nullable → pointer
}
```

### WARNING: Non-pointer types for nullable columns

**The Problem:** Scanning a NULL into a non-pointer `string` or `time.Time` causes a scan error.

**Why This Breaks:** PostgreSQL NULL cannot be represented as a zero-value Go type. sqlx will return `sql: Scan error on column index X`.

**The Fix:** Always use `*string`, `*int64`, `*time.Time` for columns that allow NULL.

```go
// BAD
CreatedAt time.Time `db:"created_at"` // fails if column is NULL

// GOOD
CreatedAt *time.Time `db:"created_at"`
```

## Configuration Types

`config.Config` is a flat struct of sub-configs. Each sub-config is a value type (not pointer).

```go
type Config struct {
    App       AppConfig
    DB        DBConfig
    Queue     QueueConfig
    Bridge    BridgeConfig
    Spark     SparkConfig
    MLS       MLSConfig
    Scheduler SchedulerConfig
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
```

## Queue Job Types

Job type constants are stringly-typed. The payload is a JSON envelope with `type` and `args`.

```go
// internal/queue/types.go
const (
    TypeNoop                   = "noop"
    TypeMLSReplicationKickoff  = "mls.replication_kickoff"
    TypeMLSProxyCachePurge     = "mls.proxy_cache_purge"
    TypeBridgeFetchPage        = "bridge.fetch_page"
    TypeBridgePersistChunk     = "bridge.persist_chunk"
    TypeSparkFetchPage         = "spark.fetch_page"
    TypeSparkPersistChunk      = "spark.persist_chunk"
)
```

## Fiber Handler Types

Handlers depend on interfaces, not concrete types. This enables testing with fakes.

```go
// internal/handler/bridge/handler.go
type proxyCacheStore interface {
    Get(ctx context.Context, partition, fingerprint string) ([]byte, bool, error)
    Put(ctx context.Context, partition, fingerprint string, body []byte) error
}

type Handler struct {
    cfg        config.Config
    factory    mlsClientFactory
    proxyCache proxyCacheStore
    logger     *slog.Logger
}
```

### WARNING: Accepting interfaces, returning structs

**The Problem:** Returning interfaces from constructors forces all consumers to depend on the interface and prevents extending the concrete type.

**The Fix:** Accept interfaces in constructor params (for testability), return concrete pointers.

```go
// BAD
func NewService(store CacheStore) CacheStore { ... }

// GOOD
func NewService(store CacheStore) *Service { ... }
```

## Database Types

The `DB` wrapper holds both `pgxpool.Pool` and `sqlx.DB`:

```go
// internal/repository/db.go
type DB struct {
    Pool *pgxpool.Pool   // advanced: LISTEN/NOTIFY, advisory locks, batch
    SQLX *sqlx.DB        // simple: SELECT → struct scan via db tags
}
```

### json.RawMessage for JSONB columns

Use `json.RawMessage` (`[]byte`) for PostgreSQL JSONB columns that need passthrough without structuring:

```go
AllowedMLSDatasets json.RawMessage `db:"allowed_mls_datasets"`
```

Do NOT unmarshal into `map[string]any` at scan time — defer to the consumer.

## MLS Payload Types

Replication payloads split across typed JSONB columns:

```go
// internal/service/mls/listing_payload.go
type ExpandedPayload struct {
    Media      json.RawMessage
    Unit       json.RawMessage
    Room       json.RawMessage
    OpenHouse  json.RawMessage
    HasMedia   bool
    HasUnit    bool
    HasRoom    bool
    HasOpenHouse bool
}
```

`MergeMirrorListing` reassembles these into a flat RESO Property JSON for API responses. `raw_data` wins on key collision; `custom_fields` is flat-merged last.

## Search Request Types

Pointer fields distinguish "not set" from "zero value":

```go
type SearchRequest struct {
    MinPrice   *float64 `json:"min_price"`   // nil = not provided
    ActiveOnly  *bool   `json:"active_only"`  // nil = not provided, false = explicit false
    Statuses   []string `json:"statuses"`
}
```

## Type Decision Guide

| PostgreSQL type | Go type | Notes |
|-----------------|---------|-------|
| `bigint NOT NULL` | `int64` | Value type |
| `bigint` (nullable) | `*int64` | Pointer |
| `text NOT NULL` | `string` | Value type |
| `text` (nullable) | `*string` | Pointer |
| `boolean NOT NULL` | `bool` | Value type |
| `timestamptz NOT NULL` | `time.Time` | Value type |
| `timestamptz` (nullable) | `*time.Time` | Pointer |
| `jsonb` | `json.RawMessage` | Passthrough |
| `geometry` (PostGIS) | `string` (WKT) or custom | See **geospatial** skill |