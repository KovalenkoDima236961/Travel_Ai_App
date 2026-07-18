#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${TEST_ENV_FILE:-${PROJECT_ROOT}/infra/.env.test.example}"
COMPOSE_PROJECT_NAME="${TEST_COMPOSE_PROJECT_NAME:-travel-ai-test}"

[[ -f "${ENV_FILE}" ]] || { echo "Test environment not found: ${ENV_FILE}" >&2; exit 1; }
"${SCRIPT_DIR}/validate-env.sh" test --env-file "${ENV_FILE}"

compose=(docker compose -p "${COMPOSE_PROJECT_NAME}" -f "${PROJECT_ROOT}/infra/docker-compose.yml" --env-file "${ENV_FILE}" --profile test)
"${compose[@]}" config --quiet

if [[ "${1:-}" == "--dependencies" ]]; then
  "${compose[@]}" up -d --build --wait postgres rabbitmq ai-planning-service-test
else
  "${compose[@]}" up -d --build --wait postgres rabbitmq ai-planning-service-test
  "${compose[@]}" run --rm --build migration-runner
  "${compose[@]}" up -d --build --wait \
    auth-service \
    user-service \
    external-integrations-service \
    notification-service \
    trip-service \
    worker-service \
    web-app
fi

echo "Test stack is ready (project=${COMPOSE_PROJECT_NAME}, env=${ENV_FILE})."
