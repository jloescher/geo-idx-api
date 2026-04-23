#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="${ROOT_DIR}/docker-compose.dev.yml"

if ! command -v docker >/dev/null 2>&1; then
    echo "docker is required but not installed."
    exit 1
fi

ACTION="${1:-up-watch}"

case "${ACTION}" in
    up)
        exec docker compose -f "${COMPOSE_FILE}" up --build
        ;;
    up-watch)
        exec docker compose -f "${COMPOSE_FILE}" up --build --watch
        ;;
    down)
        exec docker compose -f "${COMPOSE_FILE}" down
        ;;
    logs)
        exec docker compose -f "${COMPOSE_FILE}" logs -f
        ;;
    rebuild)
        exec docker compose -f "${COMPOSE_FILE}" build --no-cache
        ;;
    ps)
        exec docker compose -f "${COMPOSE_FILE}" ps
        ;;
    *)
        echo "Unknown action: ${ACTION}"
        echo "Usage: ${0} {up|up-watch|down|logs|rebuild|ps}"
        exit 1
        ;;
esac
