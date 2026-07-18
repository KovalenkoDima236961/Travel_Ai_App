#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
TARGET="${ZAP_TARGET:-http://host.docker.internal:3000}"
AUDIT_MODE=false

usage() {
  cat <<'USAGE'
Usage: scripts/security/zap-baseline.sh [--audit] [--target URL]

Runs the unauthenticated OWASP ZAP baseline scan against a running local Web
App. Start Compose first. On Linux, set ZAP_TARGET to a host-reachable URL.
The v1 baseline intentionally does not authenticate; authenticated scans are a
future scripted flow.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --audit) AUDIT_MODE=true; shift ;;
    --target) [[ $# -ge 2 ]] || { echo "--target needs a URL" >&2; exit 2; }; TARGET="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required for the ZAP baseline scan." >&2
  exit 127
fi

report_dir="${PROJECT_ROOT}/reports/security/zap"
mkdir -p "${report_dir}"
zap_args=(-t "${TARGET}" -r zap-baseline.html -J zap-baseline.json -m 2)
[[ "${AUDIT_MODE}" == true ]] && zap_args+=(-I)

echo "==> ZAP baseline: ${TARGET}"
docker run --rm -t -v "${report_dir}:/zap/wrk/:rw" owasp/zap2docker-stable zap-baseline.py "${zap_args[@]}"
