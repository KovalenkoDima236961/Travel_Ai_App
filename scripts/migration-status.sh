#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env"

usage() {
  echo "Usage: scripts/migration-status.sh [local|staging|production] [--env-file PATH]" >&2
}

TARGET=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    local|staging|production) TARGET="$1"; shift ;;
    --env-file) [[ $# -ge 2 ]] || { usage; exit 2; }; ENV_FILE="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) usage; exit 2 ;;
  esac
done
[[ -f "${ENV_FILE}" ]] || { echo "Environment file not found: ${ENV_FILE}" >&2; exit 1; }
command -v psql >/dev/null 2>&1 || { echo "psql is required to inspect migration status." >&2; exit 1; }

while IFS= read -r line || [[ -n "${line}" ]]; do
  case "${line}" in ""|\#*) continue ;; *=*) export "${line}" ;; esac
done < "${ENV_FILE}"

APP_ENV="${APP_ENV:-local}"
[[ -z "${TARGET}" || "${TARGET}" == "${APP_ENV}" ]] || { echo "APP_ENV is ${APP_ENV}, not ${TARGET}." >&2; exit 1; }
export PGHOST="${POSTGRES_HOST:-localhost}" PGPORT="${POSTGRES_PORT:-5432}" PGUSER="${POSTGRES_USER:-postgres}" PGPASSWORD="${POSTGRES_PASSWORD:-}"
[[ -n "${PGPASSWORD}" ]] || { echo "POSTGRES_PASSWORD is required." >&2; exit 1; }

status=0
inspect() {
  local service="$1" database="$2" migration_dir="$3" latest current
  latest="$(find "${PROJECT_ROOT}/${migration_dir}" -maxdepth 1 -type f -name '*_up.sql' -exec basename {} \; | sed -E 's/^([0-9]+).*/\1/' | sort -n | tail -n 1)"
  latest="${latest:-0}"
  if ! psql -d "${database}" -tAc "SELECT to_regclass('public.schema_migrations') IS NOT NULL" | grep -qx t; then
    echo "FAIL ${service}: schema_migrations is missing in ${database}"
    status=1
    return
  fi
  current="$(psql -d "${database}" -tAc "SELECT version::text || ':' || dirty::text FROM schema_migrations ORDER BY version DESC LIMIT 1")"
  if [[ -z "${current}" ]]; then
    echo "FAIL ${service}: no migration version recorded"
    status=1
    return
  fi
  if [[ "${current}" == *:true ]]; then
    echo "FAIL ${service}: dirty migration state (${current})"
    status=1
    return
  fi
  if (( 10#${current%%:*} < 10#${latest} )); then
    echo "PENDING ${service}: current ${current%%:*}, latest ${latest}"
    status=1
    return
  fi
  echo "PASS ${service}: version ${current%%:*}"
}

inspect auth-service "${AUTH_POSTGRES_DB:-auth_service}" services/auth-service/migrations
inspect user-service "${USER_POSTGRES_DB:-user_service}" services/user-service/migrations
inspect trip-service "${POSTGRES_DB:-trip_service}" services/trip-service/migrations
inspect notification-service "${NOTIFICATION_POSTGRES_DB:-notification_service}" services/notification-service/migrations
inspect external-integrations-service "${EXTERNAL_INTEGRATIONS_POSTGRES_DB:-external_integrations_service}" services/external-integrations-service/migrations
exit "${status}"
