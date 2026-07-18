#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
AUDIT_MODE=false

usage() {
  cat <<'USAGE'
Usage: scripts/security/semgrep.sh [--audit]

Runs the community security-audit and OWASP Top Ten rulesets. CI blocks ERROR
severity findings; --audit reports all configured findings without failing.
USAGE
}

for arg in "$@"; do
  case "${arg}" in
    --audit) AUDIT_MODE=true ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown option: ${arg}" >&2; usage >&2; exit 2 ;;
  esac
done

SEMGREP_BIN="${SEMGREP_BIN:-semgrep}"
if ! command -v "${SEMGREP_BIN}" >/dev/null 2>&1; then
  echo "semgrep is required. Install it with: python -m pip install semgrep" >&2
  exit 127
fi

args=(scan --config p/security-audit --config p/owasp-top-ten --severity ERROR)
if [[ "${AUDIT_MODE}" != true ]]; then
  args+=(--error)
fi
"${SEMGREP_BIN}" "${args[@]}" "${PROJECT_ROOT}"
