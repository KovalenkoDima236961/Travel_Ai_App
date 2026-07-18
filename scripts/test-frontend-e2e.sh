#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${TEST_ENV_FILE:-${PROJECT_ROOT}/infra/.env.test.example}"
COMPOSE_PROJECT_NAME="${TEST_COMPOSE_PROJECT_NAME:-travel-ai-test}"
MANAGE_STACK="${TEST_STACK_MANAGED:-true}"

cleanup() {
  local status=$?
  trap - EXIT
  if [[ ${status} -ne 0 ]]; then
    docker compose -p "${COMPOSE_PROJECT_NAME}" -f "${PROJECT_ROOT}/infra/docker-compose.yml" --env-file "${ENV_FILE}" --profile test logs --no-color --tail=200 || true
  fi
  if [[ "${MANAGE_STACK}" == "true" && "${KEEP_TEST_STACK:-false}" != "true" ]]; then
    "${SCRIPT_DIR}/test-stack-down.sh" || true
  fi
  exit "${status}"
}
trap cleanup EXIT

if [[ "${MANAGE_STACK}" == "true" ]]; then
  "${SCRIPT_DIR}/test-stack-up.sh"
fi

export PLAYWRIGHT_BASE_URL="${PLAYWRIGHT_BASE_URL:-http://127.0.0.1:13000}"
export E2E_AUTH_URL="${E2E_AUTH_URL:-http://127.0.0.1:18082}"
export E2E_TRIP_URL="${E2E_TRIP_URL:-http://127.0.0.1:18080}"
export E2E_RUN_ID="${E2E_RUN_ID:-${GITHUB_RUN_ID:-local-$$}}"

cd "${PROJECT_ROOT}/apps/web"
npm run test:e2e

echo "PASS Playwright critical-flow tests"
