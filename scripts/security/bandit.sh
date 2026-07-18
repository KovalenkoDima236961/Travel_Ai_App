#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
AUDIT_MODE=false

usage() {
  cat <<'USAGE'
Usage: scripts/security/bandit.sh [--audit]

Runs Bandit on the AI Planning Service. The gate includes high-severity and
high-confidence findings; --audit reports without failing.
USAGE
}

for arg in "$@"; do
  case "${arg}" in
    --audit) AUDIT_MODE=true ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown option: ${arg}" >&2; usage >&2; exit 2 ;;
  esac
done

BANDIT_BIN="${BANDIT_BIN:-bandit}"
if ! command -v "${BANDIT_BIN}" >/dev/null 2>&1; then
  echo "bandit is required. Install the AI development dependencies first." >&2
  exit 127
fi

args=(-r app -c "${PROJECT_ROOT}/.bandit.yaml" -lll -iii)
if [[ "${AUDIT_MODE}" == true ]]; then
  args+=(--exit-zero)
fi

cd "${PROJECT_ROOT}/services/ai-planning-service"
"${BANDIT_BIN}" "${args[@]}"
