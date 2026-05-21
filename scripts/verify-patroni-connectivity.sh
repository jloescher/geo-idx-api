#!/usr/bin/env bash
# Verify Patroni primary reachability from a Coolify host (re-db or re-node-02).
# Usage: export DB_HOST DB_PORT DB_DATABASE DB_USERNAME DB_PASSWORD DB_SSLMODE=require
#        ./scripts/verify-patroni-connectivity.sh
# Optional: API_URL=https://idx-api-nyc.example.com  (checks GET /readyz)

set -euo pipefail

: "${DB_HOST:?DB_HOST required}"
: "${DB_PORT:=5432}"
: "${DB_DATABASE:?DB_DATABASE required}"
: "${DB_USERNAME:?DB_USERNAME required}"
: "${DB_PASSWORD:?DB_PASSWORD required}"
: "${DB_SSLMODE:=require}"

export PGPASSWORD="${DB_PASSWORD}"

echo "==> psql to Patroni primary (${DB_HOST}:${DB_PORT}/${DB_DATABASE})"
psql "host=${DB_HOST} port=${DB_PORT} dbname=${DB_DATABASE} user=${DB_USERNAME} sslmode=${DB_SSLMODE}" \
  -c "SELECT version();" \
  -c "SELECT pg_is_in_recovery() AS replica, current_setting('server_version_num') AS version_num;"

if [[ -n "${API_URL:-}" ]]; then
  echo "==> GET ${API_URL}/readyz"
  curl -fsS "${API_URL%/}/readyz" | head -c 500
  echo
fi

echo "OK: Patroni connectivity checks passed"
