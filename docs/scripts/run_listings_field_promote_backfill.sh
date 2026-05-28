#!/usr/bin/env bash
# Run listings field promote backfill via psql.
#
# Default reconnect mode: one batch per psql connection (survives SSL / HAProxy drops).
# DSN: docs/scripts/.env.backfill.local (gitignored) > $GOOSE_DBSTRING > repo .env
#
# Usage:
#   docs/scripts/run_listings_field_promote_backfill.sh [batch_size] [mode] [log]
#   docs/scripts/run_listings_field_promote_backfill.sh check   # connectivity + schema only
#   mode: reconnect (default) | monolithic
#
# Long run (survives terminal close):
#   nohup docs/scripts/run_listings_field_promote_backfill.sh 500 reconnect /tmp/listings_field_promote_backfill.log >>/tmp/listings_field_promote_backfill.log 2>&1 &
#   tail -f /tmp/listings_field_promote_backfill.log
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

BATCH_SIZE="${1:-500}"
MODE="${2:-reconnect}"
LOG="${3:-/tmp/listings_field_promote_backfill.log}"
LOCK_DIR="${BACKFILL_LOCK_DIR:-/tmp/listings_field_promote_backfill.lock.d}"
SKIP_INSTALL="${SKIP_INSTALL:-0}"
BACKFILL_BATCH_TIMEOUT_SEC="${BACKFILL_BATCH_TIMEOUT_SEC:-120}"
BACKFILL_BATCH_PAUSE_SEC="${BACKFILL_BATCH_PAUSE_SEC:-}"
BACKFILL_MAX_CONSECUTIVE_FAILURES="${BACKFILL_MAX_CONSECUTIVE_FAILURES:-25}"
BACKFILL_MIN_BATCH_SIZE="${BACKFILL_MIN_BATCH_SIZE:-100}"

# Prefer direct Patroni DSN for long backfills (bypasses HAProxy :5000 idle limits).
if [[ -n "${BACKFILL_DSN:-}" ]]; then
  GOOSE_DBSTRING="$BACKFILL_DSN"
elif [[ -n "${GOOSE_DBSTRING_DIRECT:-}" ]]; then
  GOOSE_DBSTRING="$GOOSE_DBSTRING_DIRECT"
fi

PSQL_OPTS=(-v ON_ERROR_STOP=1)
export PGCONNECT_TIMEOUT="${PGCONNECT_TIMEOUT:-15}"
export PGAPPNAME="${PGAPPNAME:-idx_listings_field_promote_backfill}"

psql_cmd() {
  psql "${PSQL_OPTS[@]}" "$GOOSE_DBSTRING" "$@"
}

# Append libpq keepalive query params when missing (Tailscale / HAProxy paths).
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

default_batch_pause() {
  if [[ -n "$BACKFILL_BATCH_PAUSE_SEC" ]]; then
    echo "$BACKFILL_BATCH_PAUSE_SEC"
    return
  fi
  if [[ "$(dsn_port)" == "5000" ]]; then
    echo "2"
  else
    echo "0"
  fi
}

# Run psql with a wall-clock cap so hung HAProxy sessions do not block for 8+ minutes.
psql_timed() {
  local max="$1"
  shift
  local tmp
  tmp="$(mktemp "${TMPDIR:-/tmp}/backfill_psql.XXXXXX")"
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
    echo "ERROR: psql timed out after ${max}s (connection likely hung on HAProxy/Tailscale)" >>"$tmp"
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

psql_ping() {
  psql_cmd -Atq -c "SELECT 1" >/dev/null 2>&1
}

# CALL only (no SET in same -c). Commits on successful psql exit.
psql_call_step() {
  local phase="$1"
  local batch="$2"
  PGOPTIONS="${PGOPTIONS:--c statement_timeout=0 -c lock_timeout=30s}" \
    psql_timed "$BACKFILL_BATCH_TIMEOUT_SEC" \
    -c "CALL listings_field_promote_step_call(${batch}::bigint, '${phase}'::text);"
}

connection_error_hint() {
  local port
  port="$(dsn_port)"
  if [[ "$port" == "5000" ]]; then
    say "HINT: port 5000 is HAProxy (~60–120s idle limit). Set BACKFILL_DSN or GOOSE_DBSTRING_DIRECT to Patroni :5432 on Tailscale."
  else
    say "HINT: check Tailscale to Patroni primary, or lower batch size (e.g. 200)."
  fi
}

# Print to log and to terminal when one is attached (nohup still shows this once at start).
say() {
  local line="[$(date -u +%H:%M:%S)] $*"
  echo "$line" | tee -a "$LOG"
  if [[ -t 1 ]]; then
    : # already on stdout via tee
  elif [[ -w /dev/tty ]] 2>/dev/null; then
    echo "$line" >/dev/tty
  fi
}

dsn_port() {
  if [[ "$GOOSE_DBSTRING" =~ ://[^/@]+@[^/:]+:([0-9]+)/ ]]; then
    echo "${BASH_REMATCH[1]}"
  elif [[ "$GOOSE_DBSTRING" =~ ://[^/@]+@([^/:]+)/ ]]; then
    echo "5432"
  else
    echo ""
  fi
}

# Subcommand: quick connectivity check (no lock, no backfill).
run_check() {
  echo "host: $(echo "$GOOSE_DBSTRING" | sed -E 's#.*@([^/:]+).*#\1#') port: $(dsn_port)"
  psql_cmd -Atq -c "SELECT 'connected', current_database(), version();" || exit 1
  psql_cmd -Atq -c "
SELECT proname FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
WHERE n.nspname = 'public' AND p.proname = 'listings_field_promote_step_call';" || {
    echo "ERROR: listings_field_promote_step_call missing — run without SKIP_INSTALL once." >&2
    exit 1
  }
  if [[ "$(dsn_port)" == "5000" ]]; then
    echo "WARN: port 5000 (HAProxy) — use BACKFILL_DSN to Patroni :5432 for long backfills."
  fi
  echo "OK: DB reachable and step_call procedure exists."
}

if [[ "${1:-}" == "check" ]]; then
  run_check
  exit 0
fi

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
  if [[ -f "$LOCK_DIR/pid" ]]; then
    holder_pid="$(cat "$LOCK_DIR/pid" 2>/dev/null || true)"
  fi
  if [[ -n "$holder_pid" ]] && kill -0 "$holder_pid" 2>/dev/null; then
    echo "Another backfill is running (pid ${holder_pid}, lock ${LOCK_DIR})." >&2
    echo "  tail -f ${LOG}" >&2
    exit 1
  fi

  echo "Removing stale lock (${LOCK_DIR})..." >&2
  rm -rf "$LOCK_DIR"
  mkdir "$LOCK_DIR" || {
    echo "Could not acquire lock ${LOCK_DIR}" >&2
    exit 1
  }
  echo "$$" >"$LOCK_DIR/pid"
  trap release_lock EXIT INT TERM
}

preflight() {
  say "Preflight: connectivity..."
  psql_cmd -Atq -c "SELECT 1" >/dev/null 2>&1 || {
    say "ERROR: cannot connect with GOOSE_DBSTRING"
    exit 1
  }
  say "Preflight: connected."

  local port
  port="$(dsn_port)"
  if [[ "$MODE" == "monolithic" && "$port" == "5000" ]]; then
    say "ERROR: monolithic mode on port 5000 (HAProxy) will timeout. Use reconnect mode."
    exit 1
  fi
  if [[ "$port" == "5000" ]]; then
    say "NOTE: port 5000 = HAProxy (idle ~60–120s). Using batch timeout=${BACKFILL_BATCH_TIMEOUT_SEC}s, pause=$(default_batch_pause)s between batches."
    say "      For fewer errors: BACKFILL_DSN=postgres://USER:PASS@<patroni-tailscale>:5432/geoidxapi?sslmode=require"
  fi

  local missing_cols
  missing_cols="$(psql_cmd -Atq -c "
SELECT string_agg(col, ', ' ORDER BY col)
FROM (VALUES
  ('garage_spaces'), ('mls_area_major'), ('days_on_market'), ('tax_annual_amount'),
  ('heating_yn'), ('cooling_yn'), ('carport_yn'), ('attached_garage_yn'),
  ('internet_consumer_comment_yn'), ('internet_address_display_yn'),
  ('internet_entire_listing_display_yn'), ('internet_automated_valuation_display_yn'),
  ('idx_participation_yn'), ('idx_office_participation_yn'),
  ('unparsed_address'), ('public_remarks')
) AS expected(col)
WHERE NOT EXISTS (
  SELECT 1 FROM information_schema.columns c
  WHERE c.table_schema = 'public' AND c.table_name = 'listings' AND c.column_name = expected.col
);")"
  missing_cols="$(echo "${missing_cols:-}" | tr -d $'\r\n' | xargs)"
  if [[ -n "$missing_cols" ]]; then
    say "ERROR: missing listings columns: ${missing_cols} (run migration 00006)"
    exit 1
  fi
}

install_sql() {
  if [[ "$SKIP_INSTALL" == "1" ]]; then
    say "SKIP_INSTALL=1 — skipping SQL file install"
  else
    say "Installing SQL (see ${LOG})..."
    psql_cmd -f docs/scripts/listings_field_promote_backfill.sql >>"$LOG" 2>&1 || {
      say "ERROR: SQL install failed — last 20 log lines:"
      tail -20 "$LOG" | while read -r l; do say "$l"; done
      exit 1
    }
    say "SQL install done."
  fi

  local args
  args="$(psql_cmd -Atq -c "
SELECT COALESCE(
  (SELECT pg_get_function_identity_arguments(p.oid)
   FROM pg_proc p
   JOIN pg_namespace n ON n.oid = p.pronamespace
   WHERE n.nspname = 'public'
     AND p.proname = 'listings_field_promote_step_call'
     AND p.prokind = 'p'
   LIMIT 1),
  'MISSING'
);")"
  args="$(echo "$args" | tr -d $'\r\n' | xargs)"
  if [[ -z "$args" || "$args" == "MISSING" ]]; then
    say "ERROR: listings_field_promote_step_call not found — run without SKIP_INSTALL once"
    exit 1
  fi
  say "OK: listings_field_promote_step_call (${args})"
}

run_phase_reconnect() {
  local phase="$1"
  local batch_ok=0
  local attempt=0
  local total=0
  local consec_fail=0
  local retry_delay=5
  local max_retry_delay=45
  local batch_size="$BATCH_SIZE"
  local pause_sec
  pause_sec="$(default_batch_pause)"

  say "Phase ${phase}: starting (batch_size=${batch_size}, timeout=${BACKFILL_BATCH_TIMEOUT_SEC}s, pause=${pause_sec}s)"

  while true; do
    attempt=$((attempt + 1))
    say "Phase ${phase} attempt ${attempt}: batch_size=${batch_size} (ok_batches=${batch_ok}, consec_fail=${consec_fail})..."
    out="$(psql_call_step "$phase" "$batch_size" 2>&1)" && {
      echo "$out" >>"$LOG"
      if [[ -t 1 ]] || [[ -w /dev/tty ]] 2>/dev/null; then
        echo "$out" | grep NOTICE || true
        [[ -w /dev/tty ]] 2>/dev/null && echo "$out" | grep NOTICE >/dev/tty 2>/dev/null || true
      fi
      updated="$(echo "$out" | sed -n 's/.*updated \([0-9][0-9]*\) rows.*/\1/p' | tail -1)"
      updated="${updated:-0}"
      total=$((total + updated))
      consec_fail=0
      retry_delay=5
      batch_ok=$((batch_ok + 1))
      say "Phase ${phase} batch ${batch_ok}: updated ${updated} rows (phase cumulative ${total}; not unique listings)"

      if [[ "$updated" -eq 0 ]]; then
        say "Phase ${phase} complete (${batch_ok} batches, ${total} rows this run)."
        break
      fi

      if [[ "$pause_sec" != "0" ]]; then
        sleep "$pause_sec"
      fi
      continue
    }

    echo "$out" >>"$LOG"
    consec_fail=$((consec_fail + 1))
    local err_line
    err_line="$(echo "$out" | grep -E '^(psql:|ERROR:|SSL |connection )' | head -1)"
    [[ -n "$err_line" ]] && say "Phase ${phase}: ${err_line}"

    if is_connection_error "$out"; then
      if [[ "$consec_fail" -ge 3 && "$batch_size" -gt "$BACKFILL_MIN_BATCH_SIZE" ]]; then
        batch_size=$((batch_size / 2))
        if [[ "$batch_size" -lt "$BACKFILL_MIN_BATCH_SIZE" ]]; then
          batch_size="$BACKFILL_MIN_BATCH_SIZE"
        fi
        say "Phase ${phase}: reducing batch_size to ${batch_size} after ${consec_fail} failures"
      fi

      if [[ "$consec_fail" -ge "$BACKFILL_MAX_CONSECUTIVE_FAILURES" ]]; then
        say "ERROR: ${consec_fail} consecutive connection failures — stopping (progress is saved per successful batch)."
        connection_error_hint
        exit 1
      fi

      local wait_sec
      wait_sec="$(retry_jitter "$retry_delay")"
      say "Phase ${phase}: connection error — retry in ${wait_sec}s (consec_fail=${consec_fail}/${BACKFILL_MAX_CONSECUTIVE_FAILURES})"
      sleep "$wait_sec"
      retry_delay=$((retry_delay * 2))
      if [[ "$retry_delay" -gt "$max_retry_delay" ]]; then
        retry_delay="$max_retry_delay"
      fi
      psql_ping || say "Phase ${phase}: ping failed before retry (will try anyway)"
      continue
    fi

    say "Phase ${phase} attempt ${attempt}: fatal SQL error — not retrying"
    echo "$out" >&2
    exit 1
  done
}

acquire_lock

say "=== listings_field_promote_backfill pid=$$ ==="
say "mode=${MODE} batch_size=${BATCH_SIZE} log=${LOG}"
say "Watch progress: tail -f ${LOG}"

preflight
install_sql

say "Running backfill..."

if [[ "$MODE" == "monolithic" ]]; then
  PGOPTIONS="${PGOPTIONS:--c statement_timeout=0 -c lock_timeout=30s}" \
    psql_cmd -c "CALL run_listings_field_promote_backfill(${BATCH_SIZE}::bigint);" >>"$LOG" 2>&1
else
  run_phase_reconnect "primary"
  run_phase_reconnect "scalars"
fi

say "Done. Fast verify:"
psql_cmd <<'SQL' | tee -a "$LOG"
SELECT
  EXISTS (SELECT 1 FROM listings WHERE custom_fields ? 'GarageSpaces' LIMIT 1) AS cf_garage_left,
  EXISTS (SELECT 1 FROM listings WHERE custom_fields ? 'InternetEntireListingDisplayYN' LIMIT 1) AS cf_ield_left,
  EXISTS (SELECT 1 FROM listings WHERE custom_fields ? 'IDXParticipationYN' LIMIT 1) AS cf_idx_left,
  EXISTS (SELECT 1 FROM listings WHERE raw_data ? 'InternetEntireListingDisplayYN' LIMIT 1) AS raw_ield_left,
  EXISTS (SELECT 1 FROM listings WHERE raw_data ? 'IDXParticipationYN' LIMIT 1) AS raw_idx_left;
SQL
