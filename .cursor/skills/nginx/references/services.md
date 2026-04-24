# Services

## When to use
Understanding how Nginx fits into the idx-images service architecture and its relationship with upstream services.

## Patterns

**Define upstream with keepalive for connection pooling**
```nginx
upstream idx_api_images {
    server idx-api:8000;
    keepalive 32;
}
```

**Forward critical auth headers to upstream**
```nginx
proxy_set_header Referer $http_referer;
proxy_set_header Authorization $http_authorization;
proxy_set_header X-Domain-Slug $http_x_domain_slug;
```

**Honor upstream cache headers for CDN compatibility**
```nginx
proxy_cache_valid 200 1y;
proxy_ignore_headers Cache-Control;  # Let idx-api control cache
```

## Warning
The idx-images container does not perform authentication—it blindly proxies to idx-api. Never expose Nginx directly to the internet without TLS termination; always run behind a CDN or load balancer that handles TLS.