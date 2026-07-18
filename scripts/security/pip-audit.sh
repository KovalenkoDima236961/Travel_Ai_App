#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
AUDIT_MODE=false

usage() {
  cat <<'USAGE'
Usage: scripts/security/pip-audit.sh [--audit]

Audits runtime and development Python dependencies. pip-audit does not expose
per-advisory severity filtering, so the CI gate fails for every actionable
advisory. Use PIP_AUDIT_IGNORE=PYSEC-... only for an entry documented in
docs/security/accepted-risks.md. --audit is non-blocking.
USAGE
}

for arg in "$@"; do
  case "${arg}" in
    --audit) AUDIT_MODE=true ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown option: ${arg}" >&2; usage >&2; exit 2 ;;
  esac
done

PIP_AUDIT_BIN="${PIP_AUDIT_BIN:-pip-audit}"
if ! command -v "${PIP_AUDIT_BIN}" >/dev/null 2>&1; then
  echo "pip-audit is required. Install the AI development dependencies first." >&2
  exit 127
fi

# Keep the tool cache inside the repository's ignored development cache instead
# of writing into a developer's home directory. This also works in sandboxed CI.
PIP_AUDIT_CACHE_DIR="${PIP_AUDIT_CACHE_DIR:-${PROJECT_ROOT}/.cache/pip-audit}"
mkdir -p "${PIP_AUDIT_CACHE_DIR}"

ignore_args=()
if [[ -n "${PIP_AUDIT_IGNORE:-}" ]]; then
  IFS=',' read -r -a ignored_ids <<< "${PIP_AUDIT_IGNORE}"
  for advisory in "${ignored_ids[@]}"; do
    advisory="${advisory//[[:space:]]/}"
    [[ -n "${advisory}" ]] && ignore_args+=(--ignore-vuln "${advisory}")
  done
fi

run_audit() {
  local requirements_file="$1"
  echo "==> pip-audit: ${requirements_file}"
  local status=0
  if [[ "${AUDIT_MODE}" == true ]]; then
    if (( ${#ignore_args[@]} == 0 )); then
      "${PIP_AUDIT_BIN}" --cache-dir "${PIP_AUDIT_CACHE_DIR}" -r "${requirements_file}" || status=$?
    else
      "${PIP_AUDIT_BIN}" --cache-dir "${PIP_AUDIT_CACHE_DIR}" -r "${requirements_file}" "${ignore_args[@]}" || status=$?
    fi
    if (( status != 0 )); then
      echo "WARN: pip-audit found an issue in ${requirements_file} (audit mode)." >&2
    fi
  else
    if (( ${#ignore_args[@]} == 0 )); then
      "${PIP_AUDIT_BIN}" --cache-dir "${PIP_AUDIT_CACHE_DIR}" -r "${requirements_file}"
    else
      "${PIP_AUDIT_BIN}" --cache-dir "${PIP_AUDIT_CACHE_DIR}" -r "${requirements_file}" "${ignore_args[@]}"
    fi
  fi
}

cd "${PROJECT_ROOT}/services/ai-planning-service"
run_audit requirements.txt
run_audit requirements-dev.txt
