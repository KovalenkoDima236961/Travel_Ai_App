#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
ENV_FILE="${PROJECT_ROOT}/infra/.env.alpha"
REPORT_PATH="${PROJECT_ROOT}/dist/alpha-readiness/alpha-smoke-report.json"
MODE="mock"
BASE_URL="${WEB_APP_URL:-http://127.0.0.1:23000}"
SKIP_EXPENSIVE=false

usage() {
  cat <<'USAGE'
Usage: scripts/release/alpha-smoke-test.sh [--mock-openai|--real-openai] [--skip-expensive] [--base-url URL] [--env-file PATH] [--report-path PATH]

Runs closed-alpha smoke checks and writes a machine-readable safe report.
Real OpenAI mode requires ALPHA_REAL_OPENAI_APPROVED=true and strict caps.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mock-openai) MODE="mock"; shift ;;
    --real-openai) MODE="real"; shift ;;
    --skip-expensive) SKIP_EXPENSIVE=true; shift ;;
    --base-url) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; BASE_URL="$2"; shift 2 ;;
    --env-file) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; ENV_FILE="$2"; shift 2 ;;
    --report-path) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; REPORT_PATH="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
done

command -v jq >/dev/null 2>&1 || { echo "jq is required." >&2; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "curl is required." >&2; exit 1; }
[[ -f "${ENV_FILE}" ]] || { echo "Environment file not found: ${ENV_FILE}" >&2; exit 1; }

mkdir -p "$(dirname "${REPORT_PATH}")"
jsonl="$(mktemp)"
last_error="$(mktemp)"
trap 'rm -f "${jsonl}" "${last_error}"' EXIT

load_env_file() {
  local line
  while IFS= read -r line || [[ -n "${line}" ]]; do
    case "${line}" in
      ""|\#*) continue ;;
      *=*) export "${line}" ;;
    esac
  done < "$1"
}

sanitize_error() {
  sed -E \
    -e 's/(Authorization: Bearer )[A-Za-z0-9._~+\/=-]+/\1[redacted]/g' \
    -e 's/(OPENAI_API_KEY=)[^[:space:]]+/\1[redacted]/g' \
    -e 's/(JWT_[A-Z_]*=)[^[:space:]]+/\1[redacted]/g' \
    "$1" | tail -20
}

record() {
  local check="$1" status="$2" duration_ms="$3" service_version="${4:-}" safe_error="${5:-}"
  jq -nc \
    --arg check "${check}" \
    --arg status "${status}" \
    --argjson durationMs "${duration_ms}" \
    --arg serviceVersion "${service_version}" \
    --arg safeError "${safe_error}" \
    --arg timestamp "$(date -u +%FT%TZ)" \
    '{check:$check,status:$status,durationMs:$durationMs,serviceVersion:($serviceVersion|select(length>0)),safeError:($safeError|select(length>0)),timestamp:$timestamp}' >> "${jsonl}"
}

now_ms() {
  printf '%s000' "$(date +%s)"
}

run_check() {
  local label="$1"
  shift
  local start end duration
  start="$(now_ms)"
  if "$@" >"${last_error}" 2>&1; then
    end="$(now_ms)"
    duration=$((end - start))
    record "${label}" "pass" "${duration}"
    echo "PASS ${label}"
  else
    end="$(now_ms)"
    duration=$((end - start))
    record "${label}" "fail" "${duration}" "" "$(sanitize_error "${last_error}")"
    echo "FAIL ${label}" >&2
    cat "${last_error}" >&2
    finalize_report "fail"
    exit 1
  fi
}

endpoint_check() {
  local label="$1" url="$2"
  local start end duration body status version
  start="$(now_ms)"
  body="$(mktemp)"
  if status="$(curl -sS -o "${body}" -w "%{http_code}" --max-time 8 "${url}")" && [[ "${status}" =~ ^2 ]]; then
    end="$(now_ms)"
    duration=$((end - start))
    version="$(jq -r '.version // .status // empty' "${body}" 2>/dev/null || true)"
    record "${label}" "pass" "${duration}" "${version}"
    echo "PASS ${label}"
  else
    end="$(now_ms)"
    duration=$((end - start))
    record "${label}" "fail" "${duration}" "" "HTTP ${status:-curl_failed}"
    echo "FAIL ${label}: HTTP ${status:-curl_failed}" >&2
    rm -f "${body}"
    finalize_report "fail"
    exit 1
  fi
  rm -f "${body}"
}

finalize_report() {
  local overall="$1"
  jq -s --arg overall "${overall}" --arg generatedAt "$(date -u +%FT%TZ)" \
    '{overallStatus:$overall,generatedAt:$generatedAt,checks:.}' "${jsonl}" > "${REPORT_PATH}"
  echo "Alpha smoke report: ${REPORT_PATH#${PROJECT_ROOT}/}"
}

if [[ "${MODE}" == "real" ]]; then
  load_env_file "${ENV_FILE}"
  [[ "${ALPHA_REAL_OPENAI_APPROVED:-false}" == "true" ]] || { echo "Refusing real OpenAI smoke without ALPHA_REAL_OPENAI_APPROVED=true." >&2; exit 2; }
  [[ "${ALPHA_REAL_OPENAI_MAX_REQUESTS:-}" =~ ^[1-5]$ ]] || { echo "ALPHA_REAL_OPENAI_MAX_REQUESTS must be 1..5." >&2; exit 2; }
  [[ "${ALPHA_REAL_OPENAI_SPEND_CAP_UAH:-}" =~ ^[0-9]+([.][0-9]+)?$ ]] || { echo "ALPHA_REAL_OPENAI_SPEND_CAP_UAH is required." >&2; exit 2; }
else
  load_env_file "${ENV_FILE}"
fi

export WEB_APP_URL="${BASE_URL}"
export AUTH_SERVICE_URL="${AUTH_SERVICE_URL:-http://127.0.0.1:${AUTH_SERVICE_PORT:-28082}}"
export USER_SERVICE_URL="${USER_SERVICE_URL:-http://127.0.0.1:${USER_SERVICE_PORT:-28083}}"
export TRIP_SERVICE_URL="${TRIP_SERVICE_URL:-http://127.0.0.1:${TRIP_SERVICE_PORT:-28080}}"
export NOTIFICATION_SERVICE_URL="${NOTIFICATION_SERVICE_URL:-http://127.0.0.1:${NOTIFICATION_SERVICE_PORT:-28086}}"
export EXTERNAL_INTEGRATIONS_SERVICE_URL="${EXTERNAL_INTEGRATIONS_SERVICE_URL:-http://127.0.0.1:${EXTERNAL_INTEGRATIONS_SERVICE_PORT:-28084}}"
export WORKER_SERVICE_URL="${WORKER_SERVICE_URL:-http://127.0.0.1:${WORKER_SERVICE_PORT:-28090}}"
export AI_PLANNING_SERVICE_URL="${AI_PLANNING_SERVICE_URL:-http://127.0.0.1:${AI_PLANNING_SERVICE_PORT:-28000}}"

run_check "alpha env validation" "${PROJECT_ROOT}/scripts/validate-env.sh" alpha --env-file "${ENV_FILE}"
run_check "alpha scope validation" "${SCRIPT_DIR}/validate-alpha-scope.sh" --env-file "${ENV_FILE}"
if command -v docker >/dev/null 2>&1; then
  run_check "alpha compose config" env TRAVEL_AI_ENV_FILE="${ENV_FILE}" docker compose -f "${PROJECT_ROOT}/infra/docker-compose.prod.yml" -f "${PROJECT_ROOT}/infra/docker-compose.alpha.yml" --env-file "${ENV_FILE}" config --quiet
else
  record "alpha compose config" "skip" 0 "" "docker not installed"
  echo "SKIP alpha compose config (docker not installed)"
fi

for service in \
  "Auth Service=${AUTH_SERVICE_URL}" \
  "User Service=${USER_SERVICE_URL}" \
  "Trip Service=${TRIP_SERVICE_URL}" \
  "Notification Service=${NOTIFICATION_SERVICE_URL}" \
  "External Integrations Service=${EXTERNAL_INTEGRATIONS_SERVICE_URL}" \
  "Worker Service=${WORKER_SERVICE_URL}" \
  "AI Planning Service=${AI_PLANNING_SERVICE_URL}"; do
  name="${service%%=*}"
  url="${service#*=}"
  endpoint_check "${name} /health" "${url%/}/health"
  endpoint_check "${name} /ready" "${url%/}/ready"
  endpoint_check "${name} /version" "${url%/}/version"
done

endpoint_check "Web App reachable" "${WEB_APP_URL%/}/"
endpoint_check "Web App /api/health" "${WEB_APP_URL%/}/api/health"
endpoint_check "Web App /api/ready" "${WEB_APP_URL%/}/api/ready"
endpoint_check "Web App /api/version" "${WEB_APP_URL%/}/api/version"

smoke_args=(--core)
[[ "${MODE}" == "real" && "${SKIP_EXPENSIVE}" == false ]] && smoke_args=(--ai)
run_check "alpha authenticated core journey" env \
  SMOKE_ENV_FILE="${ENV_FILE}" \
  SMOKE_ENV_TARGET=staging \
  SMOKE_EXPECT_OBSERVABILITY=false \
  SMOKE_EXPECT_OPS_DASHBOARD=true \
  "${PROJECT_ROOT}/scripts/smoke-test.sh" "${smoke_args[@]}"

if [[ "${MODE}" == "real" && "${SKIP_EXPENSIVE}" == false ]]; then
  run_check "limited real OpenAI evaluation approval" "${PROJECT_ROOT}/scripts/ai/run-alpha-provider-evals.sh" --openai --allow-real-openai --max-requests "${ALPHA_REAL_OPENAI_MAX_REQUESTS}"
else
  run_check "mock OpenAI-compatible evaluation" "${PROJECT_ROOT}/scripts/ai/run-alpha-provider-evals.sh" --mock
fi

finalize_report "pass"
