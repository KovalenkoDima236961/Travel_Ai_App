#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
TARGET="ci"
ENV_FILE="${PROJECT_ROOT}/infra/.env.test.example"
PROJECT_NAME="travel-ai-release-migrations"

usage() {
  cat <<'USAGE'
Usage: scripts/release/check-migrations.sh [ci|local|staging] [--env-file PATH]

Applies every migration to a fresh isolated Postgres instance and verifies the
recorded migration status. For staging, pass that environment file; the script
still uses an isolated Compose project and never alters a production database.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    ci|local|staging) TARGET="$1"; shift ;;
    --env-file) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; ENV_FILE="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
done

[[ -f "${ENV_FILE}" ]] || { echo "Environment file not found: ${ENV_FILE}" >&2; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "docker is required." >&2; exit 1; }
export COMPOSE_ENV_FILE="${ENV_FILE}"

cleanup() {
  docker compose -p "${PROJECT_NAME}" -f "${PROJECT_ROOT}/infra/docker-compose.yml" --env-file "${ENV_FILE}" --profile test down --volumes --remove-orphans >/dev/null 2>&1 || true
}
trap cleanup EXIT

docker compose -p "${PROJECT_NAME}" -f "${PROJECT_ROOT}/infra/docker-compose.yml" --env-file "${ENV_FILE}" --profile test up -d --wait postgres
docker compose -p "${PROJECT_NAME}" -f "${PROJECT_ROOT}/infra/docker-compose.yml" --env-file "${ENV_FILE}" --profile test run --rm migration-runner

env_value() {
  local name="$1" default="$2" value
  value="$(grep -E "^${name}=" "${ENV_FILE}" | tail -1 | cut -d= -f2- || true)"
  printf '%s' "${value:-${default}}"
}

postgres_user="$(env_value POSTGRES_USER postgres)"
databases=(
  "$(env_value AUTH_POSTGRES_DB auth_service)"
  "$(env_value USER_POSTGRES_DB user_service)"
  "$(env_value POSTGRES_DB trip_service)"
  "$(env_value NOTIFICATION_POSTGRES_DB notification_service)"
  "$(env_value EXTERNAL_INTEGRATIONS_POSTGRES_DB external_integrations_service)"
)
for database in "${databases[@]}"; do
  status="$(docker compose -p "${PROJECT_NAME}" -f "${PROJECT_ROOT}/infra/docker-compose.yml" --env-file "${ENV_FILE}" --profile test exec -T postgres psql -U "${postgres_user}" -d "${database}" -tAc "SELECT version::text || ':' || dirty::text FROM schema_migrations ORDER BY version DESC LIMIT 1")"
  [[ -n "${status}" && "${status}" != *:true ]] || { echo "FAIL ${database}: missing or dirty migration status (${status:-none})" >&2; exit 1; }
  echo "PASS ${database}: migration ${status}"
done
echo "PASS migrations apply cleanly to a fresh ${TARGET} database."
