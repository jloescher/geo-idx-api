# Errors

## When to use
Handling error responses and troubleshooting issues in the idx-images Nginx proxy layer.

## Patterns

**Return custom error for upstream failures**
```nginx
error_page 502 503 504 /50x.html;
location = /50x.html {
    return 503 '{"error":"Service temporarily unavailable"}';
    add_header Content-Type application/json;
}
```

**Hide upstream server details**
```nginx
proxy_hide_header X-Powered-By;
proxy_hide_header Server;
more_clear_headers Server;  # Requires headers-more module
```

**Log errors for debugging upstream issues**
```nginx
error_log /var/log/nginx/error.log warn;
proxy_intercept_errors on;
```

## Warning
If clients receive 401/403 errors, check that Nginx is forwarding headers exactly—do not add, remove, or modify `Authorization`, `X-Domain-Slug`, or `Referer` headers. Use `curl -v` against idx-api directly to verify; if it works direct but fails through Nginx, a header is being dropped or modified.