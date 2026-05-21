# Storage S3 Workflows

## Contents
- Adding S3 to the Image Cache
- Migrating from Filesystem to S3
- Configuring MinIO/R2 for Development
- Multi-DC S3 Deployment
- Cache Invalidation and Lifecycle

## Adding S3 to the Image Cache

### Prerequisites

```bash
# new code to add — AWS SDK v2 dependency
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/s3
go get github.com/aws/aws-sdk-go-v2/credentials
```

### Step-by-Step Integration

Copy this checklist and track progress:

- [ ] Add `S3Config` struct to `internal/config/config.go`
- [ ] Add env vars to `.env.example`: `S3_ENDPOINT`, `S3_BUCKET`, `S3_REGION`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_PATH_STYLE`, `S3_PRESIGN_TTL`
- [ ] Create `internal/storage/s3.go` with client factory and presign helper
- [ ] Extend `images.Handler` with optional `s3Client` field
- [ ] Modify `Show()` to check S3 before filesystem
- [ ] Add async S3 write after upstream fetch
- [ ] Test with `IMAGE_CACHE_PATH=/tmp/test-images S3_ENDPOINT=localhost:9000 go test ./internal/handler/images/...`
- [ ] Verify presigned URL redirect works from browser

### Config Extension

```go
// new code to add — in internal/config/config.go Load()
Images: ImageCacheConfig{
    Path: env("IMAGE_CACHE_PATH", "/var/cache/geoidx/images"),
    TTL:  envDuration("IMAGE_CACHE_TTL", 86400*time.Second),
    S3: S3Config{
        Endpoint:   env("S3_ENDPOINT", ""),
        Region:     env("S3_REGION", "us-east-1"),
        Bucket:     env("S3_BUCKET", ""),
        AccessKey:  env("S3_ACCESS_KEY", ""),
        SecretKey:  env("S3_SECRET_KEY", ""),
        PathStyle:  envBool("S3_PATH_STYLE", false),
        PresignTTL: envDuration("S3_PRESIGN_TTL", 900*time.Second),
    },
},
```

## Migrating from Filesystem to S3

### WARNING: Do Not Remove Filesystem Cache

The filesystem cache (NVMe) provides sub-millisecond reads. S3 adds network latency. Keep filesystem as **L1** and add S3 as **L2** for cross-DC sharing.

### Migration Workflow

1. Deploy with S3 configured but filesystem cache still active
2. New images write to **both** tiers (write-through)
3. Existing disk cache continues serving until TTL expires
4. After `IMAGE_CACHE_TTL` (24h default), all images are served from S3 or re-fetched
5. Optionally backfill: iterate disk cache → upload to S3

```go
// new code to add — backfill script (run once, not in hot path)
func BackfillToS3(ctx context.Context, s3Client *s3.Client, diskPath, bucket string) error {
    return filepath.Walk(diskPath, func(path string, info os.FileInfo, err error) error {
        if err != nil || info.IsDir() {
            return nil
        }
        rel, _ := filepath.Rel(diskPath, path)
        data, err := os.ReadFile(path)
        if err != nil {
            return nil
        }
        _, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
            Bucket:      aws.String(bucket),
            Key:         aws.String(rel),
            Body:        bytes.NewReader(data),
            ContentType: aws.String("image/jpeg"),
        })
        if err != nil {
            slog.Warn("backfill failed", "key", rel, "err", err)
        }
        return nil
    })
}
```

## Configuring MinIO/R2 for Development

### Local MinIO via Docker

Add to `docker-compose.dev.yml`:

```yaml
# new code to add
minio:
  image: minio/minio:latest
  command: server /data --console-address ":9001"
  ports:
    - "9000:9000"
    - "9001:9001"
  environment:
    MINIO_ROOT_USER: minioadmin
    MINIO_ROOT_PASSWORD: minioadmin
  volumes:
    - minio-data:/data

volumes:
  minio-data:
```

### Environment Variables

```bash
# .env for local development
S3_ENDPOINT=http://localhost:9000
S3_REGION=us-east-1
S3_BUCKET=idx-images
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_PATH_STYLE=true          # required for MinIO
S3_PRESIGN_TTL=900
```

### Create Bucket

```bash
# new code to add — one-time setup
docker compose exec minio mc alias set local http://localhost:9000 minioadmin minioadmin
docker compose exec minio mc mb local/idx-images
```

### WARNING: Path-Style vs Virtual-Hosted

MinIO and Cloudflare R2 require **path-style** addressing (`S3_PATH_STYLE=true`). AWS S3 uses virtual-hosted by default. Mixing these causes `NoSuchBucket` errors that are hard to diagnose.

## Multi-DC S3 Deployment

See the **deploy-coolify** skill for the full multi-DC topology.

### S3 Placement

| Option | Latency | Consistency | Cost |
|--------|---------|-------------|------|
| Single-region S3 bucket | 5-20ms cross-DC for one region | Strong (S3 guarantee) | Standard |
| R2 with global edge | ~0ms from edge cache | Eventual (edge) | Lower egress |
| MinIO per-DC with replication | ~0ms local | Async replication | Higher ops |

### Recommended: Single Shared Bucket

For this project's scale, a single S3 bucket in one region is simplest. Both DCs read/write to the same bucket. Presigned URLs let clients fetch directly from S3, bypassing the API.

```env
# Shared across all Coolify apps (see deploy-coolify skill)
S3_ENDPOINT=https://s3.us-east-1.amazonaws.com
S3_BUCKET=quantyra-idx-images
S3_REGION=us-east-1
S3_ACCESS_KEY=...
S3_SECRET_KEY=...
```

## Cache Invalidation and Lifecycle

### S3 Lifecycle Rules

Set bucket lifecycle to expire objects after `IMAGE_CACHE_TTL` to prevent unbounded storage growth:

```go
// new code to add — bucket lifecycle config (run once via admin tool)
func SetImageLifecycle(ctx context.Context, client *s3.Client, bucket string, ttlDays int32) error {
    _, err := client.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
        Bucket: aws.String(bucket),
        LifecycleConfiguration: &types.LifecycleConfiguration{
            Rules: []types.LifecycleRule{
                {
                    ID:     aws.String("expire-image-cache"),
                    Status: types.ExpirationStatusEnabled,
                    Filter: &types.LifecycleRuleFilterMemberPrefix{
                        Value: aws.String(""),
                    },
                    Expiration: &types.LifecycleExpiration{
                        Days: aws.Int32(ttlDays),
                    },
                },
            },
        },
    })
    return err
}
```

### On-Demand Invalidation

When an image changes upstream (same `listingKey` + `photoID`, new content), overwrite the S3 key. The SHA-256 key is based on identity, not content, so the next GET after PutObject returns the new version.

### WARNING: Cache Stampede on Miss

Multiple concurrent requests for the same uncached image will all fetch from upstream and write to S3. This is safe (last-write-wins, same content) but wastes bandwidth. For high-traffic images, consider singleflight:

```go
// new code to add — deduplicate concurrent fetches
import "golang.org/x/sync/singleflight"

var fetchGroup singleflight.Group

func (h *Handler) fetchAndCache(ctx context.Context, listingKey, photoID string) ([]byte, string, error) {
    key := listingKey + "/" + photoID
    v, err, _ := fetchGroup.Do(key, func() (interface{}, error) {
        data, ct, err := h.fetchUpstream(ctx, listingKey, photoID)
        if err != nil {
            return nil, err
        }
        h.writeToCache(ctx, listingKey, photoID, data, ct)
        return cacheResult{data, ct}, nil
    })
    if err != nil {
        return nil, "", err
    }
    r := v.(cacheResult)
    return r.data, r.ct, nil
}
```