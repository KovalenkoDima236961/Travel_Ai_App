#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
TARGET="local"

usage() {
  echo "Usage: scripts/release/smoke-release.sh [local|staging] [--include-ai]" >&2
}

include_ai=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    local|staging) TARGET="$1"; shift ;;
    --include-ai) include_ai=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) usage; exit 2 ;;
  esac
done

"${SCRIPT_DIR}/check-versions.sh" "${TARGET}"

# The established smoke suite verifies public /health and /ready endpoints,
# registration/login, trip creation and retrieval, budget, notifications, and
# mock place/route/weather flows. It does not require provider keys.
args=(--core)
[[ "${include_ai}" == true ]] && args=(--ai)
"${SCRIPT_DIR}/../smoke-test.sh" "${args[@]}"

if command -v curl >/dev/null 2>&1; then
  web_url="${WEB_APP_URL:-http://localhost:3000}"
  curl --fail --silent --show-error --max-time 5 "${web_url%/}/" >/dev/null
  echo "PASS Web App reachable"
fi
