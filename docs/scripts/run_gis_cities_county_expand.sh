#!/usr/bin/env bash
# Expand gis_cities to one row per (city_name, county_slug) via psql on Patroni primary.
#
# DSN: docs/scripts/.env.backfill.local (same as listings backfill) > GOOSE_DBSTRING > .env
#
# Usage:
#   docs/scripts/run_gis_cities_county_expand.sh check
#   docs/scripts/run_gis_cities_county_expand.sh [cities_per_batch] [mode] [log]
#   mode: reconnect (default) | monolithic
#
# Run BEFORE Goose 00008 (gis_cities.county NOT NULL).
#
# Long run:
#   nohup docs/scripts/run_gis_cities_county_expand.sh 5 reconnect /tmp/gis_cities_county_expand.log >>/tmp/gis_cities_county_expand.log 2>&1 &
#   tail -f /tmp/gis_cities_county_expand.log
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$ROOT"

if [[ -f "$SCRIPT_DIR/.env.backfill.local" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "$SCRIPT_DIR/.env.backfill.local"
  set +a
fi

if [[ -z "${GOOSE_DBSTRING:-}" && -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

if [[ -z "${GOOSE_DBSTRING:-}" ]]; then
  echo "Set GOOSE_DBSTRING or create docs/scripts/.env.backfill.local" >&2
  exit 1
fi

CITIES_LIMIT="${1:-5}"
MODE="${2:-reconnect}"
LOG="${3:-/tmp/gis_cities_county_expand.log}"
LOCK_DIR="${GIS_EXPAND_LOCK_DIR:-/tmp/gis_cities_county_expand.lock.d}"
SKIP_INSTALL="${SKIP_INSTALL:-0}"
COMMIT_EVERY="${GIS_COMMIT_EVERY:-5}"
BATCH_TIMEOUT_SEC="${GIS_BATCH_TIMEOUT_SEC:-300}"
BATCH_PAUSE_SEC="${GIS_BATCH_PAUSE_SEC:-}"
MAX_CONSECUTIVE_FAILURES="${GIS_MAX_CONSECUTIVE_FAILURES:-25}"

if [[ -n "${BACKFILL_DSN:-}" ]]; then
  GOOSE_DBSTRING="$BACKFILL_DSN"
elif [[ -n "${GOOSE_DBSTRING_DIRECT:-}" ]]; then
  GOOSE_DBSTRING="$GOOSE_DBSTRING_DIRECT"
fi

PSQL_OPTS=(-v ON_ERROR_STOP=1)
export PGCONNECT_TIMEOUT="${PGCONNECT_TIMEOUT:-15}"
export PGAPPNAME="${PGAPPNAME:-idx_gis_cities_county_expand}"

psql_cmd() {
  psql "${PSQL_OPTS[@]}" "$GOOSE_DBSTRING" "$@"
}

dsn_with_keepalives() {
  if [[ "$GOOSE_DBSTRING" == *keepalives=* ]]; then
    echo "$GOOSE_DBSTRING"
  else
    local sep='?'
    [[ "$GOOSE_DBSTRING" == *'?'* ]] && sep='&'
    echo "${GOOSE_DBSTRING}${sep}keepalives=1&keepalives_idle=20&keepalives_interval=5&keepalives_count=5"
  fi
}
GOOSE_DBSTRING="$(dsn_with_keepalives)"

dsn_port() {
  if [[ "$GOOSE_DBSTRING" =~ ://[^/@]+@[^/:]+:([0-9]+)/ ]]; then
    echo "${BASH_REMATCH[1]}"
  elif [[ "$GOOSE_DBSTRING" =~ ://[^/@]+@([^/:]+)/ ]]; then
    echo "5432"
  else
    echo ""
  fi
}

say() {
  local line="[$(date -u +%H:%M:%S)] $*"
  echo "$line" | tee -a "$LOG"
  if [[ -t 1 ]]; then
    :
  elif [[ -w /dev/tty ]] 2>/dev/null; then
    echo "$line" >/dev/tty
  fi
}

psql_timed() {
  local max="$1"
  shift
  local tmp
  tmp="$(mktemp "${TMPDIR:-/tmp}/gis_expand_psql.XXXXXX")"
  ( psql "${PSQL_OPTS[@]}" "$GOOSE_DBSTRING" "$@" >"$tmp" 2>&1 ) &
  local pid=$!
  local waited=0
  while kill -0 "$pid" 2>/dev/null && (( waited < max )); do
    sleep 1
    waited=$((waited + 1))
  done
  if kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
    echo "ERROR: psql timed out after ${max}s" >>"$tmp"
    cat "$tmp"
    rm -f "$tmp"
    return 124
  fi
  wait "$pid"
  local rc=$?
  cat "$tmp"
  rm -f "$tmp"
  return $rc
}

psql_call_step() {
  PGOPTIONS="${PGOPTIONS:--c statement_timeout=0 -c lock_timeout=30s}" \
    psql_timed "$BATCH_TIMEOUT_SEC" \
    -c "CALL gis_expand_cities_step_call(${CITIES_LIMIT}::int);"
}

psql_ping() {
  psql_cmd -Atq -c "SELECT 1" >/dev/null 2>&1
}

default_batch_pause() {
  if [[ -n "$BATCH_PAUSE_SEC" ]]; then
    echo "$BATCH_PAUSE_SEC"
    return
  fi
  if [[ "$(dsn_port)" == "5000" ]]; then
    echo "2"
  else
    echo "0"
  fi
}

is_connection_error() {
  local msg="$1"
  grep -qiE '(^psql: (error|fatal):|SSL error|SSL SYSCALL|timed out after|server closed the connection unexpectedly|connection to server .*(failed|was lost)|broken pipe|could not connect to server|connection timed out|no route to host)' <<<"$msg"
}

retry_jitter() {
  local base="$1"
  if command -v jot >/dev/null 2>&1; then
    echo $((base + $(jot -r 1 0 3)))
  else
    echo $((base + RANDOM % 4))
  fi
}

connection_error_hint() {
  if [[ "$(dsn_port)" == "5000" ]]; then
    say "HINT: use BACKFILL_DSN to Patroni leader :5432 (not HAProxy :5000)."
  fi
}

run_check() {
  echo "host: $(echo "$GOOSE_DBSTRING" | sed -E 's#.*@([^/:]+).*#\1#') port: $(dsn_port)"
  psql_cmd -Atq -c "SELECT 'connected', current_database(), pg_is_in_recovery();" || exit 1
  psql_cmd -Atq -c "
SELECT proname FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
WHERE n.nspname = 'public' AND p.proname = 'gis_expand_cities_step_call';" || {
    echo "ERROR: gis_expand_cities_step_call missing — run without SKIP_INSTALL once." >&2
    exit 1
  }
  local null_ct pending
  null_ct="$(psql_cmd -Atq -c "SELECT COUNT(*) FROM gis_cities WHERE county IS NULL;")"
  pending="$(psql_cmd -Atq -c "
SELECT COUNT(*) FROM (
  SELECT DISTINCT source_generation, city_name FROM gis_cities WHERE county IS NULL
) q;")"
  echo "NULL county rows: ${null_ct}"
  echo "Pending city/generation pairs: ${pending}"
  if [[ "$(dsn_port)" == "5000" ]]; then
    echo "WARN: port 5000 (HAProxy) — set BACKFILL_DSN to Patroni :5432 for expand."
  fi
  echo "OK: ready to expand."
}

if [[ "${1:-}" == "check" ]]; then
  run_check
  exit 0
fi

release_lock() {
  rm -rf "$LOCK_DIR"
}

acquire_lock() {
  if mkdir "$LOCK_DIR" 2>/dev/null; then
    echo "$$" >"$LOCK_DIR/pid"
    trap release_lock EXIT INT TERM
    return 0
  fi
  local holder_pid=""
  [[ -f "$LOCK_DIR/pid" ]] && holder_pid="$(cat "$LOCK_DIR/pid" 2>/dev/null || true)"
  if [[ -n "$holder_pid" ]] && kill -0 "$holder_pid" 2>/dev/null; then
    echo "Another GIS expand is running (pid ${holder_pid}). tail -f ${LOG}" >&2
    exit 1
  fi
  rm -rf "$LOCK_DIR"
  mkdir "$LOCK_DIR"
  echo "$$" >"$LOCK_DIR/pid"
  trap release_lock EXIT INT TERM
}

preflight() {
  say "Preflight: connectivity..."
  psql_cmd -Atq -c "SELECT 1" >/dev/null 2>&1 || {
    say "ERROR: cannot connect"
    exit 1
  }
  say "Preflight: connected (port $(dsn_port))."

  psql_cmd -Atq -c "SELECT 1 FROM information_schema.tables
    WHERE table_schema='public' AND table_name IN ('gis_cities','gis_counties');" | grep -q 1 || {
    say "ERROR: gis_cities / gis_counties tables missing"
    exit 1
  }

  psql_cmd -Atq -c "SELECT 1 FROM information_schema.columns
    WHERE table_schema='public' AND table_name='gis_cities' AND column_name='county';" | grep -q 1 || {
    say "ERROR: gis_cities.county column missing"
    exit 1
  }

  if [[ "$MODE" == "monolithic" && "$(dsn_port)" == "5000" ]]; then
    say "ERROR: monolithic on port 5000 will timeout — use reconnect mode."
    exit 1
  fi
}

install_sql() {
  if [[ "$SKIP_INSTALL" == "1" ]]; then
    say "SKIP_INSTALL=1 — skipping SQL install"
  else
    say "Installing SQL (see ${LOG})..."
    psql_cmd -f docs/scripts/gis_cities_county_expand.sql >>"$LOG" 2>&1 || {
      say "ERROR: SQL install failed"
      tail -20 "$LOG" >&2
      exit 1
    }
    say "SQL install done."
  fi

  local ok
  ok="$(psql_cmd -Atq -c "
SELECT EXISTS (
  SELECT 1 FROM pg_proc p
  JOIN pg_namespace n ON n.oid = p.pronamespace
  WHERE n.nspname = 'public' AND p.proname = 'gis_expand_cities_step_call' AND p.prokind = 'p'
);")"
  if [[ "${ok// /}" != "t" ]]; then
    say "ERROR: gis_expand_cities_step_call not found"
    exit 1
  fi
  say "OK: gis_expand_cities_step_call installed"
}

step_done() {
  local out="$1"
  if grep -q 'no cities with NULL county (pending=0)' <<<"$out"; then
    return 0
  fi
  local processed
  processed="$(echo "$out" | sed -n 's/.*processed \([0-9][0-9]*\) cities.*/\1/p' | tail -1)"
  [[ "${processed:-1}" -eq 0 ]]
}

run_reconnect() {
  local batch_ok=0
  local attempt=0
  local total_cities=0
  local consec_fail=0
  local retry_delay=5
  local max_retry_delay=45
  local pause_sec
  pause_sec="$(default_batch_pause)"

  say "Reconnect mode: up to ${CITIES_LIMIT} cities per connection (timeout=${BATCH_TIMEOUT_SEC}s)"

  while true; do
    attempt=$((attempt + 1))
    say "Step attempt ${attempt} (ok_steps=${batch_ok}, consec_fail=${consec_fail})..."
    out="$(psql_call_step 2>&1)" && {
      echo "$out" >>"$LOG"
      echo "$out" | grep NOTICE || true

      if step_done "$out"; then
        say "Expand complete (no pending NULL-county cities)."
        break
      fi

      processed="$(echo "$out" | sed -n 's/.*processed \([0-9][0-9]*\) cities.*/\1/p' | tail -1)"
      processed="${processed:-0}"
      total_cities=$((total_cities + processed))
      batch_ok=$((batch_ok + 1))
      consec_fail=0
      retry_delay=5
      say "Step ${batch_ok}: processed ${processed} cities (run total ${total_cities})"

      if [[ "$pause_sec" != "0" ]]; then
        sleep "$pause_sec"
      fi
      continue
    }

    echo "$out" >>"$LOG"
    consec_fail=$((consec_fail + 1))
    local err_line
    err_line="$(echo "$out" | grep -E '^(psql:|ERROR:|SSL )' | head -1)"
    [[ -n "$err_line" ]] && say "${err_line}"

    if is_connection_error "$out"; then
      if [[ "$consec_fail" -ge "$MAX_CONSECUTIVE_FAILURES" ]]; then
        say "ERROR: ${consec_fail} consecutive connection failures."
        connection_error_hint
        exit 1
      fi
      local wait_sec
      wait_sec="$(retry_jitter "$retry_delay")"
      say "Connection error — retry in ${wait_sec}s"
      sleep "$wait_sec"
      retry_delay=$((retry_delay * 2))
      [[ "$retry_delay" -gt "$max_retry_delay" ]] && retry_delay="$max_retry_delay"
      psql_ping || say "Ping failed (retrying anyway)"
      continue
    fi

    say "Fatal SQL error — not retrying"
    echo "$out" >&2
    exit 1
  done
}

run_monolithic() {
  say "Monolithic: CALL run_gis_cities_county_expand(${COMMIT_EVERY}) (single connection)"
  PGOPTIONS="${PGOPTIONS:--c statement_timeout=0 -c lock_timeout=30s}" \
    psql_timed 7200 \
    -c "CALL run_gis_cities_county_expand(${COMMIT_EVERY}::int);" >>"$LOG" 2>&1
}

acquire_lock

say "=== gis_cities_county_expand pid=$$ ==="
say "mode=${MODE} cities_per_batch=${CITIES_LIMIT} log=${LOG}"
say "Watch: tail -f ${LOG}"

preflight
install_sql

if [[ "$MODE" == "monolithic" ]]; then
  run_monolithic
else
  run_reconnect
fi

say "Post-verify:"
psql_cmd -Atq -c "
SELECT
  (SELECT COUNT(*) FROM gis_cities WHERE county IS NULL) AS null_counties,
  (SELECT COUNT(DISTINCT (source_generation, city_name)) FROM gis_cities) AS city_gen_pairs;" | tee -a "$LOG"

null_left="$(psql_cmd -Atq -c "SELECT COUNT(*) FROM gis_cities WHERE county IS NULL;")"
null_left="${null_left// /}"
if [[ "$null_left" != "0" ]]; then
  say "WARN: ${null_left} rows still have NULL county — see emergency UPDATE in gis_cities_county_expand.sql before 00008"
else
  say "OK: zero NULL counties — safe to apply Goose 00008"
fi
