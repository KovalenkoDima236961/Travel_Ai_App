#!/usr/bin/env sh
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

IMAGE_REGISTRY="${IMAGE_REGISTRY:-travel-ai}"
if command -v git >/dev/null 2>&1 && git rev-parse --short HEAD >/dev/null 2>&1; then
  IMAGE_TAG="${IMAGE_TAG:-$(git rev-parse --short HEAD)}"
else
  IMAGE_TAG="${IMAGE_TAG:-$(date +%Y%m%d%H%M%S)}"
fi

docker build -t "${IMAGE_REGISTRY}/auth-service:${IMAGE_TAG}" services/auth-service
docker build -t "${IMAGE_REGISTRY}/user-service:${IMAGE_TAG}" services/user-service
docker build -t "${IMAGE_REGISTRY}/trip-service:${IMAGE_TAG}" services/trip-service
docker build -t "${IMAGE_REGISTRY}/notification-service:${IMAGE_TAG}" services/notification-service
docker build -t "${IMAGE_REGISTRY}/external-integrations-service:${IMAGE_TAG}" services/external-integrations-service
docker build -t "${IMAGE_REGISTRY}/worker-service:${IMAGE_TAG}" -f services/worker-service/Dockerfile .
docker build -t "${IMAGE_REGISTRY}/ai-planning-service:${IMAGE_TAG}" services/ai-planning-service
docker build -t "${IMAGE_REGISTRY}/migration-runner:${IMAGE_TAG}" -f infra/Dockerfile.migrations .
docker build \
  --build-arg NEXT_PUBLIC_APP_ENV="${NEXT_PUBLIC_APP_ENV:-${APP_ENV:-production}}" \
  --build-arg NEXT_PUBLIC_AUTH_SERVICE_URL="${NEXT_PUBLIC_AUTH_SERVICE_URL:?NEXT_PUBLIC_AUTH_SERVICE_URL is required}" \
  --build-arg NEXT_PUBLIC_TRIP_SERVICE_URL="${NEXT_PUBLIC_TRIP_SERVICE_URL:?NEXT_PUBLIC_TRIP_SERVICE_URL is required}" \
  --build-arg NEXT_PUBLIC_USER_SERVICE_URL="${NEXT_PUBLIC_USER_SERVICE_URL:?NEXT_PUBLIC_USER_SERVICE_URL is required}" \
  --build-arg NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL="${NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL:?NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL is required}" \
  --build-arg NEXT_PUBLIC_NOTIFICATION_SERVICE_URL="${NEXT_PUBLIC_NOTIFICATION_SERVICE_URL:?NEXT_PUBLIC_NOTIFICATION_SERVICE_URL is required}" \
  --build-arg NEXT_PUBLIC_WORKER_SERVICE_URL="${NEXT_PUBLIC_WORKER_SERVICE_URL:?NEXT_PUBLIC_WORKER_SERVICE_URL is required}" \
  -t "${IMAGE_REGISTRY}/web-app:${IMAGE_TAG}" \
  apps/web

echo "Built production images with tag ${IMAGE_REGISTRY}/*:${IMAGE_TAG}"
