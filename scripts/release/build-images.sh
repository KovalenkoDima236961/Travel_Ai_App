#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
PUSH=false
SERVICE=""

usage() {
  cat <<'USAGE'
Usage: scripts/release/build-images.sh [--service NAME] [--push]

Builds versioned images for every deployable service. Tags are
<registry>/<service>:<version> and <registry>/<service>:<version>-<shortSha>.
Set REGISTRY to a registry/repository prefix; without it, local image names are
used. --push is explicit and never reads or prints credentials.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --service) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; SERVICE="$2"; shift 2 ;;
    --push) PUSH=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

# shellcheck source=version-info.sh
source "${SCRIPT_DIR}/version-info.sh"
command -v docker >/dev/null 2>&1 || { echo "docker is required." >&2; exit 1; }

services=(auth-service user-service trip-service notification-service worker-service external-integrations-service ai-planning-service web-app migration-runner)
if [[ -n "${SERVICE}" ]]; then
  found=false
  for candidate in "${services[@]}"; do [[ "${candidate}" == "${SERVICE}" ]] && found=true; done
  "${found}" || { echo "Unknown service: ${SERVICE}" >&2; exit 2; }
  services=("${SERVICE}")
fi

image_name() {
  if [[ -n "${REGISTRY:-}" ]]; then
    printf '%s/%s' "${REGISTRY%/}" "$1"
  else
    printf '%s' "$1"
  fi
}

build_args=(--build-arg "APP_VERSION=${VERSION}" --build-arg "GIT_SHA=${GIT_SHA}" --build-arg "BUILD_TIME=${BUILD_TIME}")
web_args=(
  "${build_args[@]}"
  --build-arg "NEXT_PUBLIC_APP_ENV=${NEXT_PUBLIC_APP_ENV:-${APP_ENV:-local}}"
  --build-arg "NEXT_PUBLIC_APP_VERSION=${VERSION}"
  --build-arg "NEXT_PUBLIC_GIT_SHA=${GIT_SHA}"
  --build-arg "NEXT_PUBLIC_BUILD_TIME=${BUILD_TIME}"
  --build-arg "NEXT_PUBLIC_AUTH_SERVICE_URL=${NEXT_PUBLIC_AUTH_SERVICE_URL:-http://localhost:8082}"
  --build-arg "NEXT_PUBLIC_TRIP_SERVICE_URL=${NEXT_PUBLIC_TRIP_SERVICE_URL:-http://localhost:8080}"
  --build-arg "NEXT_PUBLIC_USER_SERVICE_URL=${NEXT_PUBLIC_USER_SERVICE_URL:-http://localhost:8083}"
  --build-arg "NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL=${NEXT_PUBLIC_EXTERNAL_INTEGRATIONS_SERVICE_URL:-http://localhost:8084}"
  --build-arg "NEXT_PUBLIC_NOTIFICATION_SERVICE_URL=${NEXT_PUBLIC_NOTIFICATION_SERVICE_URL:-http://localhost:8086}"
  --build-arg "NEXT_PUBLIC_WORKER_SERVICE_URL=${NEXT_PUBLIC_WORKER_SERVICE_URL:-http://localhost:8090}"
)

build_one() {
  local service="$1" image
  image="$(image_name "${service}")"
  local tags=(-t "${image}:${IMAGE_TAG_VERSION}" -t "${image}:${IMAGE_TAG_SHA}")
  case "${service}" in
    auth-service|user-service|trip-service|notification-service|external-integrations-service)
      docker build "${build_args[@]}" "${tags[@]}" "${PROJECT_ROOT}/services/${service}"
      ;;
    worker-service)
      docker build "${build_args[@]}" "${tags[@]}" -f "${PROJECT_ROOT}/services/worker-service/Dockerfile" "${PROJECT_ROOT}"
      ;;
    ai-planning-service)
      docker build "${build_args[@]}" "${tags[@]}" "${PROJECT_ROOT}/services/ai-planning-service"
      ;;
    web-app)
      docker build "${web_args[@]}" "${tags[@]}" "${PROJECT_ROOT}/apps/web"
      ;;
    migration-runner)
      docker build "${tags[@]}" -f "${PROJECT_ROOT}/infra/Dockerfile.migrations" "${PROJECT_ROOT}"
      ;;
  esac
  printf '%s:%s\n%s:%s\n' "${image}" "${IMAGE_TAG_VERSION}" "${image}" "${IMAGE_TAG_SHA}" >> "${image_list}"
  if [[ "${PUSH}" == true ]]; then
    docker push "${image}:${IMAGE_TAG_VERSION}"
    docker push "${image}:${IMAGE_TAG_SHA}"
  fi
}

mkdir -p "${PROJECT_ROOT}/dist/release-images"
image_list="${PROJECT_ROOT}/dist/release-images/${VERSION}.txt"
: > "${image_list}"
for service in "${services[@]}"; do build_one "${service}"; done

echo "Built release images listed in ${image_list#"${PROJECT_ROOT}/"}."
[[ "${PUSH}" == true ]] && echo "Pushed version and SHA tags." || echo "Images were not pushed (pass --push to publish)."
