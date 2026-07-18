#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
TARGET="local"
EXPECT_SHA=""

usage() {
  cat <<'USAGE'
Usage: scripts/release/check-versions.sh [local|staging] [--expected-version VERSION] [--expected-sha SHA]

Calls each public /version endpoint and fails when service name, version, or a
known expected SHA does not match. URL variables such as TRIP_SERVICE_URL and
WEB_APP_URL may override the target defaults.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    local|staging) TARGET="$1"; shift ;;
    --expected-version) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; EXPECT_VERSION="$2"; shift 2 ;;
    --expected-sha) [[ $# -ge 2 ]] || { usage >&2; exit 2; }; EXPECT_SHA="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
done

# shellcheck source=version-info.sh
source "${SCRIPT_DIR}/version-info.sh"
EXPECTED_VERSION="${EXPECT_VERSION:-${VERSION}}"
[[ "${TARGET}" == "local" || -n "${RELEASE_BASE_URL:-}" || -n "${TRIP_SERVICE_URL:-}" ]] || {
  echo "Set RELEASE_BASE_URL or service URL variables when checking a remote staging stack." >&2
  exit 2
}
command -v curl >/dev/null 2>&1 || { echo "curl is required." >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "jq is required." >&2; exit 1; }

base="${RELEASE_BASE_URL:-}"
auth_url="${AUTH_SERVICE_URL:-${base:+${base%/}/auth}}"; auth_url="${auth_url:-http://localhost:8082}"
user_url="${USER_SERVICE_URL:-${base:+${base%/}/user}}"; user_url="${user_url:-http://localhost:8083}"
trip_url="${TRIP_SERVICE_URL:-${base:+${base%/}/trip}}"; trip_url="${trip_url:-http://localhost:8080}"
notification_url="${NOTIFICATION_SERVICE_URL:-${base:+${base%/}/notification}}"; notification_url="${notification_url:-http://localhost:8086}"
worker_url="${WORKER_SERVICE_URL:-${base:+${base%/}/worker}}"; worker_url="${worker_url:-http://localhost:8090}"
external_url="${EXTERNAL_INTEGRATIONS_SERVICE_URL:-${base:+${base%/}/external-integrations}}"; external_url="${external_url:-http://localhost:8084}"
ai_url="${AI_PLANNING_SERVICE_URL:-${base:+${base%/}/ai}}"; ai_url="${ai_url:-http://localhost:8000}"
web_url="${WEB_APP_URL:-${base:-http://localhost:3000}}"

services=(
  "auth-service=${auth_url}"
  "user-service=${user_url}"
  "trip-service=${trip_url}"
  "notification-service=${notification_url}"
  "worker-service=${worker_url}"
  "external-integrations-service=${external_url}"
  "ai-planning-service=${ai_url}"
  "web-app=${web_url}"
)

status=0
for entry in "${services[@]}"; do
  service="${entry%%=*}"; url="${entry#*=}"
  if ! body="$(curl --fail --silent --show-error --max-time 5 "${url%/}/version")"; then
    echo "FAIL ${service}: /version unavailable at ${url}" >&2
    status=1
    continue
  fi
  actual_service="$(jq -r '.service // empty' <<<"${body}")"
  actual_version="$(jq -r '.version // empty' <<<"${body}")"
  actual_sha="$(jq -r '.gitSha // empty' <<<"${body}")"
  build_time="$(jq -r '.buildTime // empty' <<<"${body}")"
  if [[ "${actual_service}" != "${service}" || "${actual_version}" != "${EXPECTED_VERSION}" || -z "${build_time}" ]]; then
    echo "FAIL ${service}: expected service=${service} version=${EXPECTED_VERSION}; got service=${actual_service} version=${actual_version}" >&2
    status=1
    continue
  fi
  if [[ -n "${EXPECT_SHA}" && "${actual_sha}" != "${EXPECT_SHA}" ]]; then
    echo "FAIL ${service}: expected gitSha=${EXPECT_SHA}, got ${actual_sha}" >&2
    status=1
    continue
  fi
  printf 'PASS %-30s version=%-14s gitSha=%-12s buildTime=%s\n' "${service}" "${actual_version}" "${actual_sha:0:12}" "${build_time}"
done
exit "${status}"
