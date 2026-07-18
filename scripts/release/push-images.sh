#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SERVICE=""

usage() {
  echo "Usage: scripts/release/push-images.sh [--service NAME]" >&2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --service) [[ $# -ge 2 ]] || { usage; exit 2; }; SERVICE="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) usage; exit 2 ;;
  esac
done

# shellcheck source=version-info.sh
source "${SCRIPT_DIR}/version-info.sh"
[[ -n "${REGISTRY:-}" ]] || { echo "REGISTRY is required to push release images." >&2; exit 1; }
[[ "${VERSION}" != "dev" && "${GIT_SHA}" != "unknown" ]] || { echo "Refusing to push dev or unknown build metadata." >&2; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "docker is required." >&2; exit 1; }

services=(auth-service user-service trip-service notification-service worker-service external-integrations-service ai-planning-service web-app migration-runner)
if [[ -n "${SERVICE}" ]]; then
  services=("${SERVICE}")
fi
for service in "${services[@]}"; do
  image="${REGISTRY%/}/${service}"
  docker push "${image}:${IMAGE_TAG_VERSION}"
  docker push "${image}:${IMAGE_TAG_SHA}"
  echo "Pushed ${image}:${IMAGE_TAG_VERSION} and ${image}:${IMAGE_TAG_SHA}"
done
