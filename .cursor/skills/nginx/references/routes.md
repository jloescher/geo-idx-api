# Routes

## When to use
Configuring Nginx location blocks for the idx-images edge proxy. Use these patterns when adding new proxy paths or modifying how requests route between Nginx and the upstream idx-api service.

## Patterns

**Proxy all image requests to idx-api**
```nginx
location /images/ {
    proxy_pass http://idx_api_images;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

**Health check endpoint for container probes**
```nginx
location /health {
    access_log off;
    return 200 "ok\n";
    add_header Content-Type text/plain;
}
```

**Block all non-image paths at the edge**
```nginx
location / {
    return 404;
}
```

## Warning
Nginx must forward `Referer`, `Authorization`, and `X-Domain-Slug` headers exactly as received—any header name normalization or case changes will cause `DomainOrTokenAuth` middleware in idx-api to reject valid requests with 401/403.