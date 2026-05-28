#!/usr/bin/env bash
# Smoke-test all paths in docs/yaak-api-collection.json against a running idx-api.
# Read-only toward MLS/upstream where possible. Uses GOOSE_DBSTRING only for SELECT fixtures.
# Does not run admin mutating routes (flood-enrich) or enqueue jobs.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

# Prefer production host for smoke tests; APP_URL in .env is often dev/staging.
BASE_URL="${YAAK_BASE_URL:-https://idx.quantyralabs.cc}"
if [[ -n "${APP_URL:-}" && "${YAAK_USE_APP_URL:-}" == "1" ]]; then
  BASE_URL="${APP_URL}"
fi
BASE_URL="${BASE_URL%/}"
DOMAIN_SLUG="${YAAK_DOMAIN_SLUG:-}"
DATASET="${YAAK_DATASET:-stellar}"
BBOX="${YAAK_BBOX:--82.8,27.9,-82.6,28.0}"

PASS=0
FAIL=0
SKIP=0
WARN=0

_red() { printf '\033[31m'; }
_green() { printf '\033[32m'; }
_yellow() { printf '\033[33m'; }
_cyan() { printf '\033[36m'; }
_reset() { printf '\033[0m'; }

note() { printf "%b\n" "$*"; }

# --- auth ---
TOKEN="${YAAK_BEARER_TOKEN:-}"
AUTH_PAYLOAD=""
if [[ -z "$TOKEN" && -n "${ADMIN_SEED_EMAIL:-}" && -n "${ADMIN_SEED_PASSWORD:-}" ]]; then
  AUTH_PAYLOAD=$(ADMIN_SEED_EMAIL="$ADMIN_SEED_EMAIL" ADMIN_SEED_PASSWORD="$ADMIN_SEED_PASSWORD" python3 -c \
    'import json,os; print(json.dumps({"email":os.environ["ADMIN_SEED_EMAIL"],"password":os.environ["ADMIN_SEED_PASSWORD"]}))')
  AUTH_JSON=$(curl -sS -X POST "${BASE_URL}/api/auth/token" \
    -H "Content-Type: application/json" \
    -d "$AUTH_PAYLOAD" 2>/dev/null || true)
  TOKEN=$(printf '%s' "$AUTH_JSON" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('token',''))" 2>/dev/null || true)
fi

if [[ -z "$DOMAIN_SLUG" && -n "${GOOSE_DBSTRING:-}" ]]; then
  DOMAIN_SLUG=$(psql "$GOOSE_DBSTRING" -t -A -c \
    "SELECT domain_slug FROM domains WHERE is_active ORDER BY id LIMIT 1;" 2>/dev/null | tr -d '[:space:]' || true)
fi
DOMAIN_SLUG="${DOMAIN_SLUG:-example.com}"

auth_curl() {
  if [[ -n "$TOKEN" ]]; then
    curl "$@" -H "Authorization: Bearer ${TOKEN}" -H "X-Domain-Slug: ${DOMAIN_SLUG}"
  else
    curl "$@"
  fi
}

if [[ -z "$TOKEN" ]]; then
  note "$(_yellow)WARN: No bearer token (set YAAK_BEARER_TOKEN or ADMIN_SEED_*); MLS routes expect 401.$(_reset)"
  ((WARN++)) || true
  MLS_CODES=(401)
  SEARCH_CODES=(401)
else
  MLS_CODES=(200 404 502)
  SEARCH_CODES=(200 502)
fi

# --- DB fixtures (SELECT only) ---
LISTING_KEY=""
MLS_LISTING_ID=""
PHOTO_ID=""
if [[ -n "${GOOSE_DBSTRING:-}" ]]; then
  read -r LISTING_KEY MLS_LISTING_ID <<<"$(psql "$GOOSE_DBSTRING" -t -A -c \
    "SELECT listing_key, COALESCE(mls_listing_id, '') FROM listings
     WHERE dataset_slug = '${DATASET}' AND LOWER(TRIM(COALESCE(standard_status,''))) = 'active'
     ORDER BY modification_timestamp DESC NULLS LAST LIMIT 1;" 2>/dev/null | tr '|' ' ')" || true
  PHOTO_ID=$(psql "$GOOSE_DBSTRING" -t -A -c \
    "SELECT COALESCE(media->0->>'MediaKey', media->0->>'PhotoId', '')
     FROM listings WHERE dataset_slug = '${DATASET}' AND listing_key = '$(printf "%s" "$LISTING_KEY" | sed "s/'/''/g")'
     AND jsonb_array_length(COALESCE(media,'[]'::jsonb)) > 0 LIMIT 1;" 2>/dev/null | tr -d '[:space:]' || true)
fi
LISTING_KEY="${LISTING_KEY:-STELLAR-PLACEHOLDER}"
MLS_LISTING_ID="${MLS_LISTING_ID:-1}"
PHOTO_ID="${PHOTO_ID:-1}"

# --- HTTP helper ---
# Usage: call METHOD PATH [curl args...]
# Sets globals: LAST_CODE LAST_BODY
call() {
  local method="$1" path="$2"
  shift 2
  local url="${BASE_URL}${path}"
  local tmp
  tmp=$(mktemp)
  LAST_CODE=$(auth_curl -sS -o "$tmp" -w '%{http_code}' -X "$method" "$url" "$@" 2>/dev/null || echo "000")
  LAST_BODY=$(cat "$tmp")
  rm -f "$tmp"
}

# Usage: expect_ok "name" "GET" "/path" 200 401 -- curl args
expect_ok() {
  local name="$1" method="$2" path="$3"
  shift 3
  local allowed=()
  while [[ $# -gt 0 && "$1" =~ ^[0-9]+$ ]]; do
    allowed+=("$1")
    shift
  done
  call "$method" "$path" "$@"
  local ok=0 exp
  for exp in "${allowed[@]}"; do
    if [[ "$LAST_CODE" == "$exp" ]]; then ok=1; break; fi
  done
  if [[ $ok -eq 1 ]]; then
    note "$(_green)PASS$(_reset) [$LAST_CODE] $method $path — $name"
    ((PASS++)) || true
  else
    local snippet
    snippet=$(printf '%s' "$LAST_BODY" | head -c 120 | tr '\n' ' ')
    note "$(_red)FAIL$(_reset) [$LAST_CODE] $method $path — $name (want ${allowed[*]}) ${snippet}"
    ((FAIL++)) || true
  fi
}

skip() {
  note "$(_cyan)SKIP$(_reset) $*"
  ((SKIP++)) || true
}

note ""
note "$(_cyan)=== Yaak API smoke tests ===$(_reset)"
note "Base:    $BASE_URL"
note "Domain:  $DOMAIN_SLUG"
note "Dataset: $DATASET"
note "Fixture listing_key: $LISTING_KEY"
note ""

# Infrastructure (no auth)
expect_ok "healthz" GET /healthz 200
expect_ok "readyz" GET /readyz 200 503
expect_ok "health/replicas" GET /health/replicas 200 503
expect_ok "metrics" GET /metrics 200

# Auth
if [[ -n "$TOKEN" ]]; then
  expect_ok "auth user" GET /api/auth/user 200
  if [[ -n "$AUTH_PAYLOAD" ]]; then
    expect_ok "auth token (login)" POST /api/auth/token 200 \
      -H "Content-Type: application/json" -d "$AUTH_PAYLOAD"
  fi
else
  expect_ok "auth user (no token)" GET /api/auth/user 401
  if [[ -n "${ADMIN_SEED_EMAIL:-}" && -n "${ADMIN_SEED_PASSWORD:-}" ]]; then
    skip "POST /api/auth/token (login did not return a token — check credentials)"
  else
    skip "POST /api/auth/token (set ADMIN_SEED_EMAIL / ADMIN_SEED_PASSWORD in .env)"
  fi
fi

# MLS collections
QS="?dataset=${DATASET}&\$top=2"
expect_ok "listings" GET "/api/v1/listings?dataset=${DATASET}&limit=2" "${MLS_CODES[@]}"
expect_ok "listing by id" GET "/api/v1/listings/${MLS_LISTING_ID}?dataset=${DATASET}" "${MLS_CODES[@]}"
expect_ok "properties" GET "/api/v1/properties${QS}" "${MLS_CODES[@]}"
expect_ok "properties POST" POST "/api/v1/properties?dataset=${DATASET}" "${MLS_CODES[@]}" \
  -H "Content-Type: application/json" -d '{"city":"Largo","limit":2}'
expect_ok "property by key" GET "/api/v1/properties/${LISTING_KEY}?dataset=${DATASET}&\$select=ListingKey,City" "${MLS_CODES[@]}"
expect_ok "agents" GET "/api/v1/agents?dataset=${DATASET}&limit=2" "${MLS_CODES[@]}"
expect_ok "agent by id" GET "/api/v1/agents/1?dataset=${DATASET}" "${MLS_CODES[@]}"
expect_ok "offices" GET "/api/v1/offices?dataset=${DATASET}&limit=2" "${MLS_CODES[@]}"
expect_ok "office by id" GET "/api/v1/offices/1?dataset=${DATASET}" "${MLS_CODES[@]}"
expect_ok "reso-offices" GET "/api/v1/reso-offices${QS}" "${MLS_CODES[@]}"
expect_ok "reso-office by key" GET "/api/v1/reso-offices/x?dataset=${DATASET}" "${MLS_CODES[@]}"
expect_ok "openhouses" GET "/api/v1/openhouses?dataset=${DATASET}&limit=2" "${MLS_CODES[@]}"
expect_ok "openhouse by id" GET "/api/v1/openhouses/1?dataset=${DATASET}" "${MLS_CODES[@]}"
expect_ok "reso-openhouses" GET "/api/v1/reso-openhouses${QS}" "${MLS_CODES[@]}"
expect_ok "reso-openhouse by key" GET "/api/v1/reso-openhouses/x?dataset=${DATASET}" "${MLS_CODES[@]}"
expect_ok "members" GET "/api/v1/members${QS}" "${MLS_CODES[@]}"
expect_ok "member by key" GET "/api/v1/members/x?dataset=${DATASET}" "${MLS_CODES[@]}"
expect_ok "lookup" GET "/api/v1/lookup?dataset=${DATASET}" "${MLS_CODES[@]}"

# Public records (Bridge pub API — may 502 if upstream unavailable)
expect_ok "pub parcels" GET "/api/v1/pub/parcels?dataset=${DATASET}&limit=2" "${MLS_CODES[@]}"
expect_ok "pub parcel" GET "/api/v1/pub/parcels/1?dataset=${DATASET}" "${MLS_CODES[@]}"
expect_ok "pub parcel assessments" GET "/api/v1/pub/parcels/1/assessments?dataset=${DATASET}" "${MLS_CODES[@]}"
expect_ok "pub parcel transactions" GET "/api/v1/pub/parcels/1/transactions?dataset=${DATASET}" "${MLS_CODES[@]}"
expect_ok "pub assessments" GET "/api/v1/pub/assessments?dataset=${DATASET}&limit=2" "${MLS_CODES[@]}"
expect_ok "pub transactions" GET "/api/v1/pub/transactions?dataset=${DATASET}&limit=2" "${MLS_CODES[@]}"

# Search (Largo regression body)
SEARCH_BODY='{"min_price":250000,"min_beds":2,"city":"Largo","statuses":["Active"],"low_risk_floodzone":true,"max_monthly_fees":500,"page":{"limit":24,"skip":0}}'
expect_ok "search" POST "/api/v1/search?dataset=${DATASET}" "${SEARCH_CODES[@]}" \
  -H "Content-Type: application/json" -d "$SEARCH_BODY"

# GIS
GIS_CODES=(200 400)
if [[ -z "$TOKEN" ]]; then GIS_CODES=(401); fi
expect_ok "gis bbox" GET "/api/v1/gis?bbox=${BBOX}" "${GIS_CODES[@]}"
expect_ok "mls gis" GET "/api/v1/mls/${DATASET}/gis?bbox=${BBOX}" "${GIS_CODES[@]}"

# Ops
STATS_CODES=(200 403)
if [[ -z "$TOKEN" ]]; then STATS_CODES=(401); fi
expect_ok "bridge stats" GET "/api/v1/bridge/stats?dataset=${DATASET}" "${STATS_CODES[@]}"

# Comps (off-market minimal — read-only subject, may still call upstream for comps)
COMPS_BODY='{"subject":{"type":"off_market","lat":27.916,"lng":-82.769,"bedrooms":3,"bathrooms":2,"living_area_sqft":1800},"mode":"A","scope":{"type":"radius","radius_miles":3},"filters":{"max_sold_comps":3,"sold_months_back":12}}'
COMPS_CODES=(200 422 502)
if [[ -z "$TOKEN" ]]; then COMPS_CODES=(401); fi
expect_ok "comps run" POST "/api/v1/comps/run?dataset=${DATASET}" "${COMPS_CODES[@]}" \
  -H "Content-Type: application/json" -d "$COMPS_BODY"

# Images
expect_ok "listing photo" GET "/images/${LISTING_KEY}/${PHOTO_ID}" "${MLS_CODES[@]}"

# Admin (session cookie required — never POST flood-enrich on production)
skip "GET /api/v1/admin/monitoring (requires dashboard session_id cookie)"
skip "POST /api/v1/admin/flood-enrich (mutating; requires session — not run)"

note ""
note "$(_cyan)=== Summary ===$(_reset)"
note "$(_green)PASS: $PASS$(_reset)  $(_red)FAIL: $FAIL$(_reset)  $(_cyan)SKIP: $SKIP$(_reset)  $(_yellow)WARN: $WARN$(_reset)"
if [[ $FAIL -gt 0 ]]; then
  exit 1
fi
