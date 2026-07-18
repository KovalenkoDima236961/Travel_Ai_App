#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
AUDIT_MODE=false

usage() {
  cat <<'USAGE'
Usage: scripts/security/govulncheck.sh [--audit]

Runs govulncheck for each Go module. The normal mode is a CI gate; --audit
prints all results but does not fail so findings can be triaged locally.
USAGE
}

for arg in "$@"; do
  case "${arg}" in
    --audit) AUDIT_MODE=true ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown option: ${arg}" >&2; usage >&2; exit 2 ;;
  esac
done

GOVULNCHECK_BIN="${GOVULNCHECK_BIN:-govulncheck}"
if ! command -v "${GOVULNCHECK_BIN}" >/dev/null 2>&1; then
  echo "govulncheck is required. Install it with: go install golang.org/x/vuln/cmd/govulncheck@latest" >&2
  exit 127
fi

services=(
  services/auth-service
  services/user-service
  services/trip-service
  services/notification-service
  services/external-integrations-service
  services/worker-service
)

for service in "${services[@]}"; do
  echo "==> govulncheck: ${service}"
  if [[ "${AUDIT_MODE}" == true ]]; then
    if ! (cd "${PROJECT_ROOT}/${service}" && "${GOVULNCHECK_BIN}" ./...); then
      echo "WARN: govulncheck found an issue in ${service} (audit mode)." >&2
    fi
  else
    (cd "${PROJECT_ROOT}/${service}" && "${GOVULNCHECK_BIN}" ./...)
  fi
done
