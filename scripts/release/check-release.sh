#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
MODE="${1:-local}"
[[ $# -gt 0 ]] && shift
ENV_FILE=""
skip_e2e=false
skip_security=false
skip_docker=false

usage() {
  cat <<'USAGE'
Usage: scripts/release/check-release.sh [local|staging|ci] [--env-file PATH] [--skip-e2e] [--skip-security] [--skip-docker-build]

Local checks may explicitly skip slow gates. CI mode rejects every skip flag.
USAGE
}

case "${MODE}" in local|staging|ci) ;; *) usage >&2; exit 2 ;; esac
while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-file) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; ENV_FILE="$2"; shift 2 ;;
    --skip-e2e) skip_e2e=true; shift ;;
    --skip-security) skip_security=true; shift ;;
    --skip-docker-build) skip_docker=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
done
if [[ "${MODE}" == ci && ( "${skip_e2e}" == true || "${skip_security}" == true || "${skip_docker}" == true ) ]]; then
  echo "CI release checks do not allow skipped gates." >&2
  exit 2
fi
case "${MODE}" in
  local) ENV_FILE="${ENV_FILE:-${PROJECT_ROOT}/infra/.env}"; env_target=local ;;
  staging) ENV_FILE="${ENV_FILE:-${PROJECT_ROOT}/infra/.env.staging}"; env_target=staging ;;
  ci) ENV_FILE="${ENV_FILE:-${PROJECT_ROOT}/infra/.env.ci.example}"; env_target=test ;;
esac

[[ -f "${ENV_FILE}" ]] || { echo "Environment file not found: ${ENV_FILE}" >&2; exit 1; }
"${PROJECT_ROOT}/scripts/validate-env.sh" "${env_target}" --env-file "${ENV_FILE}"
if [[ "${MODE}" == staging ]]; then
  TRAVEL_AI_ENV_FILE="${ENV_FILE}" docker compose -f "${PROJECT_ROOT}/infra/docker-compose.prod.yml" -f "${PROJECT_ROOT}/infra/docker-compose.release.yml" --env-file "${ENV_FILE}" config --quiet
else
  COMPOSE_ENV_FILE="${ENV_FILE}" docker compose -f "${PROJECT_ROOT}/infra/docker-compose.yml" --env-file "${ENV_FILE}" config --quiet
fi
"${SCRIPT_DIR}/check-changelog.sh"
"${PROJECT_ROOT}/scripts/contracts/validate-openapi.sh"
"${PROJECT_ROOT}/scripts/contracts/check-generated.sh"
"${PROJECT_ROOT}/scripts/test-frontend.sh"
"${PROJECT_ROOT}/scripts/test-go.sh"
"${PROJECT_ROOT}/scripts/test-python.sh"
"${PROJECT_ROOT}/scripts/test-backend-integration.sh"
"${SCRIPT_DIR}/check-migrations.sh" "${MODE}" --env-file "${ENV_FILE}"

if [[ "${skip_security}" == false ]]; then "${PROJECT_ROOT}/scripts/security-scan.sh"; else echo "SKIP security (explicit local override)"; fi
if [[ "${skip_docker}" == false ]]; then "${SCRIPT_DIR}/build-images.sh"; else echo "SKIP Docker build (explicit local override)"; fi
if [[ "${skip_e2e}" == false ]]; then
  "${PROJECT_ROOT}/scripts/test-frontend-e2e.sh"
else
  echo "SKIP E2E (explicit local override)"
fi

if [[ "${MODE}" == staging ]]; then "${SCRIPT_DIR}/smoke-release.sh" staging; fi
echo "PASS release checks (${MODE})."
