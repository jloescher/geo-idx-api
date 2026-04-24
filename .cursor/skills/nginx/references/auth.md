# Authentication

## When to use
Configuring Nginx to properly forward authentication credentials to idx-api where `DomainOrTokenAuth` middleware performs actual validation.

## Patterns

**Preserve Authorization header for Bearer tokens**
```nginx
proxy_set_header Authorization $http_authorization;
proxy_pass_header Authorization;
```

**Forward domain identification headers**
```nginx
proxy_set_header X-Domain-Slug $http_x_domain_slug;
proxy_set_header Referer $http_referer;
```

**Pass original client IP for audit logging**
```nginx
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
```

## Warning
Never add authentication logic to Nginx configuration (like `auth_basic` or JWT validation). All auth decisions must happen in Laravel's `DomainOrTokenAuth` middleware—Nginx is only a transparent proxy. Adding auth in Nginx would bypass critical audit logging to `bridge_proxy_audit_logs`.