#!/usr/bin/env sh
set -eu

usage() {
  cat <<'USAGE'
Usage: scripts/run-migrations.sh [env-file]

Runs all service-owned Postgres migrations with the existing golang-migrate
startup path. The script never runs down/reset migrations.

Environment:
  APP_ENV                  local|staging|production
  POSTGRES_HOST            Postgres host
  POSTGRES_PORT            Postgres port
  POSTGRES_USER            Postgres user
  POSTGRES_PASSWORD        Postgres password
  POSTGRES_MIN_CONNS       Optional, default 1
  POSTGRES_MAX_CONNS       Optional, default 5
USAGE
}

if [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

ENV_FILE="${1:-}"
if [ -n "$ENV_FILE" ]; then
  if [ ! -f "$ENV_FILE" ]; then
    echo "Env file not found: $ENV_FILE" >&2
    exit 1
  fi
  load_env_file() {
    while IFS= read -r line || [ -n "$line" ]; do
      case "$line" in
        ""|\#*) continue ;;
        *=*) export "$line" ;;
      esac
    done < "$1"
  }
  load_env_file "$ENV_FILE"
fi

APP_ENV="${APP_ENV:-local}"
POSTGRES_HOST="${POSTGRES_HOST:-}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"
POSTGRES_MIN_CONNS="${POSTGRES_MIN_CONNS:-1}"
POSTGRES_MAX_CONNS="${POSTGRES_MAX_CONNS:-5}"
TRIP_POSTGRES_DB="${POSTGRES_DB:-trip_service}"

require_var() {
  name="$1"
  eval "value=\${$name:-}"
  if [ -z "$value" ]; then
    echo "$name is required for migrations" >&2
    exit 1
  fi
}

require_var POSTGRES_HOST
require_var POSTGRES_PORT
require_var POSTGRES_USER
require_var POSTGRES_PASSWORD

run_service_migration() {
  service="$1"
  db="$2"
  binary="$3"
  service_dir="$4"

  echo "==> Running ${service} migrations (${db})"
  export APP_ENV
  export POSTGRES_DB="$db"
  export POSTGRES_USER
  export POSTGRES_PASSWORD
  export POSTGRES_HOST
  export POSTGRES_PORT
  export POSTGRES_MIN_CONNS
  export POSTGRES_MAX_CONNS

  if command -v "$binary" >/dev/null 2>&1; then
    export POSTGRES_MIG_PATH="/app/migrations/${service}"
    "$binary"
    return
  fi

  export POSTGRES_MIG_PATH="${service_dir}/migrations"
  (cd "$service_dir" && go run ./cmd/migrate)
}

run_service_migration "auth-service" "auth_service" "auth-service-migrate" "services/auth-service"
run_service_migration "user-service" "user_service" "user-service-migrate" "services/user-service"
run_service_migration "trip-service" "$TRIP_POSTGRES_DB" "trip-service-migrate" "services/trip-service"
run_service_migration "notification-service" "notification_service" "notification-service-migrate" "services/notification-service"
run_service_migration "external-integrations-service" "external_integrations_service" "external-integrations-service-migrate" "services/external-integrations-service"

echo "All migrations applied for APP_ENV=${APP_ENV}"
