# Database

## When to use
Nginx does not interact with databases directly. These patterns describe how Nginx configuration supports database-backed features in idx-api.

## Patterns

**Support long-running cache refresh requests**
```nginx
proxy_connect_timeout 30s;
proxy_send_timeout 30s;
proxy_read_timeout 60s;  # Allow time for cache miss fetches from Bridge
```

**Log upstream timing for cache hit/miss analysis**
```nginx
log_format upstream_time '$remote_addr - $remote_user [$time_local] '
    '"$request" $status $body_bytes_sent '
    'upstream_response_time $upstream_response_time';
access_log /var/log/nginx/access.log upstream_time;
```

**Disable buffering for large image responses**
```nginx
proxy_buffering off;
proxy_request_buffering off;
```

## Warning
Database-backed cache freshness (IMAGE_CACHE_TTL) is managed entirely by idx-api—Nginx does not check cache expiration. Ensure idx-api returns proper `Cache-Control: immutable` headers so downstream CDNs cache correctly.