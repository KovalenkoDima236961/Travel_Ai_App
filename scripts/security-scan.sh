#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
AUDIT_MODE=false
RUN_ZAP=false

usage() {
  cat <<'USAGE'
Usage: scripts/security-scan.sh [--audit] [--zap]

Runs the local Security Hardening v1 checks. The standard mode is intended for
CI and fails on the tool thresholds documented in docs/security/tools.md.
--audit is a non-blocking triage mode. --zap additionally runs an unauthenticated
baseline scan against the running local Web App.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --audit) AUDIT_MODE=true; shift ;;
    --zap) RUN_ZAP=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done

mode_args=()
[[ "${AUDIT_MODE}" == true ]] && mode_args=(--audit)

"${SCRIPT_DIR}/security/gitleaks.sh" "${mode_args[@]}"
"${SCRIPT_DIR}/security/gosec.sh" "${mode_args[@]}"
"${SCRIPT_DIR}/security/govulncheck.sh" "${mode_args[@]}"
"${SCRIPT_DIR}/security/bandit.sh" "${mode_args[@]}"
"${SCRIPT_DIR}/security/pip-audit.sh" "${mode_args[@]}"
"${SCRIPT_DIR}/security/npm-audit.sh" "${mode_args[@]}"
"${SCRIPT_DIR}/security/trivy.sh" "${mode_args[@]}"
"${SCRIPT_DIR}/security/semgrep.sh" "${mode_args[@]}"

if [[ "${RUN_ZAP}" == true ]]; then
  "${SCRIPT_DIR}/security/zap-baseline.sh" "${mode_args[@]}"
fi
