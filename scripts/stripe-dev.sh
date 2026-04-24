#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if ! command -v stripe >/dev/null 2>&1; then
    echo "stripe CLI is required but not installed."
    exit 1
fi

if [ -f "${ROOT_DIR}/.env" ]; then
    set -a
    # shellcheck disable=SC1091
    source "${ROOT_DIR}/.env"
    set +a
fi

if [ -z "${STRIPE_SECRET:-}" ]; then
    echo "STRIPE_SECRET is not set in .env."
    exit 1
fi

ACTION="${1:-listen}"
FORWARD_BASE_URL="${IDX_PLATFORM_URL:-${APP_URL:-https://localhost}}"
FORWARD_TO="${FORWARD_BASE_URL%/}/stripe/webhook"

case "${ACTION}" in
    listen)
        exec stripe --api-key "${STRIPE_SECRET}" listen --forward-to "${FORWARD_TO}"
        ;;
    trigger-checkout-completed)
        exec stripe --api-key "${STRIPE_SECRET}" trigger checkout.session.completed
        ;;
    *)
        echo "Unknown action: ${ACTION}"
        echo "Usage: ${0} {listen|trigger-checkout-completed}"
        exit 1
        ;;
esac
