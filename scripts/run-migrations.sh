#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env"
SERVICE="all"
TARGET=""

usage() {
  cat <<'USAGE'
Usage: scripts/run-migrations.sh [SERVICE] [local|staging|production] [--env-file PATH]

SERVICE is one of: auth-service, user-service, trip-service,
notification-service, external-integrations-service, or all (default).

Runs only up migrations and stops on the first failure. When invoked by the
Compose migration-runner, set MIGRATIONS_SKIP_ENV_FILE=true; otherwise the
script reads infra/.env by default.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    auth-service|user-service|trip-service|notification-service|external-integrations-service|all)
      [[ "${SERVICE}" == all ]] || { echo "Only one service may be selected." >&2; exit 2; }
      SERVICE="$1"
      shift
      ;;
    local|staging|production)
      TARGET="$1"
      shift
      ;;
    --env-file)
      [[ $# -ge 2 ]] || { echo "--env-file needs a path" >&2; exit 2; }
      ENV_FILE="$2"
      shift 2
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      # Preserve the previous `run-migrations.sh infra/.env` form.
      [[ -f "$1" && "${SERVICE}" == all && -z "${TARGET}" ]] || { echo "Unknown argument: $1" >&2; usage >&2; exit 2; }
      ENV_FILE="$1"
      shift
      ;;
  esac
done

load_env_file() {
  local line
  while IFS= read -r line || [[ -n "${line}" ]]; do
    case "${line}" in ""|\#*) continue ;; *=*) export "${line}" ;; esac
  done < "$1"
}

if [[ "${MIGRATIONS_SKIP_ENV_FILE:-false}" != true ]]; then
  [[ -f "${ENV_FILE}" ]] || { echo "Environment file not found: ${ENV_FILE}" >&2; exit 1; }
  load_env_file "${ENV_FILE}"
fi

APP_ENV="${APP_ENV:-local}"
case "${APP_ENV}" in local|development|test|staging|production) ;; *) echo "Invalid APP_ENV: ${APP_ENV}" >&2; exit 1 ;; esac
if [[ -n "${TARGET}" && "${TARGET}" != "${APP_ENV}" ]]; then
  echo "APP_ENV is ${APP_ENV}, but ${TARGET} migrations were requested." >&2
  exit 1
fi

for var in POSTGRES_HOST POSTGRES_PORT POSTGRES_USER POSTGRES_PASSWORD; do
  [[ -n "${!var:-}" ]] || { echo "${var} is required for migrations." >&2; exit 1; }
done

run_service_migration() {
  local service="$1" database="$2" binary="$3" service_dir="$4"
  echo "==> Applying ${service} migrations to ${database} (${APP_ENV})"
  export POSTGRES_DB="${database}"
  export POSTGRES_MIN_CONNS="${POSTGRES_MIN_CONNS:-1}"
  export POSTGRES_MAX_CONNS="${POSTGRES_MAX_CONNS:-5}"

  if command -v "${binary}" >/dev/null 2>&1; then
    export POSTGRES_MIG_PATH="/app/migrations/${service}"
    "${binary}"
  else
    export POSTGRES_MIG_PATH="${PROJECT_ROOT}/${service_dir}/migrations"
    (cd "${PROJECT_ROOT}/${service_dir}" && go run ./cmd/migrate)
  fi
  echo "PASS ${service} migrations"
}

run_selected() {
  local requested="$1" service="$2" database="$3" binary="$4" service_dir="$5"
  [[ "${requested}" == all || "${requested}" == "${service}" ]] || return 0
  run_service_migration "${service}" "${database}" "${binary}" "${service_dir}"
}

run_selected "${SERVICE}" auth-service "${AUTH_POSTGRES_DB:-auth_service}" auth-service-migrate services/auth-service
run_selected "${SERVICE}" user-service "${USER_POSTGRES_DB:-user_service}" user-service-migrate services/user-service
run_selected "${SERVICE}" trip-service "${POSTGRES_DB:-trip_service}" trip-service-migrate services/trip-service
run_selected "${SERVICE}" notification-service "${NOTIFICATION_POSTGRES_DB:-notification_service}" notification-service-migrate services/notification-service
run_selected "${SERVICE}" external-integrations-service "${EXTERNAL_INTEGRATIONS_POSTGRES_DB:-external_integrations_service}" external-integrations-service-migrate services/external-integrations-service

echo "All requested migrations completed."
