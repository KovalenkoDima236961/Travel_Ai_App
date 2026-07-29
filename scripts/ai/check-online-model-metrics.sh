#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/ai/check-online-model-metrics.sh --deployment-id UUID
USAGE
}

TRIP_SERVICE_URL="${TRIP_SERVICE_URL:-http://localhost:8080}"
DEPLOYMENT_ID=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --deployment-id) DEPLOYMENT_ID="${2:-}"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage; exit 2 ;;
  esac
done

if [[ -z "${DEPLOYMENT_ID}" ]]; then
  usage >&2
  exit 2
fi

args=(-fsS)
if [[ -n "${OPS_AUTH_TOKEN:-}" ]]; then
  args+=(-H "Authorization: Bearer ${OPS_AUTH_TOKEN}")
fi

curl "${args[@]}" "${TRIP_SERVICE_URL%/}/ops/ai/model-deployments/${DEPLOYMENT_ID}/online-summary"
