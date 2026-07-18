#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
AUDIT_MODE=false

usage() {
  cat <<'USAGE'
Usage: scripts/security/gitleaks.sh [--audit]

Scans the working tree for secrets with gitleaks. The normal gate fails on any
unreviewed secret. A reviewed historical false positive belongs in
.gitleaks.toml and docs/security/accepted-risks.md, never in an ad-hoc ignore.
USAGE
}

for arg in "$@"; do
  case "${arg}" in
    --audit) AUDIT_MODE=true ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown option: ${arg}" >&2; usage >&2; exit 2 ;;
  esac
done

GITLEAKS_BIN="${GITLEAKS_BIN:-gitleaks}"
if ! command -v "${GITLEAKS_BIN}" >/dev/null 2>&1; then
  echo "gitleaks is required. Install it from https://github.com/gitleaks/gitleaks." >&2
  exit 127
fi

args=(dir "${PROJECT_ROOT}" --config "${PROJECT_ROOT}/.gitleaks.toml" --redact --no-banner)
if [[ "${AUDIT_MODE}" == true ]]; then
  args+=(--exit-code 0)
fi
"${GITLEAKS_BIN}" "${args[@]}"
