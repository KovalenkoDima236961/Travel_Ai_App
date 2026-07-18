#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
AUDIT_MODE=false

usage() {
  cat <<'USAGE'
Usage: scripts/security/gosec.sh [--audit]

Scans every Go service with gosec. The default gate reports high-severity,
medium-or-higher-confidence findings and exits non-zero. --audit keeps the
report useful locally without failing the shell.
USAGE
}

for arg in "$@"; do
  case "${arg}" in
    --audit) AUDIT_MODE=true ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown option: ${arg}" >&2; usage >&2; exit 2 ;;
  esac
done

GOSEC_BIN="${GOSEC_BIN:-gosec}"
if ! command -v "${GOSEC_BIN}" >/dev/null 2>&1; then
  echo "gosec is required. Install it with: go install github.com/securego/gosec/v2/cmd/gosec@latest" >&2
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
  echo "==> gosec: ${service}"
  args=(-severity high -confidence medium)
  if [[ "${AUDIT_MODE}" == true ]]; then
    args+=(-no-fail)
  fi
  (
    cd "${PROJECT_ROOT}/${service}"
    "${GOSEC_BIN}" "${args[@]}" ./...
  )
done
