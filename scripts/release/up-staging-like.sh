#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env.staging"
PROJECT_NAME="${RELEASE_COMPOSE_PROJECT_NAME:-travel-ai-release}"

usage() {
  echo "Usage: scripts/release/up-staging-like.sh [--env-file PATH] [--skip-smoke]" >&2
}

skip_smoke=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-file) [[ $# -ge 2 ]] || { usage; exit 2; }; ENV_FILE="$2"; shift 2 ;;
    --skip-smoke) skip_smoke=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) usage; exit 2 ;;
  esac
done

[[ -f "${ENV_FILE}" ]] || { echo "Environment file not found: ${ENV_FILE}. Copy infra/.env.staging.example first." >&2; exit 1; }
"${PROJECT_ROOT}/scripts/validate-env.sh" staging --env-file "${ENV_FILE}"
# shellcheck source=version-info.sh
source "${SCRIPT_DIR}/version-info.sh"
release_git_sha="${GIT_SHA}"
release_build_time="${BUILD_TIME}"
command -v docker >/dev/null 2>&1 || { echo "docker is required." >&2; exit 1; }

set -a
# shellcheck disable=SC1090
source "${ENV_FILE}"
set +a
export APP_VERSION="${VERSION}" GIT_SHA="${release_git_sha}" BUILD_TIME="${release_build_time}" IMAGE_TAG="${IMAGE_TAG_SHA}"
export TRAVEL_AI_ENV_FILE="${ENV_FILE}"

compose=(docker compose -p "${PROJECT_NAME}" -f "${PROJECT_ROOT}/infra/docker-compose.prod.yml" -f "${PROJECT_ROOT}/infra/docker-compose.release.yml" --env-file "${ENV_FILE}")
"${compose[@]}" config --quiet
"${compose[@]}" up -d --no-build --wait

export AUTH_SERVICE_URL="http://127.0.0.1:${AUTH_SERVICE_PORT:-28082}"
export USER_SERVICE_URL="http://127.0.0.1:${USER_SERVICE_PORT:-28083}"
export TRIP_SERVICE_URL="http://127.0.0.1:${TRIP_SERVICE_PORT:-28080}"
export NOTIFICATION_SERVICE_URL="http://127.0.0.1:${NOTIFICATION_SERVICE_PORT:-28086}"
export EXTERNAL_INTEGRATIONS_SERVICE_URL="http://127.0.0.1:${EXTERNAL_INTEGRATIONS_SERVICE_PORT:-28084}"
export WORKER_SERVICE_URL="http://127.0.0.1:${WORKER_SERVICE_PORT:-28090}"
export AI_PLANNING_SERVICE_URL="http://127.0.0.1:${AI_PLANNING_SERVICE_PORT:-28000}"
export WEB_APP_URL="http://127.0.0.1:${WEB_APP_PORT:-23000}"

if [[ "${skip_smoke}" == false ]]; then
  SMOKE_ENV_FILE="${ENV_FILE}" SMOKE_ENV_TARGET=staging SMOKE_EXPECT_OBSERVABILITY=false "${SCRIPT_DIR}/smoke-release.sh" staging
fi

echo "Staging-like stack is ready: ${WEB_APP_URL}"
echo "Use scripts/release/down-staging-like.sh to stop it without removing volumes."
