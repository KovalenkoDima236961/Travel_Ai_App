#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env.staging"
PROJECT_NAME="${RELEASE_COMPOSE_PROJECT_NAME:-travel-ai-release}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-file) [[ $# -ge 2 ]] || { echo "--env-file needs a path" >&2; exit 2; }; ENV_FILE="$2"; shift 2 ;;
    --help|-h) echo "Usage: scripts/release/down-staging-like.sh [--env-file PATH]"; exit 0 ;;
    *) echo "Unknown option: $1" >&2; exit 2 ;;
  esac
done

[[ -f "${ENV_FILE}" ]] || { echo "Environment file not found: ${ENV_FILE}" >&2; exit 1; }
export TRAVEL_AI_ENV_FILE="${ENV_FILE}"
docker compose -p "${PROJECT_NAME}" -f "${PROJECT_ROOT}/infra/docker-compose.prod.yml" -f "${PROJECT_ROOT}/infra/docker-compose.release.yml" --env-file "${ENV_FILE}" down --remove-orphans
echo "Staging-like stack stopped. Named volumes were preserved."
