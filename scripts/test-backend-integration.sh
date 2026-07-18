#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${TEST_ENV_FILE:-${PROJECT_ROOT}/infra/.env.test.example}"
COMPOSE_PROJECT_NAME="${TEST_COMPOSE_PROJECT_NAME:-travel-ai-test}"
MANAGE_STACK="${TEST_STACK_MANAGED:-true}"

load_env_file() {
  local line
  while IFS= read -r line || [[ -n "${line}" ]]; do
    case "${line}" in ""|\#*) continue ;; *=*) export "${line}" ;; esac
  done < "$1"
}

load_env_file "${ENV_FILE}"
[[ "${APP_ENV:-}" == "test" ]] || { echo "Refusing integration cleanup outside APP_ENV=test." >&2; exit 1; }
[[ "${COMPOSE_PROJECT_NAME}" == travel-ai-test* ]] || { echo "Refusing non-test Compose project." >&2; exit 1; }

cleanup() {
  local status=$?
  trap - EXIT
  if [[ "${MANAGE_STACK}" == "true" && "${KEEP_TEST_STACK:-false}" != "true" ]]; then
    "${SCRIPT_DIR}/test-stack-down.sh" || true
  fi
  exit "${status}"
}
trap cleanup EXIT

compose=(docker compose -p "${COMPOSE_PROJECT_NAME}" -f "${PROJECT_ROOT}/infra/docker-compose.yml" --env-file "${ENV_FILE}" --profile test)
if [[ "${MANAGE_STACK}" == "true" ]]; then
  "${compose[@]}" up -d --wait postgres
fi
"${compose[@]}" run --rm --build migration-runner

export EIS_TEST_DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@127.0.0.1:${POSTGRES_PUBLISHED_PORT}/external_integrations_service?sslmode=disable"
(cd "${PROJECT_ROOT}/services/external-integrations-service" && go test ./internal/providerlimits -run '^TestPostgresStore' -count=1)

echo "PASS backend Postgres integration tests and fresh migrations"
