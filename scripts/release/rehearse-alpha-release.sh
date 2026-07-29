#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env.alpha"
PROJECT_NAME="${ALPHA_COMPOSE_PROJECT_NAME:-travel-ai-alpha}"
REPORT_DIR="${PROJECT_ROOT}/dist/alpha-readiness"
MODE="mock"
SKIP_E2E=false
SKIP_SECURITY=false
SKIP_BUILD=false

usage() {
  cat <<'USAGE'
Usage: scripts/release/rehearse-alpha-release.sh [--env-file PATH] [--mock-openai|--real-openai] [--skip-e2e] [--skip-security] [--skip-build]

Validates the closed-alpha configuration, builds or uses release images, starts a
production-like Compose stack with isolated alpha volumes, runs migrations,
smoke, alpha Playwright, contracts, security scans, Prometheus target checks,
backup verification, and writes a readiness report.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-file) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; ENV_FILE="$2"; shift 2 ;;
    --mock-openai) MODE="mock"; shift ;;
    --real-openai) MODE="real"; shift ;;
    --skip-e2e) SKIP_E2E=true; shift ;;
    --skip-security) SKIP_SECURITY=true; shift ;;
    --skip-build) SKIP_BUILD=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
done

mkdir -p "${REPORT_DIR}"
REPORT="${REPORT_DIR}/alpha-release-rehearsal-report.md"
: > "${REPORT}"

log_step() {
  printf '\n## %s\n\n' "$1" >> "${REPORT}"
  echo "==> $1"
}

run_and_record() {
  local label="$1"
  shift
  log_step "${label}"
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

[[ -f "${ENV_FILE}" ]] || { echo "Environment file not found: ${ENV_FILE}" >&2; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "docker is required." >&2; exit 1; }
load_env_file "${ENV_FILE}"
unset GIT_SHA SHORT_SHA BUILD_TIME IMAGE_TAG_VERSION IMAGE_TAG_SHA
# shellcheck source=version-info.sh
source "${SCRIPT_DIR}/version-info.sh"
export APP_VERSION="${VERSION}" GIT_SHA="${GIT_SHA}" BUILD_TIME="${BUILD_TIME}" IMAGE_TAG="${IMAGE_TAG_SHA}"

{
  echo "# Alpha Release Rehearsal"
  echo
  echo "- Generated: $(date -u +%FT%TZ)"
  echo "- Env file: ${ENV_FILE#${PROJECT_ROOT}/}"
  echo "- OpenAI mode: ${MODE}"
  echo "- Compose project: ${PROJECT_NAME}"
} >> "${REPORT}"

run_and_record "Validate alpha env" "${PROJECT_ROOT}/scripts/validate-env.sh" alpha --env-file "${ENV_FILE}"
run_and_record "Validate alpha scope" "${SCRIPT_DIR}/validate-alpha-scope.sh" --env-file "${ENV_FILE}"
run_and_record "Validate alpha compose" env TRAVEL_AI_ENV_FILE="${ENV_FILE}" docker compose -p "${PROJECT_NAME}" -f "${PROJECT_ROOT}/infra/docker-compose.prod.yml" -f "${PROJECT_ROOT}/infra/docker-compose.alpha.yml" --env-file "${ENV_FILE}" config --quiet

if [[ "${SKIP_BUILD}" == false ]]; then
  run_and_record "Build release images" env REGISTRY="${IMAGE_REGISTRY:-travel-ai}" "${SCRIPT_DIR}/build-images.sh"
else
  log_step "Build release images"
  echo "SKIP build requested; rehearsal will use existing built images." | tee -a "${REPORT}"
fi

compose=(docker compose -p "${PROJECT_NAME}" -f "${PROJECT_ROOT}/infra/docker-compose.prod.yml" -f "${PROJECT_ROOT}/infra/docker-compose.alpha.yml" --env-file "${ENV_FILE}")
run_and_record "Start alpha stack" env TRAVEL_AI_ENV_FILE="${ENV_FILE}" "${compose[@]}" up -d --no-build --wait

export AUTH_SERVICE_URL="http://127.0.0.1:${AUTH_SERVICE_PORT:-28082}"
export USER_SERVICE_URL="http://127.0.0.1:${USER_SERVICE_PORT:-28083}"
export TRIP_SERVICE_URL="http://127.0.0.1:${TRIP_SERVICE_PORT:-28080}"
export NOTIFICATION_SERVICE_URL="http://127.0.0.1:${NOTIFICATION_SERVICE_PORT:-28086}"
export EXTERNAL_INTEGRATIONS_SERVICE_URL="http://127.0.0.1:${EXTERNAL_INTEGRATIONS_SERVICE_PORT:-28084}"
export WORKER_SERVICE_URL="http://127.0.0.1:${WORKER_SERVICE_PORT:-28090}"
export AI_PLANNING_SERVICE_URL="http://127.0.0.1:${AI_PLANNING_SERVICE_PORT:-28000}"
export WEB_APP_URL="http://127.0.0.1:${WEB_APP_PORT:-23000}"

run_and_record "Migration status" "${PROJECT_ROOT}/scripts/migration-status.sh" --env-file "${ENV_FILE}"
run_and_record "Version consistency" "${SCRIPT_DIR}/check-versions.sh" staging
run_and_record "Alpha smoke" "${SCRIPT_DIR}/alpha-smoke-test.sh" "--${MODE}-openai" --env-file "${ENV_FILE}" --base-url "${WEB_APP_URL}" --report-path "${REPORT_DIR}/alpha-smoke-report.json"
run_and_record "API contract checks" "${PROJECT_ROOT}/scripts/contracts/validate-openapi.sh"
run_and_record "Generated client check" "${PROJECT_ROOT}/scripts/contracts/check-generated.sh"

if [[ "${SKIP_SECURITY}" == false ]]; then
  run_and_record "Security scan" "${PROJECT_ROOT}/scripts/security-scan.sh"
  run_and_record "ZAP baseline" "${PROJECT_ROOT}/scripts/security/zap-baseline.sh" --target "${WEB_APP_URL}"
else
  log_step "Security scan"
  echo "SKIP security requested." | tee -a "${REPORT}"
fi

if [[ "${SKIP_E2E}" == false ]]; then
  run_and_record "Playwright alpha suite" env PLAYWRIGHT_BASE_URL="${WEB_APP_URL}" E2E_AUTH_URL="${AUTH_SERVICE_URL}" E2E_TRIP_URL="${TRIP_SERVICE_URL}" E2E_NOTIFICATION_URL="${NOTIFICATION_SERVICE_URL}" npm --prefix "${PROJECT_ROOT}/apps/web" run test:e2e:alpha
else
  log_step "Playwright alpha suite"
  echo "SKIP E2E requested." | tee -a "${REPORT}"
fi

if [[ -n "${PROMETHEUS_URL:-}" ]]; then
  run_and_record "Prometheus target check" env SMOKE_CHECK_PROMETHEUS_TARGETS=true PROMETHEUS_URL="${PROMETHEUS_URL}" "${PROJECT_ROOT}/scripts/smoke-test.sh" --core
fi

BACKUP_OUTPUT="${REPORT_DIR}/backups"
run_and_record "Create alpha backup" "${PROJECT_ROOT}/scripts/backup-postgres.sh" --env-file "${ENV_FILE}" --output "${BACKUP_OUTPUT}"
latest_backup="$(find "${BACKUP_OUTPUT}" -maxdepth 1 -type d -name 'postgres-*' | sort | tail -1)"
if [[ -n "${latest_backup}" ]]; then
  run_and_record "Verify alpha backup" "${PROJECT_ROOT}/scripts/verify-backup.sh" "${latest_backup}" --env-file "${ENV_FILE}"
else
  echo "No backup directory produced." | tee -a "${REPORT}"
  exit 1
fi

log_step "Result"
echo "Alpha rehearsal completed. Review smoke, security, migration, backup, and E2E artifacts before go/no-go." | tee -a "${REPORT}"
echo "Report: ${REPORT#${PROJECT_ROOT}/}"
