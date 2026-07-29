#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env.alpha"
REPORT_DIR="${PROJECT_ROOT}/dist/alpha-readiness"
REPORT="${REPORT_DIR}/rollback-rehearsal-report.md"
PROJECT_NAME="${ALPHA_COMPOSE_PROJECT_NAME:-travel-ai-alpha-rollback}"
PREVIOUS_IMAGE_TAG="${PREVIOUS_IMAGE_TAG:-}"

usage() {
  cat <<'USAGE'
Usage: scripts/release/rehearse-alpha-rollback.sh [--env-file PATH] [--previous-image-tag TAG]

Rehearses rollback evidence collection. It never runs down migrations and it
marks database rollback blocked when schema compatibility cannot be proven.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-file) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; ENV_FILE="$2"; shift 2 ;;
    --previous-image-tag) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; PREVIOUS_IMAGE_TAG="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
done

mkdir -p "${REPORT_DIR}"
: > "${REPORT}"

record() {
  printf '\n## %s\n\n' "$1" >> "${REPORT}"
  echo "==> $1"
}

run() {
  local label="$1"
  shift
  record "${label}"
  if "$@" >> "${REPORT}" 2>&1; then
    echo "PASS ${label}" | tee -a "${REPORT}"
  else
    echo "FAIL ${label}" | tee -a "${REPORT}"
    exit 1
  fi
}

load_env_file() {
  local line
  while IFS= read -r line || [[ -n "${line}" ]]; do
    case "${line}" in
      ""|\#*) continue ;;
      *=*) export "${line}" ;;
    esac
  done < "$1"
}

{
  echo "# Alpha Rollback Rehearsal"
  echo
  echo "- Generated: $(date -u +%FT%TZ)"
  echo "- Env file: ${ENV_FILE#${PROJECT_ROOT}/}"
  echo "- Previous image tag: ${PREVIOUS_IMAGE_TAG:-not supplied}"
  echo "- Policy: no automatic DB down migration"
} >> "${REPORT}"

[[ -f "${ENV_FILE}" ]] || { echo "Environment file not found: ${ENV_FILE}" >&2; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "docker is required." >&2; exit 1; }
load_env_file "${ENV_FILE}"
unset GIT_SHA SHORT_SHA BUILD_TIME IMAGE_TAG_VERSION IMAGE_TAG_SHA
# shellcheck source=version-info.sh
source "${SCRIPT_DIR}/version-info.sh"
export APP_VERSION="${VERSION}" GIT_SHA="${GIT_SHA}" BUILD_TIME="${BUILD_TIME}" IMAGE_TAG="${IMAGE_TAG_SHA}"

run "Validate alpha env" "${PROJECT_ROOT}/scripts/validate-env.sh" alpha --env-file "${ENV_FILE}"
run "Create pre-rollback backup" "${PROJECT_ROOT}/scripts/backup-postgres.sh" --env-file "${ENV_FILE}" --output "${REPORT_DIR}/rollback-backups"
backup_dir="$(find "${REPORT_DIR}/rollback-backups" -maxdepth 1 -type d -name 'postgres-*' | sort | tail -1)"
[[ -n "${backup_dir}" ]] || { echo "No backup directory produced." >&2; exit 1; }
run "Verify backup readability" "${PROJECT_ROOT}/scripts/verify-backup.sh" "${backup_dir}" --env-file "${ENV_FILE}"

compose=(docker compose -p "${PROJECT_NAME}" -f "${PROJECT_ROOT}/infra/docker-compose.prod.yml" -f "${PROJECT_ROOT}/infra/docker-compose.alpha.yml" --env-file "${ENV_FILE}")
run "Deploy current release candidate" env TRAVEL_AI_ENV_FILE="${ENV_FILE}" "${compose[@]}" up -d --no-build --wait

export AUTH_SERVICE_URL="http://127.0.0.1:${AUTH_SERVICE_PORT:-28082}"
export USER_SERVICE_URL="http://127.0.0.1:${USER_SERVICE_PORT:-28083}"
export TRIP_SERVICE_URL="http://127.0.0.1:${TRIP_SERVICE_PORT:-28080}"
export NOTIFICATION_SERVICE_URL="http://127.0.0.1:${NOTIFICATION_SERVICE_PORT:-28086}"
export EXTERNAL_INTEGRATIONS_SERVICE_URL="http://127.0.0.1:${EXTERNAL_INTEGRATIONS_SERVICE_PORT:-28084}"
export WORKER_SERVICE_URL="http://127.0.0.1:${WORKER_SERVICE_PORT:-28090}"
export AI_PLANNING_SERVICE_URL="http://127.0.0.1:${AI_PLANNING_SERVICE_PORT:-28000}"
export WEB_APP_URL="http://127.0.0.1:${WEB_APP_PORT:-23000}"

run "Create disposable alpha data" "${SCRIPT_DIR}/alpha-smoke-test.sh" --mock-openai --env-file "${ENV_FILE}" --base-url "${WEB_APP_URL}" --report-path "${REPORT_DIR}/rollback-alpha-smoke-report.json"
run "Migration status after candidate" "${PROJECT_ROOT}/scripts/migration-status.sh" --env-file "${ENV_FILE}"

record "Rollback image switch"
if [[ -z "${PREVIOUS_IMAGE_TAG}" ]]; then
  {
    echo "ROLLBACK BLOCKED: PREVIOUS_IMAGE_TAG was not supplied."
    echo
    echo "Service rollback compatibility with the current schema cannot be proven automatically."
    echo "Use a forward fix or rerun with --previous-image-tag after reviewing migration safety."
  } | tee -a "${REPORT}"
else
  IMAGE_TAG="${PREVIOUS_IMAGE_TAG}" run "Start prior service images" env TRAVEL_AI_ENV_FILE="${ENV_FILE}" IMAGE_TAG="${PREVIOUS_IMAGE_TAG}" "${compose[@]}" up -d --no-build --wait
  run "Smoke prior images on current schema" "${SCRIPT_DIR}/alpha-smoke-test.sh" --mock-openai --env-file "${ENV_FILE}" --base-url "${WEB_APP_URL}" --report-path "${REPORT_DIR}/rollback-prior-image-smoke-report.json"
fi

record "Backup restore validation"
{
  echo "Backup created: ${backup_dir#${PROJECT_ROOT}/}"
  echo "Restore into a separate validation database is intentionally not run against APP_ENV=staging by this script."
  echo "Use scripts/verify-backup.sh --restore-test only with a local/test env file."
} | tee -a "${REPORT}"

record "Result"
echo "Rollback rehearsal evidence written. A go/no-go reviewer must sign the rollback decision separately." | tee -a "${REPORT}"
echo "Report: ${REPORT#${PROJECT_ROOT}/}"
