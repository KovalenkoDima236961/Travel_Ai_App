#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
AUDIT_MODE=false
images=()

usage() {
  cat <<'USAGE'
Usage: scripts/security/trivy.sh [--audit] [--image IMAGE]...

Scans the repository filesystem for vulnerabilities, secrets, and
misconfiguration. Optionally scans already-built Docker images. High and
critical findings fail unless --audit is supplied.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --audit) AUDIT_MODE=true; shift ;;
    --image) [[ $# -ge 2 ]] || { echo "--image needs an image name" >&2; exit 2; }; images+=("$2"); shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

TRIVY_BIN="${TRIVY_BIN:-trivy}"
if ! command -v "${TRIVY_BIN}" >/dev/null 2>&1; then
  echo "trivy is required. Install it from https://trivy.dev/ before running this scan." >&2
  exit 127
fi

exit_code=1
[[ "${AUDIT_MODE}" == true ]] && exit_code=0
common_args=(--severity HIGH,CRITICAL --exit-code "${exit_code}" --scanners vuln,secret,misconfig --skip-dirs .git --skip-dirs .cache --skip-dirs graphify-out --skip-dirs node_modules --skip-dirs .next)

echo "==> trivy filesystem scan"
"${TRIVY_BIN}" fs "${common_args[@]}" "${PROJECT_ROOT}"

for image in "${images[@]}"; do
  echo "==> trivy image scan: ${image}"
  "${TRIVY_BIN}" image --severity HIGH,CRITICAL --exit-code "${exit_code}" "${image}"
done
