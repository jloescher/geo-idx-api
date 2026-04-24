#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ROOT_DIR}/.env"

if ! command -v stripe >/dev/null 2>&1; then
    echo "stripe CLI is required but not installed."
    exit 1
fi

if [ -f "${ENV_FILE}" ]; then
    set -a
    # shellcheck disable=SC1091
    source "${ENV_FILE}"
    set +a
fi

if [ -z "${STRIPE_SECRET:-}" ]; then
    echo "STRIPE_SECRET is not set in .env."
    exit 1
fi

ACTION="${1:-listen}"

resolve_forward_base_url() {
    if [ -n "${STRIPE_WEBHOOK_FORWARD_URL:-}" ]; then
        echo "${STRIPE_WEBHOOK_FORWARD_URL}"
        return
    fi

    if [ "${APP_ENV:-}" = "local" ] && [ -n "${IDX_PLATFORM_HOSTS:-}" ]; then
        IFS=',' read -r -a hosts <<< "${IDX_PLATFORM_HOSTS}"
        for host in "${hosts[@]}"; do
            clean_host="$(echo "${host}" | xargs)"
            if [[ "${clean_host}" == dev-* ]]; then
                echo "https://${clean_host}"
                return
            fi
        done
    fi

    if [ -n "${IDX_PLATFORM_URL:-}" ]; then
        echo "${IDX_PLATFORM_URL}"
        return
    fi

    echo "${APP_URL:-https://localhost}"
}

sync_webhook_secret() {
    if [ ! -f "${ENV_FILE}" ]; then
        echo ".env file not found. Cannot sync STRIPE_WEBHOOK_SECRET."
        exit 1
    fi

    new_secret="$(stripe --api-key "${STRIPE_SECRET}" listen --print-secret)"
    if [ -z "${new_secret}" ]; then
        echo "Failed to fetch webhook secret from Stripe CLI."
        exit 1
    fi

    if rg -n '^STRIPE_WEBHOOK_SECRET=' "${ENV_FILE}" >/dev/null 2>&1; then
        perl -0pi -e "s/^STRIPE_WEBHOOK_SECRET=.*/STRIPE_WEBHOOK_SECRET=${new_secret}/m" "${ENV_FILE}"
    else
        printf '\nSTRIPE_WEBHOOK_SECRET=%s\n' "${new_secret}" >> "${ENV_FILE}"
    fi

    echo "Updated STRIPE_WEBHOOK_SECRET in .env"
}

FORWARD_BASE_URL="$(resolve_forward_base_url)"
FORWARD_TO="${FORWARD_BASE_URL%/}/stripe/webhook"

case "${ACTION}" in
    listen)
        exec stripe --api-key "${STRIPE_SECRET}" listen --forward-to "${FORWARD_TO}"
        ;;
    listen-sync-secret)
        sync_webhook_secret
        exec stripe --api-key "${STRIPE_SECRET}" listen --forward-to "${FORWARD_TO}"
        ;;
    sync-webhook-secret)
        sync_webhook_secret
        ;;
    trigger-checkout-completed)
        exec stripe --api-key "${STRIPE_SECRET}" trigger checkout.session.completed
        ;;
    *)
        echo "Unknown action: ${ACTION}"
        echo "Usage: ${0} {listen|listen-sync-secret|sync-webhook-secret|trigger-checkout-completed}"
        exit 1
        ;;
esac
