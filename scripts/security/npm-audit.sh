#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
AUDIT_MODE=false

usage() {
  cat <<'USAGE'
Usage: scripts/security/npm-audit.sh [--audit]

Audits the Web App lockfile with its configured package manager. High and
critical advisories fail the normal gate. --audit prints them without failing.
USAGE
}

for arg in "$@"; do
  case "${arg}" in
    --audit) AUDIT_MODE=true ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown option: ${arg}" >&2; usage >&2; exit 2 ;;
  esac
done

cd "${PROJECT_ROOT}/apps/web"
NPM_AUDIT_CACHE_DIR="${NPM_AUDIT_CACHE_DIR:-${PROJECT_ROOT}/.cache/npm-audit}"
mkdir -p "${NPM_AUDIT_CACHE_DIR}"
if [[ -f package-lock.json ]]; then
  command=(npm audit --audit-level=high)
elif [[ -f pnpm-lock.yaml ]]; then
  command=(pnpm audit --audit-level=high)
elif [[ -f yarn.lock ]]; then
  command=(yarn npm audit --severity=high)
else
  echo "No supported frontend lockfile found." >&2
  exit 1
fi

echo "==> ${command[*]}"
if [[ "${AUDIT_MODE}" == true ]]; then
  if ! npm_config_cache="${NPM_AUDIT_CACHE_DIR}" "${command[@]}"; then
    echo "WARN: frontend dependency audit found an issue (audit mode)." >&2
  fi
else
  npm_config_cache="${NPM_AUDIT_CACHE_DIR}" "${command[@]}"
fi
