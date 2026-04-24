---
name: nginx
description: Configures Nginx reverse proxy for idx-images edge container
allowed-tools: [Read, Edit, Write, Glob, Grep, Bash]
---

# Nginx Skill

Configures the Nginx edge proxy for the idx-images service, which acts as a lightweight TLS termination layer that forwards all image requests to idx-api for authentication and caching.

## Quick Start

```bash
# Build the idx-images image from project root
docker build -f Dockerfile.idx-images -t quantyra/idx-images:latest .

# Test nginx config locally
docker run --rm -p 8080:8080 quantyra/idx-images:latest
```

## Key Concepts

| Concept | Detail |
|---------|--------|
| **Edge Role** | Nginx terminates TLS and proxies to `idx-api:8000` upstream |
| **Auth Delegation** | All auth happens in Laravel (`DomainOrTokenAuth`); Nginx forwards `Referer`, `Authorization`, `X-Domain-Slug` headers |
| **Route Scope** | Only `/images/*` is proxied; root and other paths return 404 |
| **Upstream** | `idx_api_images` upstream points to `idx-api:8000` with keepalive |
| **Health Check** | `GET /health` returns 200 OK for container health probes |

## Common Patterns

**Add custom headers to forward to idx-api**

```nginx
proxy_set_header X-Custom-Header $http_x_custom_header;
```

**Adjust cache behavior for image responses** (idx-api sets `Cache-Control: public, max-age=31536000, immutable`)

```nginx
location /images/ {
    proxy_pass http://idx_api_images;
    proxy_cache_valid 200 1y;  # Honor upstream cache headers
    proxy_buffering on;
}
```

**Restrict to specific paths**

```nginx
# Only proxy /images/ - all other paths 404 (security)
location / {
    return 404;
}
```

**Build-time configuration**

| File | Purpose |
|------|---------|
| `Dockerfile.idx-images` | Alpine nginx image, exposes 8080, copies config |
| `nginx.idx-images.conf` | Upstream definition, proxy headers, health endpoint |