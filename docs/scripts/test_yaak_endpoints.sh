#!/usr/bin/env bash
# Smoke-test all paths in docs/yaak-api-collection.json against a running idx-api.
# Delegates to the Go smoke test suite (response shape + AI failure reports).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

if command -v go >/dev/null 2>&1; then
  exec make test-api-smoke
fi

echo "Go toolchain not found; install Go 1.25+ or run: make test-api-smoke" >&2
exit 1
