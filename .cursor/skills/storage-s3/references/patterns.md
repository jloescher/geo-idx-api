# Storage S3 Patterns

## Contents
- S3 Client Initialization
- Dual-Tier Cache Strategy
- Presigned URL Redirect vs Proxy
- Multi-DC Shared Cache
- Anti-Patterns

## S3 Client Initialization

### WARNING: Shared Client Singleton

Go's AWS SDK v2 clients are safe for concurrent use. Create **one** client at handler init, not per request.

```go
// new code to add — initialize once in handler constructor
import (
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func newS3Client(cfg S3Config) (*s3.Client, error) {
    opts := []func(*config.LoadOptions) error{
        config.WithRegion(cfg.Region),
    }
    if cfg.Endpoint != "" {
        opts = append(opts, config.WithBaseEndpoint(cfg.Endpoint))
    }
    if cfg.AccessKey != "" {
        opts = append(opts, config.WithCredentialsProvider(
            credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
        ))
    }
    awscfg, err := config.LoadDefaultConfig(context.Background(), opts...)
    if err != nil {
        return nil, fmt.Errorf("s3 config: %w", err)
    }
    return s3.NewFromConfig(awscfg, func(o *s3.Options) {
        o.UsePathStyle = cfg.PathStyle
    }), nil
}
```

### Lazy Initialization

```go
// new code to add — skip S3 when not configured
func (h *Handler) Show(c *fiber.Ctx) error {
    if h.s3Client != nil {
        // S3 path
    }
    // existing filesystem-only path
}
```

## Dual-Tier Cache Strategy

### Tier Order Matters

Check the **shared** tier (S3) before the **local** tier (disk). In multi-DC, S3 may have images fetched by the other DC.

```go
// new code to add — tiered cache lookup
func (h *Handler) cachedImage(ctx context.Context, listingKey, photoID string) ([]byte, string, error) {
    key := h.s3Key(listingKey, photoID)

    // Tier 1: S3 (shared across DCs)
    if h.s3Client != nil {
        obj, err := h.s3Client.GetObject(ctx, &s3.GetObjectInput{
            Bucket: aws.String(h.cfg.Images.S3.Bucket),
            Key:    aws.String(key),
        })
        if err == nil {
            defer obj.Body.Close()
            data, _ := io.ReadAll(obj.Body)
            return data, aws.ToString(obj.ContentType), nil
        }
    }

    // Tier 2: Local filesystem
    cachePath := h.cacheFile(listingKey, photoID)
    if data, err := os.ReadFile(cachePath); err == nil {
        return data, "image/jpeg", nil
    }

    return nil, "", fmt.Errorf("cache miss")
}
```

### Write-Through to Both Tiers

After upstream fetch, write to S3 first (shared) then disk (local fast path):

```go
// new code to add — write to both cache tiers
func (h *Handler) writeToCache(ctx context.Context, listingKey, photoID string, data []byte, ct string) {
    key := h.s3Key(listingKey, photoID)

    // S3 write (shared, async is acceptable)
    if h.s3Client != nil {
        if _, err := h.s3Client.PutObject(ctx, &s3.PutObjectInput{
            Bucket:      aws.String(h.cfg.Images.S3.Bucket),
            Key:         aws.String(key),
            Body:        bytes.NewReader(data),
            ContentType: aws.String(ct),
        }); err != nil {
            slog.Error("s3 cache write failed", "key", key, "err", err)
        }
    }

    // Disk write (local L2)
    cachePath := h.cacheFile(listingKey, photoID)
    _ = os.MkdirAll(filepath.Dir(cachePath), 0o755)
    if err := os.WriteFile(cachePath, data, 0o644); err != nil {
        slog.Error("disk cache write failed", "path", cachePath, "err", err)
    }
}
```

## Presigned URL Redirect vs Proxy

### WARNING: Do Not Proxy S3 Through the API

Proxying S3 bytes through the API wastes bandwidth and adds latency. Use **presigned URL redirects** instead.

```go
// BAD — double hop: S3 → API → client
data, _ := s3Client.GetObject(ctx, ...)
return c.Send(data)

// GOOD — redirect client directly to S3
url, _ := presigner.PresignGET(ctx, bucket, key, 15*time.Minute)
return c.Redirect(url, fiber.StatusFound)
```

**Why Redirect Breaks Some Clients:** Some older browsers and CDN configurations don't follow 302 for image tags. If `idx-images` Nginx needs to serve the image directly, keep the proxy path for that edge case and use redirects for API consumers.

```go
// new code to add — conditional redirect vs proxy
func (h *Handler) Show(c *fiber.Ctx) error {
    // idx-images Nginx needs the bytes directly
    if c.Get("X-IDX-Internal") == "true" || c.Get("Accept") == "application/octet-stream" {
        return h.proxyFromS3(c, key)
    }
    // Browser/CDN clients get a redirect
    url, err := h.presignGET(c.Context(), key)
    if err == nil {
        return c.Redirect(url, fiber.StatusFound)
    }
    // Fallback to upstream
    return h.fetchUpstream(c, listingKey, photoID)
}
```

## Multi-DC Shared Cache

The current filesystem approach means each DC maintains its own cache (docs/coolify-deployment.md §8). S3 eliminates redundant MLS origin fetches.

| Aspect | Filesystem (current) | S3 (proposed) |
|--------|---------------------|---------------|
| Cache sharing | Per-DC, isolated | Shared across NYC + ATL |
| Miss rate | 2x fetches for same image | 1 fetch, both DCs served |
| Persistence | Lost on pod restart | Durable across deployments |
| Latency | NVMe (~0.1ms) | Network (~5-20ms) |
| Cost | Included in host | Per-GB storage + requests |

### WARNING: S3 Latency in Hot Path

Do NOT add S3 latency to every image request. Use **presigned URLs** to bypass the API entirely, or keep filesystem as L1 with S3 as write-through L2.

## Anti-Patterns

### WARNING: Public Bucket Without Access Control

```go
// BAD — anyone with the URL can read all cached images
s3Client.PutObject(ctx, &s3.PutObjectInput{
    Bucket: aws.String("idx-images-public"),
    Key:    aws.String(key),
    ACL:    types.ObjectCannedACL_PUBLIC_READ, // NEVER
})
```

**The Fix:** Use presigned URLs with short TTL. Keep the bucket private.

### WARNING: Synchronous S3 Put in Request Path

```go
// BAD — blocks the response while uploading to S3
s3Client.PutObject(ctx, &s3.PutObjectInput{...}) // 50-200ms
return c.Send(body)
```

**The Fix:** Fire-and-forget goroutine with error logging, or a background queue job.

```go
// GOOD — respond immediately, cache asynchronously
go func() {
    if _, err := h.s3Client.PutObject(context.Background(), ...); err != nil {
        slog.Error("async s3 write failed", "key", key, "err", err)
    }
}()
return c.Send(body)
```