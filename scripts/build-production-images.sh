#!/usr/bin/env bash
set -eu

usage() {
  cat <<'USAGE'
Usage: scripts/build-production-images.sh [env-file]

Builds production images for all services. Public NEXT_PUBLIC_* values are
passed as web build args because Next.js embeds them into browser bundles.
USAGE
}

if [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi

ENV_FILE="${1:-infra/.env.production}"
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

for name in NEXT_PUBLIC_AUTH_SERVICE_URL NEXT_PUBLIC_TRIP_SERVICE_URL NEXT_PUBLIC_USER_SERVICE_URL NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL NEXT_PUBLIC_NOTIFICATION_SERVICE_URL NEXT_PUBLIC_WORKER_SERVICE_URL; do
  [[ -n "${!name:-}" ]] || { echo "${name} is required" >&2; exit 1; }
done

REGISTRY="${IMAGE_REGISTRY:-travel-ai}" "${PROJECT_ROOT}/scripts/release/build-images.sh"
