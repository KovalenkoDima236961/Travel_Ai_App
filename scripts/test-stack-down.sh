#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${TEST_ENV_FILE:-${PROJECT_ROOT}/infra/.env.test.example}"
COMPOSE_PROJECT_NAME="${TEST_COMPOSE_PROJECT_NAME:-travel-ai-test}"

[[ "${COMPOSE_PROJECT_NAME}" == travel-ai-test* ]] || {
  echo "Refusing to stop a non-test Compose project: ${COMPOSE_PROJECT_NAME}" >&2
  exit 1
}

docker compose -p "${COMPOSE_PROJECT_NAME}" -f "${PROJECT_ROOT}/infra/docker-compose.yml" --env-file "${ENV_FILE}" --profile test down --remove-orphans
