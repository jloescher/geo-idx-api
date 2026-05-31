#!/usr/bin/env bash
# verify-idx-images-edge.sh — read-only checks for idx-images nginx edge (Coolify / Cloudflare).
set -euo pipefail

COOLIFY_HOST="${COOLIFY_HOST:-localhost}"
EDGE_URL="${IDX_IMAGES_PUBLIC_URL:-https://idx-images.quantyralabs.cc}"
FAIL=0

echo "== idx-images edge verification =="
echo "Coolify host: ${COOLIFY_HOST}"
echo "Public URL:   ${EDGE_URL}"

if curl -sf "http://${COOLIFY_HOST}:8080/health" | grep -q OK; then
  echo "OK  local :8080/health"
else
  echo "FAIL local :8080/health (is idx-images container running on ${COOLIFY_HOST}?)"
  FAIL=1
fi

if curl -sf "${EDGE_URL}/health" | grep -q OK; then
  echo "OK  ${EDGE_URL}/health"
else
  echo "WARN ${EDGE_URL}/health (Cloudflare origin may be down; check Coolify idx-images-nyc/atl)"
  FAIL=1
fi

if [[ -n "${YAAK_BEARER_TOKEN:-}" && -n "${YAAK_DOMAIN_SLUG:-}" && -n "${SMOKE_LISTING_KEY:-}" && -n "${SMOKE_PHOTO_ID:-}" ]]; then
  CODE=$(curl -sf -o /dev/null -w '%{http_code}' \
    -H "Authorization: Bearer ${YAAK_BEARER_TOKEN}" \
    -H "X-Domain-Slug: ${YAAK_DOMAIN_SLUG}" \
    "${EDGE_URL}/images/${SMOKE_LISTING_KEY}/${SMOKE_PHOTO_ID}" || echo "000")
  if [[ "${CODE}" == "200" ]]; then
    echo "OK  edge image proxy HTTP ${CODE}"
  else
    echo "WARN edge image proxy HTTP ${CODE} (expected 200 with auth)"
    FAIL=1
  fi
else
  echo "SKIP edge image route (set YAAK_BEARER_TOKEN, YAAK_DOMAIN_SLUG, SMOKE_LISTING_KEY, SMOKE_PHOTO_ID to test)"
fi

exit "${FAIL}"
